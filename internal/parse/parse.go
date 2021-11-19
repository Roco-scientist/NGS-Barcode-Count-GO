package parse

import (
	// "regexp"
	"fmt"
	// "strings"
	"sync"
	"github.com/Roco-scientist/barcode-count-go/internal/input"
	"github.com/Roco-scientist/barcode-count-go/internal/results"
)

func ParseSequences(sequences chan string, wg *sync.WaitGroup, counts *results.Counts, format input.SequenceFormat) {
	defer wg.Done()
	for sequence := range sequences {
		// fmt.Println(sequence)
		// parse
		if format.Format_regex.MatchString(sequence) {
			fmt.Println("Matched")
		}else{
		}
	}
}
