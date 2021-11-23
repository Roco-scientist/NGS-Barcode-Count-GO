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
	// Format_regex holds the regex which includes the search groups for the barcodes
	Format_regex regexp.Regexp
	// Format_string is a string of the sequence format where the barcodes are replaced with Ns.  This is used for error corrections
	Format_string string
	// Constant_size is how many nucleotides are not barcodes in order to calculate the amount of allowed errors within the constant region.
	// Defaulted to 20% max
	Constant_size int
}

// AddSearchRegex method uses the format scheme within the format file to create the Format_regex, Format_string, and Constant_size.
func (f *SequenceFormat) AddSearchRegex(format_file_path string) {
	// format_text contains all text from the format_file that is from a line not preceded by '#'
	var format_text string
	file, err := os.Open(format_file_path)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			format_text += line
		}
	}

	// digit_search is used to find digits within any bracket style from the format scheme
	digit_search := regexp.MustCompile(`\d+`)
	// barcode_search finds different format types, ie barcode or constant region, in order to iterate over each
	barcode_search := regexp.MustCompile(`(?i)(\{\d+\})|(\[\d+\])|(\(\d+\))|N+|[ATGC]+`)
	// counted_barcode_num is increased after each barcode is found.  This is used for capture group name
	counted_barcode_num := 0
	// regex_string is built with capture groups then used for the regex object
	var regex_string string
	// iterates through each barcode_search group and create the regex string
	for _, group := range barcode_search.FindAllString(format_text, -1) {
		// group_name is the capture group name for the regex object
		var group_name string
		// if the group contains any of the bracket styles that indicate a barcode,
		// save the group_name then create the named capture group
		if strings.Contains(group, "[") {
			group_name = "sample"
		} else if strings.Contains(group, "{") {
			counted_barcode_num++
			group_name = fmt.Sprintf("counted_%v", counted_barcode_num)
		} else if strings.Contains(group, "(") {
			group_name = "random"
		}

		if len(group_name) != 0 {
			digits_string := digit_search.FindString(group)
			digits, _ := strconv.Atoi(digits_string)
			// Add as many Ns as there are nucleotides within the barcocde to the format string
			for i := 0; i < digits; i++ {
				f.Format_string += "N"
			}
			// Create the named capture group
			regex_string += fmt.Sprintf("(?P<%v>[ATGCN]{%v})", group_name, digits_string)
		} else if strings.Contains(group, "N") {
			// If there are Ns within the format scheme, add these as any nucleotide within the search
			regex_string += fmt.Sprintf("[ATGCN]{%v}", len(group))
			f.Format_string += group
			f.Constant_size += len(group)
		} else {
			// If there are not any barcodes nor Ns, it should be the constant region.
			regex_string += group
			f.Format_string += group
		}

	}
	f.Format_regex = *regexp.MustCompile(regex_string)
}

// Print outputs to stdout a string which represents the sequencing read format with barcodes replaced by Ns
func (f *SequenceFormat) Print() {
	fmt.Println("-FORMAT-")
	fmt.Println(f.Format_string)
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
func NewSampleBarcodes(sample_file_path string) SampleBarcodes {
	file, err := os.Open(sample_file_path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var sample_barcodes SampleBarcodes

	scanner := bufio.NewScanner(file)
	scanner.Scan() // remove the header
	sample_barcodes.Conversion = make(map[string]string)
	for scanner.Scan() {
		row := strings.Split(scanner.Text(), ",")
		sample_barcodes.Conversion[row[0]] = row[1]
		sample_barcodes.Barcodes = append(sample_barcodes.Barcodes, row[0])
	}
	return sample_barcodes
}

// CountedBarcodes contains counted barcode information
type CountedBarcodes struct {
	// Conversion is a slice of maps where each sequential counted barcode within the same read has its own map.
	// The key for the maps are the DNA sequence for the counted barcodes.  The values are the corresponding IDs
	Conversion []map[string]string
	// Barcodes is a slice of slices for each sequential counted barcode.  This is used for sequencing error correction
	Barcodes [][]string
	// Num_barcodes is how many counted barcodes are within each sequencing read.
	Num_barcodes int
}

// NewCountedBarcodes creates a CountedBarcodes struct with the information within the counted barcodes file
func NewCountedBarcodes(counted_bc_file_path string) CountedBarcodes {
	file, err := os.Open(counted_bc_file_path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var counted_barcodes CountedBarcodes

	scanner := bufio.NewScanner(file)
	scanner.Scan() // remove the header
	var barcode_nums []int
	var rows []string
	// First iterate through all lines and hold the information.  This is done so that the total number of
	// counted barcodes can be retrieved and all data can be placed within slices with the index being the
	// the sequential number of the counted barcode
	for scanner.Scan() {
		row_split := strings.Split(scanner.Text(), ",")
		barcode_num, _ := strconv.Atoi(row_split[2])
		barcode_nums = append(barcode_nums, barcode_num)
		rows = append(rows, row_split[0]+","+row_split[1]+","+row_split[2])
	}
	counted_barcodes.Num_barcodes = max(barcode_nums)

	// After knowing how many total counted barcodes per sequencing read, create slices of maps and string slices
	// where the index is the counted barcode number
	for i := 0; i < counted_barcodes.Num_barcodes; i++ {
		counted_barcodes.Conversion = append(counted_barcodes.Conversion, make(map[string]string))
		counted_barcodes.Barcodes = append(counted_barcodes.Barcodes, make([]string, 0))
	}

	// Insert all data from the counted barcode file gather previously
	for _, row := range rows {
		row_split := strings.Split(row, ",")
		barcode_num, _ := strconv.Atoi(row_split[2])
		insert_num := barcode_num - 1
		counted_barcodes.Conversion[insert_num][row_split[0]] = row_split[1]
		counted_barcodes.Barcodes[insert_num] = append(counted_barcodes.Barcodes[insert_num], row_split[0])
	}
	return counted_barcodes
}

// Max finds the maximum in within a slice of integers
func max(int_slice []int) int {
	max_int := -10000000
	for _, integer := range int_slice {
		if integer > max_int {
			max_int = integer
		}
	}
	return max_int
}

// ReadFastq reads the fastq file line by line and posts the sequence to the sequences string channel.
// This channel is then read by other parsing threads to parse the sequence
func ReadFastq(fastq_path string, sequences chan string, wg *sync.WaitGroup) int {
	defer close(sequences)
	defer wg.Done()
	file, err := os.Open(fastq_path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	total_reads := 0
	line_num := 0
	if strings.HasSuffix(fastq_path, "gz") {
		rawContents, err := gzip.NewReader(file)
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(rawContents)
		for scanner.Scan() {
			line_num++
			switch line_num {
			case 2:
				total_reads++
				for len(sequences) > 10000 {
				}
				sequences <- scanner.Text()
				if total_reads%10000 == 0 {
					fmt.Printf("\rTotal reads:                 %v", total_reads)
				}
			case 4:
				line_num = 0
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	} else if strings.HasSuffix(fastq_path, "fastq") {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line_num++
			switch line_num {
			case 2:
				total_reads++
				for len(sequences) > 10000 {
				}
				sequences <- scanner.Text()
				if total_reads%10000 == 0 {
					fmt.Printf("\rTotal reads:                 %v", total_reads)
				}
			case 4:
				line_num = 0
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}else{
		log.Fatal("fastq file must end with 'gz' or 'fastq'")
	}

	fmt.Printf("\rTotal reads:                 %v\n", total_reads)
	return total_reads
}
