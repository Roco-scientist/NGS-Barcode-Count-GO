package results

import (
	// "bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Roco-scientist/barcode-count-go/internal/input"
)

type Counts struct {
	mu        sync.Mutex
	No_random map[string]map[string]int
	Random    map[string]map[string]map[string]bool
}

func NewCount(sample_barcodes []string) *Counts {
	var count Counts
	count.No_random = make(map[string]map[string]int)
	count.Random = make(map[string]map[string]map[string]bool)
	for _, sample_barcode := range sample_barcodes {
		count.No_random[sample_barcode] = make(map[string]int)
		count.Random[sample_barcode] = make(map[string]map[string]bool)
	}
	return &count
}

func (c *Counts) AddCount(sample_barcode string, counted_barcodes string, random_barcode string) {
	if len(random_barcode) == 0 {
		c.mu.Lock()
		c.No_random[sample_barcode][counted_barcodes]++
		c.mu.Unlock()
	} else {
		c.mu.Lock()
		c.Random[sample_barcode][counted_barcodes][random_barcode] = true
		c.mu.Unlock()
	}
}

func (c *Counts) WriteCsv(outpath string, merge bool, enrich bool, counted_barcodes_struct input.CountedBarcodes, sample_barcodes input.SampleBarcodes) {
	today := time.Now().Local().Format("2006-01-02")
	var sample_ids []string
	for key := range sample_barcodes.Conversion {
		sample_ids = append(sample_ids, sample_barcodes.Conversion[key])
	}
	sort.Strings(sample_ids)

	var sample_barcodes_sorted []string

	for _, sample_id := range sample_ids {
		for key, value := range sample_barcodes.Conversion {
			if value == sample_id {
				sample_barcodes_sorted = append(sample_barcodes_sorted, key)
				break
			}
		}
	}
	var header_start string
	for i:= 0 ; i < counted_barcodes_struct.Num_barcodes ; i++ {
		header_start += "Barcode_" + strconv.Itoa(i+1) + ","
	}

	sample_header := header_start + "Count"
	for _, sample_barcode := range sample_barcodes_sorted {
		out_file_name := outpath + today + "_" + sample_barcodes.Conversion[sample_barcode] + ".counts.csv"
		sample_out := sample_header
		for counted_barcodes, count := range c.No_random[sample_barcode] {
			sample_out += "\n" + counted_barcodes + "," + strconv.Itoa(count)
		}
		file, err := os.Create(out_file_name)
		if err != nil {
			log.Fatal(err)
		}
		_, write_err := file.WriteString(sample_out)
		if write_err != nil {
			log.Fatal(err)
		}
	}
}

type ParseErrors struct {
	correct     int
	constant    int
	sample      int
	counted     int
	correct_mu  sync.Mutex
	constant_mu sync.Mutex
	sample_mu   sync.Mutex
	counted_mu  sync.Mutex
}

func (p *ParseErrors) AddCorrect() {
	p.correct_mu.Lock()
	p.correct++
	p.correct_mu.Unlock()
}

func (p *ParseErrors) AddConstantError() {
	p.constant_mu.Lock()
	p.constant++
	p.constant_mu.Unlock()
}

func (p *ParseErrors) AddSampleError() {
	p.sample_mu.Lock()
	p.sample++
	p.sample_mu.Unlock()
}

func (p *ParseErrors) AddCountedError() {
	p.counted_mu.Lock()
	p.counted++
	p.counted_mu.Unlock()
}

func (p *ParseErrors) Print() {
	fmt.Printf("Correctly matched sequences: %v\n"+
		"Constant region errrors:     %v\n"+
		"Sample barcode errors:       %v\n"+
		"Counted barcode errors:      %v\n",
		p.correct, p.constant, p.sample, p.counted)
}
