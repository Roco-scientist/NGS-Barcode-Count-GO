package results

import "sync"

type Counts struct {
	mu sync.Mutex
	no_random map[string]map[string]int
	random    map[string]map[string]map[string]bool
}

func (c *Counts) AddCount(sample_barcode string, counted_barcodes string, random_barcode string) {
	if len(random_barcode) == 0 {
		c.mu.Lock()
		c.no_random[sample_barcode][counted_barcodes]++
		c.mu.Unlock()
	} else {
		c.mu.Lock()
		c.random[sample_barcode][counted_barcodes][random_barcode] = true
		c.mu.Unlock()
	}
}
