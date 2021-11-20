package parse

import (
	"strings"
	"sync"

	"github.com/Roco-scientist/barcode-count-go/internal/input"
	"github.com/Roco-scientist/barcode-count-go/internal/results"
)

func ParseSequences(sequences chan string, wg *sync.WaitGroup, counts *results.Counts, format input.SequenceFormat, sample_barcodes []string) {
	defer wg.Done()
	sample_barcodes_check := make(map[string]struct{})
	for _, sample_barcode := range sample_barcodes {
		sample_barcodes_check[sample_barcode] = struct{}{}
	}
	for sequence := range sequences {
		if !format.Format_regex.MatchString(sequence) {
		}
		sequence_match := format.Format_regex.FindStringSubmatch(sequence)
		if sequence_match != nil {
			var sample_barcode, random_barcode, counted_barcodes string
			for i, name := range format.Format_regex.SubexpNames() {
				switch {
				case name == "sample":
					sample_barcode = sequence_match[i]
					if _, ok := sample_barcodes_check[sample_barcode]; !ok {
						sample_barcode = fix_sequence(sample_barcode, sample_barcodes, len(sample_barcode)/5)
					}
					if sample_barcode == "" {
						break
					}
				case name == "random":
					random_barcode = sequence_match[i]
				case strings.Contains(name, "counted"):
					if len(counted_barcodes) != 0 {
						counted_barcodes += ","
					}
					counted_barcodes += sequence_match[i]
				}
			}
			if sample_barcode != "" {
				counts.AddCount(sample_barcode, counted_barcodes, random_barcode)
			}
		}
	}
}

func fix_sequence(query_sequence string, subject_sequences []string, max_errors int) {
	best_mismatches := max_errors + 1
	var best_match string
	var mismatches int

	for _, subject_sequence := range subject_sequences {
		mismatches = 0
		for i := 0; i < len(query_sequence); i++ {
			if (query_sequence[i] != 'N' && subject_sequence[i] != 'N') && query_sequence[i] != subject_sequence[i] {
				mismatches++
			}
			if mismatches > best_mismatches {
				break
			}
		}
		if mismatches == best_mismatches {
			best_match = ""
		}
		if mismatches < best_mismatches {
			best_mismatches = mismatches
			best_match = subject_sequence
		}
	}
	return best_match
}
