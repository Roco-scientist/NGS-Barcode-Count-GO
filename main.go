package main

import (
	"fmt"
	"runtime"
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
	runtime.GOMAXPROCS(args.Threads)
	// start is used to measure the compute and total time of the algorithm
	start := time.Now()

	// wg is passed to each thread to make sure to wait for completion
	var wg sync.WaitGroup

	// formatInfo contains all information for barcode and sequencing format.  This is
	// used for regex searches and general information
	var formatInfo input.SequenceFormat
	formatInfo.AddSearchRegex(args.FormatPath)
	formatInfo.Print()

	// sampleBarcodes contains conversion information for the sample barcodes  This is used in all parsing
	// threads for sequencing error correction and while writing to csv to convert for the final file
	sampleBarcodes := input.NewSampleBarcodes(args.SampleBarcodesPath)

	// countedBarcodes contains conversion information for the counted barcodes.  This is used in all parsing
	// threads for sequencing error correction and while writing to csv to convert for the final file
	countedBarcodes := input.NewCountedBarcodes(args.CountedBarcodesPath, formatInfo.CountedBarcodeNum)

	// counts is the struct that is used to keep track of how many matches
	counts := results.NewCount(sampleBarcodes.Barcodes)

	// seqErrors keeps track of all of the sequencing errors within the sequencing reads
	var seqErrors results.ParseErrors

	// maxErrors contains how many sequecing errors are allowed per barcode.  This defaults to 
	// 20% of the lenght of any of the barcodes, but changes if any of the --max-errors flags are called
	maxErrors := results.NewMaxErrors(args.SampleErrors, args.BarcodesErrors, args.ConstantErrors, formatInfo)
	maxErrors.Print()

	// sequences is the channel for which the reading thread post sequences, and the parsing threads pull sequences
	sequences := make(chan string)

	// reader thread
	wg.Add(1)
	go input.ReadFastq(args.FastqPath, sequences, &wg)

	// parsing threads.  Using 3x the number of threads as using 1x tended to underutilize the cores.  With GO thread scheduler
	// this should be safe as long as GOMAXPROCS is set
	for i := 1; i < (args.Threads * 3); i++ {
		wg.Add(1)
		go parse.ParseSequences(sequences, &wg, counts, formatInfo, sampleBarcodes, countedBarcodes, &seqErrors, maxErrors)
	}

	// wait for all threads to finish
	wg.Wait()
	seqErrors.Print()

	compTime := elapsedTime(start)
	fmt.Printf("Compute time: %v\n\n", compTime)

	fmt.Println("-WRITING COUNTS-")
	enrich := false
	counts.WriteCsv(args.OutputDir, args.MergeOutput, enrich, countedBarcodes, sampleBarcodes)

	totTime := elapsedTime(start)
	fmt.Printf("Total time: %v\n", totTime)
}

// elapsedTime returns the time elapsed as a string in the format '# hours # minutes #.### seconds'
func elapsedTime(startTime time.Time) string {
	endTime := time.Now()
	totalTime := endTime.Sub(startTime)
	millisecondsString := strconv.Itoa(int(totalTime.Milliseconds()) % 1000)

	// Make sure the milliseconds are zero padded to be 0.### seconds
	for len(millisecondsString) < 3 {
		millisecondsString = "0" + millisecondsString
	}
	var totalString string

	minutes := int(totalTime.Minutes()) % 60
	seconds := int(totalTime.Seconds()) % 60

	if totalTime.Hours() >= 1 {
		totalString += fmt.Sprintf("%v hours %v minutes ", int(totalTime.Hours()), minutes)
	} else if minutes >= 1 {
		totalString += fmt.Sprintf("%v minutes ", minutes)
	} else if seconds >= 1 {

	}
	totalString += fmt.Sprintf("%v.%v seconds", seconds, millisecondsString)

	return totalString
}
