package main

import (
	"fmt"
	"github.com/Roco-scientist/barcode-count-go/internal/input"
	"os/user"
)

func main() {
	usr, _ := user.Current()
	home := usr.HomeDir
	fastq_path := home + "/test_del/test.10000.fastq"
	var sequences []string
	total_count := input.ReadFastq(fastq_path, sequences)
	fmt.Printf("Total reads: %v", total_count)
}
