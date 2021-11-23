package results

import (
	// "bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Roco-scientist/barcode-count-go/internal/input"
)

const NoSampleName = "barcode_counts"

type Counts struct {
	mu       sync.Mutex
	NoRandom map[string]map[string]int
	Random   map[string]map[string]map[string]bool
}

func NewCount(sampleBarcodes []string) *Counts {
	var count Counts
	count.NoRandom = make(map[string]map[string]int)
	count.Random = make(map[string]map[string]map[string]bool)
	for _, sampleBarcode := range sampleBarcodes {
		count.NoRandom[sampleBarcode] = make(map[string]int)
		count.Random[sampleBarcode] = make(map[string]map[string]bool)
	}
	return &count
}

func (c *Counts) AddCount(sampleBarcode string, countedBarcodes string, randomBarcode string, samplBarcodeIncluded bool) {
	if !samplBarcodeIncluded {
		sampleBarcode = NoSampleName
	}
	if len(randomBarcode) == 0 {
		c.mu.Lock()
		c.NoRandom[sampleBarcode][countedBarcodes]++
		c.mu.Unlock()
	} else {
		c.mu.Lock()
		c.Random[sampleBarcode][countedBarcodes][randomBarcode] = true
		c.mu.Unlock()
	}
}

func (c *Counts) WriteCsv(outpath string, merge bool, enrich bool, countedBarcodesStruct input.CountedBarcodes, sampleBarcodes input.SampleBarcodes) {
	var sampleIds []string
	for key := range sampleBarcodes.Conversion {
		sampleIds = append(sampleIds, sampleBarcodes.Conversion[key])
	}
	sort.Strings(sampleIds)

	var sampleBarcodesSorted []string

	for _, sampleId := range sampleIds {
		for key, value := range sampleBarcodes.Conversion {
			if value == sampleId {
				sampleBarcodesSorted = append(sampleBarcodesSorted, key)
				break
			}
		}
	}
	var headerStart string
	for i := 0; i < countedBarcodesStruct.NumBarcodes; i++ {
		headerStart += "Barcode_" + strconv.Itoa(i+1) + ","
	}

	sampleHeader := headerStart + "Count"
	mergeHeader := headerStart
	var mergeOut strings.Builder
	countedBarcodesFinished := make(map[string]bool)
	if merge {
		for i, sampleId := range sampleIds {
			if i != 0 {
				mergeHeader += ","
			}
			mergeHeader += sampleId
		}
		mergeOut.WriteString(mergeHeader)
	}
	today := time.Now().Local().Format("2006-01-02")
	for _, sampleBarcode := range sampleBarcodesSorted {
		fmt.Printf("Gathering for %v\n", sampleBarcodes.Conversion[sampleBarcode])
		var sampleOut strings.Builder
		sampleOut.WriteString(sampleHeader)
		total := 0
		for countedBarcodes, count := range c.NoRandom[sampleBarcode] {
			total++
			var convertedBarcodes string
			if countedBarcodesStruct.Included {
				convertedBarcodes = convertCounted(countedBarcodes, countedBarcodesStruct)
			} else {
				convertedBarcodes = countedBarcodes
			}
			sampleOut.WriteString("\n" + convertedBarcodes + "," + strconv.Itoa(count))
			if merge {
				if _, ok := countedBarcodesFinished[countedBarcodes]; !ok {
					mergeRow := "\n" + convertedBarcodes
					for _, sampleBarcode := range sampleBarcodesSorted {
						mergeRow += "," + strconv.Itoa(c.NoRandom[sampleBarcode][countedBarcodes])
					}
					mergeOut.WriteString(mergeRow)
					countedBarcodesFinished[countedBarcodes] = true
				}
			}
			if total%10000 == 0 {
				fmt.Printf("\rTotal: %v", total)
			}
		}
		fmt.Printf("\rTotal: %v\nWriting...\n", total)
		outFileName := outpath + today + "_" + sampleBarcodes.Conversion[sampleBarcode] + ".counts.csv"
		file, err := os.Create(outFileName)
		if err != nil {
			log.Fatal(err)
		}
		_, writeErr := file.WriteString(sampleOut.String())
		if writeErr != nil {
			log.Fatal(err)
		}
	}
	fmt.Println()
	mergeFileName := outpath + today + "_counts.all.csv"
	mergeFile, mergeErr := os.Create(mergeFileName)
	if mergeErr != nil {
		log.Fatal(mergeErr)
	}
	_, mergeWriteErr := mergeFile.WriteString(mergeOut.String())
	if mergeWriteErr != nil {
		log.Fatal(mergeWriteErr)
	}
}

func convertCounted(countedBarcodes string, countedBarcodesStruct input.CountedBarcodes) string {
	var convertedBarcodes string
	for i, countedBarcode := range strings.Split(countedBarcodes, ",") {
		if len(convertedBarcodes) != 0 {
			convertedBarcodes += ","
		}
		convertedBarcodes += countedBarcodesStruct.Conversion[i][countedBarcode]
	}
	convertedBarcodes = strings.TrimSuffix(convertedBarcodes, ",")
	return convertedBarcodes
}

type ParseErrors struct {
	correct    int
	constant   int
	sample     int
	counted    int
	correctMu  sync.Mutex
	constantMu sync.Mutex
	sampleMu   sync.Mutex
	countedMu  sync.Mutex
}

func (p *ParseErrors) AddCorrect() {
	p.correctMu.Lock()
	p.correct++
	p.correctMu.Unlock()
}

func (p *ParseErrors) AddConstantError() {
	p.constantMu.Lock()
	p.constant++
	p.constantMu.Unlock()
}

func (p *ParseErrors) AddSampleError() {
	p.sampleMu.Lock()
	p.sample++
	p.sampleMu.Unlock()
}

func (p *ParseErrors) AddCountedError() {
	p.countedMu.Lock()
	p.counted++
	p.countedMu.Unlock()
}

func (p *ParseErrors) Print() {
	fmt.Printf("Correctly matched sequences: %v\n"+
		"Constant region errrors:     %v\n"+
		"Sample barcode errors:       %v\n"+
		"Counted barcode errors:      %v\n\n",
		p.correct, p.constant, p.sample, p.counted)
}
