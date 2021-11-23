package arguments

import (
	"github.com/akamensky/argparse"
	"log"
	"os"
	"runtime"
)

// Args holds all input argument information
type Args struct {
	FastqPath                string // fastq file path
	FormatPath               string // format scheme file path
	SampleBarcodesPath      string // sample barcode file path.  Optional
	CountedBarcodesPath     string // building block barcode file path. Optional
	OutputDir                string // output directory.  Deafaults to './'
	Threads                   int    // Number of threads to use.  Defaults to number of threads on the machine
	Prefix                    string // Prefix string for the output files
	MergeOutput              bool   // Whether or not to create an additional output file that merges all samples
	BarcodesErrors           int    // Optional input of how many errors are allowed in each building block barcode.  Defaults to 20% of the length
	SampleErrors             int    // Optional input of how many errors are allowed in each sample barcode.  Defaults to 20% of the length
	ConstantErrors           int    // Optional input of how many errors are allowed in each constant region barcode.  Defaults to 20% of the length
	MinAverageQualityScore float32
	Enrich                    bool
}

// GetArgs retrieves all arguments passed from the CLI
func GetArgs() Args {
	var args Args
	parser := argparse.NewParser("barcode-count-go", "Counts barcodes located in sequencing data")
	fastqPath := parser.String("f", "fastq", &argparse.Options{Required: true, Help: "FASTQ file unzipped"})
	formatPath := parser.String("q", "sequence-format", &argparse.Options{Required: true, Help: "Sequence format file"})
	countedPath := parser.String("c", "counted-barcodes", &argparse.Options{Required: true, Help: "Counted barcodes file"})
	samplePath := parser.String("s", "sample-barcodes", &argparse.Options{Required: true, Help: "Sample barcodes file"})
	outputDir := parser.String("o", "output-dir", &argparse.Options{Default: "./", Help: "Directory to output the counts to"})
	mergeOutput := parser.Flag("m", "merge-output", &argparse.Options{Help: "Merge sample output counts into a single file.  Not necessary when there is only one sample"})
	threads := parser.Int("t", "threads", &argparse.Options{Default: runtime.NumCPU(), Help: "Number of threads"})
	err := parser.Parse(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	args.FastqPath = *fastqPath
	args.FormatPath = *formatPath
	args.CountedBarcodesPath = *countedPath
	args.SampleBarcodesPath = *samplePath
	args.OutputDir = *outputDir
	args.MergeOutput = *mergeOutput
	args.Threads = *threads
	return args
}
