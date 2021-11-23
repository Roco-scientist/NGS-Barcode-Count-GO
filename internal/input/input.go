package input

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// SequenceFormat holds values which are used to find the barcodes within each sequencing read.
type SequenceFormat struct {
	// FormatRegex holds the regex which includes the search groups for the barcodes
	FormatRegex regexp.Regexp
	// FormatString is a string of the sequence format where the barcodes are replaced with Ns.  This is used for error corrections
	FormatString string
	// ConstantSize is how many nucleotides are not barcodes in order to calculate the amount of allowed errors within the constant region.
	// Defaulted to 20% max
	ConstantSize int
}

// AddSearchRegex method uses the format scheme within the format file to create the FormatRegex, FormatString, and ConstantSize.
func (f *SequenceFormat) AddSearchRegex(formatFilePath string) {
	// formatText contains all text from the formatFile that is from a line not preceded by '#'
	var formatText string
	file, err := os.Open(formatFilePath)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			formatText += line
		}
	}

	// digitSearch is used to find digits within any bracket style from the format scheme
	digitSearch := regexp.MustCompile(`\d+`)
	// barcodeSearch finds different format types, ie barcode or constant region, in order to iterate over each
	barcodeSearch := regexp.MustCompile(`(?i)(\{\d+\})|(\[\d+\])|(\(\d+\))|N+|[ATGC]+`)
	// countedBarcodeNum is increased after each barcode is found.  This is used for capture group name
	countedBarcodeNum := 0
	// regexString is built with capture groups then used for the regex object
	var regexString string
	// iterates through each barcodeSearch group and create the regex string
	for _, group := range barcodeSearch.FindAllString(formatText, -1) {
		// groupName is the capture group name for the regex object
		var groupName string
		// if the group contains any of the bracket styles that indicate a barcode,
		// save the groupName then create the named capture group
		if strings.Contains(group, "[") {
			groupName = "sample"
		} else if strings.Contains(group, "{") {
			countedBarcodeNum++
			groupName = fmt.Sprintf("counted_%v", countedBarcodeNum)
		} else if strings.Contains(group, "(") {
			groupName = "random"
		}

		if len(groupName) != 0 {
			digitsString := digitSearch.FindString(group)
			digits, _ := strconv.Atoi(digitsString)
			// Add as many Ns as there are nucleotides within the barcocde to the format string
			for i := 0; i < digits; i++ {
				f.FormatString += "N"
			}
			// Create the named capture group
			regexString += fmt.Sprintf("(?P<%v>[ATGCN]{%v})", groupName, digitsString)
		} else if strings.Contains(group, "N") {
			// If there are Ns within the format scheme, add these as any nucleotide within the search
			regexString += fmt.Sprintf("[ATGCN]{%v}", len(group))
			f.FormatString += group
		} else {
			// If there are not any barcodes nor Ns, it should be the constant region.
			regexString += group
			f.FormatString += group
			f.ConstantSize += len(group)
		}

	}
	f.FormatRegex = *regexp.MustCompile(regexString)
}

// Print outputs to stdout a string which represents the sequencing read format with barcodes replaced by Ns
func (f *SequenceFormat) Print() {
	fmt.Println("-FORMAT-")
	fmt.Println(f.FormatString)
	fmt.Println()
}

// SampleBarcodes contains sample barcode information
type SampleBarcodes struct {
	// Conversion is a map where the key is the sample DNA barcode and the value is the sample id
	Conversion map[string]string
	// Barcodes is a slice of sample DNA barcodes
	Barcodes []string
}

// NewSampleBarcodes creates a new SampleBarcodes struct using the sample barcodes file
func NewSampleBarcodes(sampleFilePath string) SampleBarcodes {
	file, err := os.Open(sampleFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var sampleBarcodes SampleBarcodes

	scanner := bufio.NewScanner(file)
	scanner.Scan() // remove the header
	sampleBarcodes.Conversion = make(map[string]string)
	for scanner.Scan() {
		row := strings.Split(scanner.Text(), ",")
		sampleBarcodes.Conversion[row[0]] = row[1]
		sampleBarcodes.Barcodes = append(sampleBarcodes.Barcodes, row[0])
	}
	return sampleBarcodes
}

// CountedBarcodes contains counted barcode information
type CountedBarcodes struct {
	// Conversion is a slice of maps where each sequential counted barcode within the same read has its own map.
	// The key for the maps are the DNA sequence for the counted barcodes.  The values are the corresponding IDs
	Conversion []map[string]string
	// Barcodes is a slice of slices for each sequential counted barcode.  This is used for sequencing error correction
	Barcodes [][]string
	// NumBarcodes is how many counted barcodes are within each sequencing read.
	NumBarcodes int
}

// NewCountedBarcodes creates a CountedBarcodes struct with the information within the counted barcodes file
func NewCountedBarcodes(countedBcFilePath string) CountedBarcodes {
	file, err := os.Open(countedBcFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var countedBarcodes CountedBarcodes

	scanner := bufio.NewScanner(file)
	scanner.Scan() // remove the header
	var barcodeNums []int
	var rows []string
	// First iterate through all lines and hold the information.  This is done so that the total number of
	// counted barcodes can be retrieved and all data can be placed within slices with the index being the
	// the sequential number of the counted barcode
	for scanner.Scan() {
		rowSplit := strings.Split(scanner.Text(), ",")
		barcodeNum, _ := strconv.Atoi(rowSplit[2])
		barcodeNums = append(barcodeNums, barcodeNum)
		rows = append(rows, rowSplit[0]+","+rowSplit[1]+","+rowSplit[2])
	}
	countedBarcodes.NumBarcodes = max(barcodeNums)

	// After knowing how many total counted barcodes per sequencing read, create slices of maps and string slices
	// where the index is the counted barcode number
	for i := 0; i < countedBarcodes.NumBarcodes; i++ {
		countedBarcodes.Conversion = append(countedBarcodes.Conversion, make(map[string]string))
		countedBarcodes.Barcodes = append(countedBarcodes.Barcodes, make([]string, 0))
	}

	// Insert all data from the counted barcode file gather previously
	for _, row := range rows {
		rowSplit := strings.Split(row, ",")
		barcodeNum, _ := strconv.Atoi(rowSplit[2])
		insertNum := barcodeNum - 1
		countedBarcodes.Conversion[insertNum][rowSplit[0]] = rowSplit[1]
		countedBarcodes.Barcodes[insertNum] = append(countedBarcodes.Barcodes[insertNum], rowSplit[0])
	}
	return countedBarcodes
}

// Max finds the maximum in within a slice of integers
func max(intSlice []int) int {
	maxInt := -10000000
	for _, integer := range intSlice {
		if integer > maxInt {
			maxInt = integer
		}
	}
	return maxInt
}

// ReadFastq reads the fastq file line by line and posts the sequence to the sequences string channel.
// This channel is then read by other parsing threads to parse the sequence
func ReadFastq(fastqPath string, sequences chan string, wg *sync.WaitGroup) int {
	defer close(sequences)
	defer wg.Done()
	file, err := os.Open(fastqPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	totalReads := 0
	lineNum := 0
	if strings.HasSuffix(fastqPath, "gz") {
		rawContents, err := gzip.NewReader(file)
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(rawContents)
		for scanner.Scan() {
			lineNum++
			switch lineNum {
			case 2:
				totalReads++
				for len(sequences) > 10000 {
				}
				sequences <- scanner.Text()
				if totalReads%10000 == 0 {
					fmt.Printf("\rTotal reads:                 %v", totalReads)
				}
			case 4:
				lineNum = 0
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	} else if strings.HasSuffix(fastqPath, "fastq") {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lineNum++
			switch lineNum {
			case 2:
				totalReads++
				for len(sequences) > 10000 {
				}
				sequences <- scanner.Text()
				if totalReads%10000 == 0 {
					fmt.Printf("\rTotal reads:                 %v", totalReads)
				}
			case 4:
				lineNum = 0
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}else{
		log.Fatal("fastq file must end with 'gz' or 'fastq'")
	}

	fmt.Printf("\rTotal reads:                 %v\n", totalReads)
	return totalReads
}
