package parse

import (
	"strings"
	"sync"
)

func ParseSequences(sequences chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for sequence := range sequences {
		// fmt.Println(sequence)
		// parse
		strings.ToLower(sequence)
	}
}
