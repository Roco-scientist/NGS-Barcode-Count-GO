package input

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func ReadFastq(fastq_path string, sequences []string) int {
	file, err := os.Open(fastq_path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return 1
}
