package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Roco-scientist/barcode-count-go/internal/arguments"
	"github.com/Roco-scientist/barcode-count-go/internal/input"
	"github.com/Roco-scientist/barcode-count-go/internal/parse"
	"github.com/Roco-scientist/barcode-count-go/internal/results"
)

func main() {
	args := arguments.GetArgs()
	start := time.Now()

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

	comp_time := elapsedTime(start)
	fmt.Printf("Compute time: %v\n\n", comp_time)

	fmt.Println("-WRITING COUNTS-")
	enrich := false
	counts.WriteCsv(args.Output_dir, args.Merge_output, enrich, counted_barcodes, sample_barcodes)

	tot_time := elapsedTime(start)
	fmt.Printf("Total time: %v\n", tot_time)
}

func elapsedTime(startTime time.Time) string {
	endTime := time.Now()
	total_time := endTime.Sub(startTime)
	milliseconds_string := strconv.Itoa(int(total_time.Milliseconds()) % 1000)
	for len(milliseconds_string) < 3 {
		milliseconds_string = "0" + milliseconds_string
	}
	var total_string string

	minutes := int(total_time.Minutes()) % 60
	seconds := int(total_time.Seconds()) % 60

	if total_time.Hours() >= 1 {
		total_string += fmt.Sprintf("%v hours %v minutes ", int(total_time.Hours()), minutes)
	} else if minutes >= 1 {
		total_string += fmt.Sprintf("%v minutes ", minutes)
	} else if seconds >= 1 {

	}
	total_string += fmt.Sprintf("%v.%v seconds", seconds, milliseconds_string)

	return total_string
}
