package results

import (
	"fmt"
	"sync"
)

type Counts struct {
	mu sync.Mutex
	No_random map[string]map[string]int
	Random    map[string]map[string]map[string]bool
}

func NewCount(sample_barcodes []string) *Counts {
	var count Counts
	count.No_random = make(map[string]map[string]int)
	count.Random = make(map[string]map[string]map[string]bool)
	for _, sample_barcode := range sample_barcodes{
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

type ParseErrors struct {
	constant int
	sample int
	counted int
	constant_mu sync.Mutex
	sample_mu sync.Mutex
	counted_mu sync.Mutex
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
	fmt.Printf(
	"Constant region errrors: %v\n" +
	"Sample barcode errors:   %v\n" +
	"Counted barcode errors:  %v\n", p.constant, p.sample, p.counted)
}
