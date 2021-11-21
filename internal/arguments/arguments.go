package arguments

import (
	"github.com/akamensky/argparse"
	"log"
	"os"
)

type Args struct {
	Fastq_path                string // fastq file path
	Format_path               string // format scheme file path
	Sample_barcodes_path      string // sample barcode file path.  Optional
	Counted_barcodes_path     string // building block barcode file path. Optional
	Output_dir                string // output directory.  Deafaults to './'
	Threads                   int    // Number of threads to use.  Defaults to number of threads on the machine
	Prefix                    string // Prefix string for the output files
	Merge_output              bool   // Whether or not to create an additional output file that merges all samples
	Barcodes_errors           int    // Optional input of how many errors are allowed in each building block barcode.  Defaults to 20% of the length
	Sample_errors             int    // Optional input of how many errors are allowed in each sample barcode.  Defaults to 20% of the length
	Constant_errors           int    // Optional input of how many errors are allowed in each constant region barcode.  Defaults to 20% of the length
	Min_average_quality_score float32
	Enrich                    bool
}

func GetArgs() Args {
	var args Args
	parser := argparse.NewParser("barcode-count", "Counts barcodes located in sequencing data")
	fastq_path := parser.String("f", "fastq", &argparse.Options{Required: true, Help: "FASTQ file unzipped"})
	format_path := parser.String("q", "sequence-format", &argparse.Options{Required: true, Help: "Sequence format file"})
	counted_path := parser.String("c", "counted-barcodes", &argparse.Options{Required: true, Help: "Counted barcodes file"})
	sample_path := parser.String("s", "sample-barcodes", &argparse.Options{Required: true, Help: "Sample barcodes file"})
	output_dir := parser.String("o", "output-dir", &argparse.Options{Default: "./", Help: "Directory to output the counts to"})
	err := parser.Parse(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	args.Fastq_path = *fastq_path
	args.Format_path = *format_path
	args.Counted_barcodes_path = *counted_path
	args.Sample_barcodes_path = *sample_path
	args.Output_dir = *output_dir
	return args
}
