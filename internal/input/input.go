package input

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sync"
)

func ReadFastq(fastq_path string, sequences chan string, wg *sync.WaitGroup) int {
	defer close(sequences)
	defer wg.Done()
	file, err := os.Open(fastq_path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	total_reads := 0
	line_num := 0
	for scanner.Scan() {
		line_num++
		switch line_num {
		case 2:
			total_reads++
			for len(sequences) > 10000 {}
			sequences <- scanner.Text()
			if total_reads%10000 == 0 {
				fmt.Printf("\rTotal reads: %v", total_reads)
			}
		case 4:
			line_num = 0
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\rTotal reads: %v\n", total_reads)
	return total_reads
}
