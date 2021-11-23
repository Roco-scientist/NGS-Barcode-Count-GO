package parse

import (
	"github.com/Roco-scientist/barcode-count-go/internal/input"
	"github.com/Roco-scientist/barcode-count-go/internal/results"
	"strings"
	"sync"
)

func ParseSequences(
	sequences chan string,
	wg *sync.WaitGroup,
	counts *results.Counts,
	format input.SequenceFormat,
	sampleBarcodes input.SampleBarcodes,
	countedBarcodesStruct input.CountedBarcodes,
	seqErrors *results.ParseErrors,
) {
	defer wg.Done()
	sampleBarcodesCheck := make(map[string]struct{})
	for _, sampleBarcode := range sampleBarcodes.Barcodes {
		sampleBarcodesCheck[sampleBarcode] = struct{}{}
	}
	for sequence := range sequences {
		if !format.FormatRegex.MatchString(sequence) {
			sequence = fixConstant(sequence, format.FormatString, format.ConstantSize/5)
		}
		sequenceMatch := format.FormatRegex.FindStringSubmatch(sequence)
		if sequenceMatch != nil {
			var sampleBarcode, randomBarcode, countedBarcodes, countedBarcode string
			countedBarcodeNum := 0
			sequenceFail := false
			for i, name := range format.FormatRegex.SubexpNames() {
				switch {
				case name == "sample":
					sampleBarcode = sequenceMatch[i]
					if _, ok := sampleBarcodesCheck[sampleBarcode]; !ok {
						sampleBarcode = fixSequence(sampleBarcode, sampleBarcodes.Barcodes, len(sampleBarcode)/5)
					}
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
					if _, ok := countedBarcodesStruct.Conversion[countedBarcodeNum][countedBarcode]; !ok {
						countedBarcode = fixSequence(countedBarcode, countedBarcodesStruct.Barcodes[countedBarcodeNum], len(countedBarcode)/5)
					}
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
			if !sequenceFail {
				counts.AddCount(sampleBarcode, countedBarcodes, randomBarcode)
				seqErrors.AddCorrect()
			}
		} else {
			seqErrors.AddConstantError()
		}
	}
}

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

func fixSequence(querySequence string, subjectSequences []string, maxErrors int) string {
	bestMismatches := maxErrors + 1
	var bestMatch string
	var mismatches int

	for _, subjectSequence := range subjectSequences {
		mismatches = 0
		for i := 0; i < len(querySequence); i++ {
			if  querySequence[i] != subjectSequence[i] && querySequence[i] != 'N' && subjectSequence[i] != 'N' {
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
