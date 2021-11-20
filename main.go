package main

import (
	"fmt"
	"github.com/Roco-scientist/barcode-count-go/internal/input"
	"github.com/Roco-scientist/barcode-count-go/internal/parse"
	"github.com/Roco-scientist/barcode-count-go/internal/results"
	"os/user"
	"sync"
)

func main() {
	threads := 8
	var wg sync.WaitGroup
	usr, _ := user.Current()
	home := usr.HomeDir
	fastq_path := home + "/test_del/test.4.fastq"
	format_path := home + "/test_del/test.scheme.txt"
	sample_file_path := home + "/test_del/sample_barcode_file.csv"

	sample_barcodes := input.NewSampleBarcodes(sample_file_path)

	counts := results.NewCount(sample_barcodes.Barcodes)

	var format_info input.SequenceFormat
	format_info.AddSearchRegex(format_path)

	sequences := make(chan string)
	wg.Add(1)
	go input.ReadFastq(fastq_path, sequences, &wg)
	for i := 1; i < threads; i++ {
		wg.Add(1)
		go parse.ParseSequences(sequences, &wg, counts, format_info)
	}
	wg.Wait()
	fmt.Println(counts.No_random)
}
