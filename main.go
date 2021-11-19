package main

import (
	"github.com/Roco-scientist/barcode-count-go/internal/input"
	"github.com/Roco-scientist/barcode-count-go/internal/parse"
	"os/user"
	"sync"
)

func main() {
	threads := 8
	var wg sync.WaitGroup
	usr, _ := user.Current()
	home := usr.HomeDir
	fastq_path := home + "/test_del/test.1000000.fastq"
	sequences := make(chan string)
	wg.Add(1)
	go input.ReadFastq(fastq_path, sequences, &wg)
	for i := 1; i < threads; i++ {
		wg.Add(1)
		go parse.ParseSequences(sequences, &wg)
	}
	wg.Wait()
}
