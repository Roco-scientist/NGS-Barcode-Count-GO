package parse

import (
	"github.com/Roco-scientist/barcode-count-go/internal/input"
	"github.com/Roco-scientist/barcode-count-go/internal/results"
	"strings"
	"sync"
)

// ParseSequences iterates over the sequences which are added to a channel by a reader thread,
// and then finds the barcodes within the sequence and, sequening errors are not above the threshold,
// will add the counted barcode to the results.  This is meant to be threadsafe, so it can be spawned
// multiple times to decrease computation time
func ParseSequences(
	// sequences is a channel which holds the sequences read by input.ReadFastq
	sequences chan string,
	wg *sync.WaitGroup,
	// counts is the struct which holds the counted results
	counts *results.Counts,
	// format is a struct which holds information essential for finding barcodes and do sequence error correction
	// This includes the regex used to search for barcodes
	format input.SequenceFormat,
	// sampleBarcodes holds barcode conversion to ID for samples
	sampleBarcodes input.SampleBarcodes,
	// countedBarcodesStruct holds barcode conversion to id for counted barcodes
	countedBarcodesStruct input.CountedBarcodes,
	// seqErrors is a struct which keeps track of the quantity of seequencing errors
	seqErrors *results.ParseErrors,
	// maxErrors holds the maximum sequencing errors allowed per barcode
	maxErrors results.MaxBarcodeErrorsAllowed,
) {
	defer wg.Done()
	// a map:struct is created to check whether or not a sampleBarcode exists.  This is used in place
	// of what would normally be a set.  Faster than checking the contents of a slice
	sampleBarcodesCheck := make(map[string]struct{})
	for _, sampleBarcode := range sampleBarcodes.Barcodes {
		sampleBarcodesCheck[sampleBarcode] = struct{}{}
	}

	for sequence := range sequences {
		// If the regex does not work on the sequence, there's a good chance there are sequencing sequencing
		// errors within the constant region.
		if !format.FormatRegex.MatchString(sequence) {
			sequence = fixConstant(sequence, format.FormatString, maxErrors.Constant)
		}

		sequenceMatch := format.FormatRegex.FindStringSubmatch(sequence)
		if sequenceMatch == nil {
			seqErrors.AddConstantError()
		} else {
			var sampleBarcode, randomBarcode, countedBarcodes, countedBarcode string
			// countedBarcodeNum holds the number of counted barcode that should be used as an index
			// for the current barcode iteration
			countedBarcodeNum := 0
			// sequenceFail is used to end the iteratoin early if any of the sequencing errors fail to get fixed
			sequenceFail := false
			for i, name := range format.FormatRegex.SubexpNames() {
				// the barcode name from the capture group exists as either sample, random, or counted_#
				switch {
				case name == "sample":
					sampleBarcode = sequenceMatch[i]
					if sampleBarcodes.Included {
						if _, ok := sampleBarcodesCheck[sampleBarcode]; !ok {
							sampleBarcode = fixSequence(sampleBarcode, sampleBarcodes.Barcodes, maxErrors.Sample)
						}
					}
					// If fixSequence does not find a best match, it returns an empty string
					if sampleBarcode == "" {
						seqErrors.AddSampleError()
						sequenceFail = true
					}
				case name == "random":
					randomBarcode = sequenceMatch[i]
				case strings.Contains(name, "counted"):
					if countedBarcodeNum != 0 {
						countedBarcodes += ","
					}
					countedBarcode = sequenceMatch[i]
					// When a counted barcodes file is not included hte conversion is not created
					if countedBarcodesStruct.Included {
						if _, ok := countedBarcodesStruct.Conversion[countedBarcodeNum][countedBarcode]; !ok {
							countedBarcode = fixSequence(countedBarcode, countedBarcodesStruct.Barcodes[countedBarcodeNum], maxErrors.Counted)
						}
					}
					// If fixSequence does not find a best match, it returns an empty string
					if countedBarcode == "" {
						seqErrors.AddCountedError()
						sequenceFail = true
					} else {
						countedBarcodes += countedBarcode
						countedBarcodeNum++
					}
				}
				if sequenceFail {
					break
				}
			}
			// If none of the error corrections failed and good matches were found, add the count
			if !sequenceFail {
				if inserted := counts.AddCount(sampleBarcode, countedBarcodes, randomBarcode, sampleBarcodes.Included); inserted {
					seqErrors.AddCorrect()
				} else {
					seqErrors.AddDuplicateError()
				}
			}
		}
	}
}

// fixConstant fixes the constant region of the sequence when the regex search does not match
func fixConstant(querySequence string, formatString string, maxErrors int) string {
	lengthDiff := len(querySequence) - len(formatString)
	var possibleSeqs []string
	for i := 0; i < lengthDiff; i++ {
		possibleSeqs = append(possibleSeqs, querySequence[i:i+len(formatString)])
	}
	bestSeqeunce := fixSequence(formatString, possibleSeqs, maxErrors)
	if bestSeqeunce != "" {
		fixedSequence := swapBarcodes(bestSeqeunce, formatString)
		return fixedSequence
	}
	return ""
}

// swapBarcodes creates a fixed sequence. It does this by using the formatString which
// contains the sequencing format where the barcodes are replaced by Ns.  This funciton
// swaps out the Ns for the barcodes found within the sequencing read.
func swapBarcodes(bestSeqeunce string, formatString string) string {
	var fixedSequence string
	for i := 0; i < len(formatString); i++ {
		if formatString[i] == 'N' {
			fixedSequence += string(bestSeqeunce[i])
		} else {
			fixedSequence += string(formatString[i])
		}
	}
	return fixedSequence
}

// fixSequence iterates through subjectSequences to find the best match to the querySequence.  maxErrors is 
// passed to this function to make sure that the match is close enough, ie, you would not want to call a match
// within which only have of the nucleotides match.  The default of NGS-Barcode-Count is 20% errors or len(querySequence)/5
func fixSequence(querySequence string, subjectSequences []string, maxErrors int) string {
	bestMismatches := maxErrors + 1
	var bestMatch string
	var mismatches int

	for _, subjectSequence := range subjectSequences {
		mismatches = 0
		for i := 0; i < len(querySequence); i++ {
			if querySequence[i] != subjectSequence[i] && querySequence[i] != 'N' && subjectSequence[i] != 'N' {
				mismatches++
			}
			if mismatches > bestMismatches {
				break
			}
		}
		if mismatches == bestMismatches {
			bestMatch = ""
		}
		if mismatches < bestMismatches {
			bestMismatches = mismatches
			bestMatch = subjectSequence
		}
	}
	return bestMatch
}
