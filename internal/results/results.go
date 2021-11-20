package results

import (
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
