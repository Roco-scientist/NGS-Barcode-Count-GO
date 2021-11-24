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
	start := time.Now()

	var wg sync.WaitGroup

	sampleBarcodes := input.NewSampleBarcodes(args.SampleBarcodesPath)

	counts := results.NewCount(sampleBarcodes.Barcodes)

	var formatInfo input.SequenceFormat
	formatInfo.AddSearchRegex(args.FormatPath)
	formatInfo.Print()

	var seqErrors results.ParseErrors

	maxErrors := results.NewMaxErrors(args.SampleErrors, args.BarcodesErrors, args.ConstantErrors, formatInfo)
	maxErrors.Print()

	countedBarcodes := input.NewCountedBarcodes(args.CountedBarcodesPath, formatInfo.CountedBarcodeNum)

	sequences := make(chan string)
	wg.Add(1)
	go input.ReadFastq(args.FastqPath, sequences, &wg)
	for i := 1; i < (args.Threads * 3); i++ {
		wg.Add(1)
		go parse.ParseSequences(sequences, &wg, counts, formatInfo, sampleBarcodes, countedBarcodes, &seqErrors, maxErrors)
	}
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

func elapsedTime(startTime time.Time) string {
	endTime := time.Now()
	totalTime := endTime.Sub(startTime)
	millisecondsString := strconv.Itoa(int(totalTime.Milliseconds()) % 1000)
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
