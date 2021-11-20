package main

import (
	"fmt"
	"github.com/Roco-scientist/barcode-count-go/internal/input"
	"github.com/Roco-scientist/barcode-count-go/internal/parse"
	"github.com/Roco-scientist/barcode-count-go/internal/results"
	"os/user"
	"sync"
	"time"
)

func main() {
	defer un(trace("Total"))
	threads := 8
	var wg sync.WaitGroup
	usr, _ := user.Current()
	home := usr.HomeDir
	fastq_path := home + "/test_del/test.10000.double.fastq"
	format_path := home + "/test_del/test.scheme.txt"
	sample_file_path := home + "/test_del/sample_barcode_file.csv"
	counted_bc_path := home + "/test_del/counted_barcodes.csv"

	sample_barcodes := input.NewSampleBarcodes(sample_file_path)

	counts := results.NewCount(sample_barcodes.Barcodes)

	var format_info input.SequenceFormat
	format_info.AddSearchRegex(format_path)

	var seq_errors results.ParseErrors

	counted_barcodes := input.NewCountedBarcodes(counted_bc_path)

	sequences := make(chan string)
	wg.Add(1)
	go input.ReadFastq(fastq_path, sequences, &wg)
	for i := 1; i < threads; i++ {
		wg.Add(1)
		go parse.ParseSequences(sequences, &wg, counts, format_info, sample_barcodes, counted_barcodes, &seq_errors)
	}
	wg.Wait()
	seq_errors.Print()
}

func trace(s string) (string, time.Time) {
    // log.Println("START:", s)
    return s, time.Now()
}

func un(s string, startTime time.Time) {
    endTime := time.Now()
    fmt.Printf("%v time: %v\n", s, endTime.Sub(startTime))
}

