package main

import (
	"fmt"
	"github.com/Roco-scientist/barcode-count-go/internal/arguments"
	"github.com/Roco-scientist/barcode-count-go/internal/input"
	"github.com/Roco-scientist/barcode-count-go/internal/parse"
	"github.com/Roco-scientist/barcode-count-go/internal/results"
	"sync"
	"time"
)

func main() {
	args := arguments.GetArgs()
	defer un(trace("Total"))
	var wg sync.WaitGroup

	sample_barcodes := input.NewSampleBarcodes(args.Sample_barcodes_path)

	counts := results.NewCount(sample_barcodes.Barcodes)

	var format_info input.SequenceFormat
	format_info.AddSearchRegex(args.Format_path)
	format_info.Print()

	var seq_errors results.ParseErrors

	counted_barcodes := input.NewCountedBarcodes(args.Counted_barcodes_path)

	sequences := make(chan string)
	wg.Add(1)
	go input.ReadFastq(args.Fastq_path, sequences, &wg)
	for i := 1; i < args.Threads; i++ {
		wg.Add(1)
		go parse.ParseSequences(sequences, &wg, counts, format_info, sample_barcodes, counted_barcodes, &seq_errors)
	}
	wg.Wait()
	seq_errors.Print()
	fmt.Println()
	enrich := false
	fmt.Println("-WRITING COUNTS-")
	counts.WriteCsv(args.Output_dir, args.Merge_output, enrich, counted_barcodes, sample_barcodes)
}

func trace(s string) (string, time.Time) {
    // log.Println("START:", s)
    return s, time.Now()
}

func un(s string, startTime time.Time) {
    endTime := time.Now()
    fmt.Printf("%v time: %v\n", s, endTime.Sub(startTime))
}

