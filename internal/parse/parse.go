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
	sample_barcodes input.SampleBarcodes,
	counted_barcodes_struct input.CountedBarcodes,
	seq_errors *results.ParseErrors,
) {
	defer wg.Done()
	sample_barcodes_check := make(map[string]struct{})
	for _, sample_barcode := range sample_barcodes.Barcodes {
		sample_barcodes_check[sample_barcode] = struct{}{}
	}
	for sequence := range sequences {
		if !format.Format_regex.MatchString(sequence) {
			sequence = fix_constant(sequence, format.Format_string)
		}
		sequence_match := format.Format_regex.FindStringSubmatch(sequence)
		if sequence_match != nil {
			var sample_barcode, random_barcode, counted_barcodes, counted_barcode string
			counted_barcode_num := 0
			sequence_fail := false
			for i, name := range format.Format_regex.SubexpNames() {
				switch {
				case name == "sample":
					sample_barcode = sequence_match[i]
					if _, ok := sample_barcodes_check[sample_barcode]; !ok {
						sample_barcode = fix_sequence(sample_barcode, sample_barcodes.Barcodes, len(sample_barcode)/5)
					}
					if sample_barcode == "" {
						seq_errors.AddSampleError()
						sequence_fail = true
					}
				case name == "random":
					random_barcode = sequence_match[i]
				case strings.Contains(name, "counted"):
					if counted_barcode_num != 0 {
						counted_barcodes += ","
					}
					counted_barcode = sequence_match[i]
					if _, ok := counted_barcodes_struct.Conversion[counted_barcode_num][counted_barcode]; !ok {
						counted_barcode = fix_sequence(counted_barcode, counted_barcodes_struct.Barcodes[counted_barcode_num], len(counted_barcode)/5)
					}
					if counted_barcode == "" {
						seq_errors.AddCountedError()
						sequence_fail = true
					} else {
						counted_barcodes += counted_barcode
						counted_barcode_num++
					}
				}
				if sequence_fail {
					break
				}
			}
			if !sequence_fail {
				counts.AddCount(sample_barcode, counted_barcodes, random_barcode)
				seq_errors.AddCorrect()
			}
		} else {
			seq_errors.AddConstantError()
		}
	}
}

func fix_constant(query_sequence string, format_string string) string {
	length_diff := len(query_sequence) - len(format_string)
	var possible_seqs []string
	for i := 0; i < length_diff; i++ {
		possible_seqs = append(possible_seqs, query_sequence[i:i+len(format_string)])
	}
	best_seqeunce := fix_sequence(format_string, possible_seqs, len(format_string)/5)
	if best_seqeunce != "" {
		fixed_sequence := swap_barcodes(best_seqeunce, format_string)
		return fixed_sequence
	}
	return ""
}

func swap_barcodes(best_seqeunce string, format_string string) string {
	var fixed_sequence string
	for i := 0; i < len(format_string); i++ {
		if format_string[i] == 'N' {
			fixed_sequence += string(best_seqeunce[i])
		} else {
			fixed_sequence += string(format_string[i])
		}
	}
	return fixed_sequence
}

func fix_sequence(query_sequence string, subject_sequences []string, max_errors int) string {
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
