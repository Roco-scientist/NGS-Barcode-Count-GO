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

const NoSampleName = "barcode"

// Counts holds the accumulated counts for each sequence.  NoRandom holds the count when there is not a random barcode included.
// The format of NoRandom is SampleBarcode:CommaSeparatedCountedBarcodes:Count.  Random is used when a random barcode is included.
// This map holds SampleBarcode:CommaSeparatedCountedBarcodes:RandomBarcodes:true.  Since RandomBarcodes is a map key, this only
// holds unique RandomBarcodes causing duplicates to be discarded.
type Counts struct {
	mu sync.Mutex
	// NoRandom holds counts when there is not a random barcode
	NoRandom map[string]map[string]int
	// Random holds counts when there is a random barcode
	Random map[string]map[string]map[string]bool
	// single holds counts for single barcode enrichment
	single map[string]map[string]int
	// double holds counts for double barcode enrichment
	double                  map[string]map[string]int
	sampleOut               strings.Builder
	sampleOutSingle         strings.Builder
	sampleOutDouble         strings.Builder
	mergeOut                strings.Builder
	mergeOutSingle          strings.Builder
	mergeOutDouble          strings.Builder
	countedBarcodesFinished map[string]bool
	sampleBarcodesSorted    []string
	merge                   bool
	enrich                  bool
	barcodeNum              int
}

// NewCount creates a new Counts struct.  It inserts the sampleBarcodes into NoRandom and Random maps to prevent a nil map insert
// when trying to insert a value later
func NewCount(sampleBarcodes []string) *Counts {
	var count Counts
	count.NoRandom = make(map[string]map[string]int)
	count.single = make(map[string]map[string]int)
	count.double = make(map[string]map[string]int)
	count.Random = make(map[string]map[string]map[string]bool)
	for _, sampleBarcode := range sampleBarcodes {
		count.NoRandom[sampleBarcode] = make(map[string]int)
		count.single[sampleBarcode] = make(map[string]int)
		count.double[sampleBarcode] = make(map[string]int)
		count.Random[sampleBarcode] = make(map[string]map[string]bool)
	}
	return &count
}

// AddCount adds 1 to NoRandom map if a random barcode is not included.  Adds the random barcode to the Random map if a
// random barcode is included.  This is done in a thread safe manner using mutex locks part of the Counts struct.  A bool
// is returned if the insertion or addition was successful.  This is used for when the random barcode already exists for the
// sample:countedBarcodes.  When this is the case false is returned.  This false is used elsewhere to record how many duplicates
// occured.
func (c *Counts) AddCount(sampleBarcode string, countedBarcodes string, randomBarcode string, samplBarcodeIncluded bool) bool {
	if !samplBarcodeIncluded {
		sampleBarcode = NoSampleName
	}
	if randomBarcode == "" {
		c.mu.Lock()
		c.NoRandom[sampleBarcode][countedBarcodes]++
		c.mu.Unlock()
	} else {
		c.mu.Lock()
		if _, ok := c.Random[sampleBarcode][countedBarcodes]; !ok {
			newMap := make(map[string]bool)
			newMap[randomBarcode] = true
			c.Random[sampleBarcode][countedBarcodes] = newMap
			c.mu.Unlock()
		} else {
			if _, ok := c.Random[sampleBarcode][countedBarcodes][randomBarcode]; ok {
				c.mu.Unlock()
				return false
			} else {
				c.Random[sampleBarcode][countedBarcodes][randomBarcode] = true
				c.mu.Unlock()
			}
		}
	}
	return true
}

// WriteCsv writes the counts to csv files.  It creates a separate file for each sample.  If the --merge flag is called, it also outputs a csv
// which merges the results into one file where each sample gets a column.  This method works for both Random and NoRandom results.  The method is
// split when starting to need to use either map due to the different formats of the two datasets
func (c *Counts) WriteCsv(outpath string, merge bool, enrich bool, countedBarcodesStruct input.CountedBarcodes, sampleBarcodes input.SampleBarcodes) {
	c.merge = merge
	c.enrich = enrich

	// sampleBarcodes will be unordered.  The following orders the sampleBarcodes by the order of the sampleIDs.  This is necessary
	// for clean merged file output
	var sampleIds []string
	for key := range sampleBarcodes.Conversion {
		sampleIds = append(sampleIds, sampleBarcodes.Conversion[key])
	}
	sort.Strings(sampleIds)

	for _, sampleId := range sampleIds {
		for key, value := range sampleBarcodes.Conversion {
			if value == sampleId {
				c.sampleBarcodesSorted = append(c.sampleBarcodesSorted, key)
				break
			}
		}
	}

	// headerStart holds the header for the CSV files.  It will generally be Barcode_1,Barcode_2,..,Barcode_N.
	// This is then used to create the final header.  For the smaple files, the next column is Count.  For
	// merge file, the next columns are the sample names
	var headerStart string
	for i := 0; i < countedBarcodesStruct.NumBarcodes; i++ {
		headerStart += "Barcode_" + strconv.Itoa(i+1) + ","
	}

	sampleHeader := headerStart + "Count"

	if c.merge {
		// countedBarcodesFinished holds what comma separated counted barcodes have already been done.  This is
		// used while creating the merge file so that countedBarcodes are not repeatedly counted
		c.countedBarcodesFinished = make(map[string]bool)

		mergeHeader := headerStart
		for i, sampleId := range sampleIds {
			if i != 0 {
				mergeHeader += ","
			}
			mergeHeader += sampleId
		}
		c.mergeOut.WriteString(mergeHeader)
	}
	today := time.Now().Local().Format("2006-01-02")
	for _, sampleBarcode := range c.sampleBarcodesSorted {
		fmt.Printf("Gathering for %v\n", sampleBarcodes.Conversion[sampleBarcode])
		c.sampleOut.WriteString(sampleHeader)
		var total int
		// If there were no random barcodes use gatherCounts, otherwise use gatherRandom
		if len(c.Random[sampleBarcode]) == 0 {
			total = c.gatherCounts(sampleBarcode, countedBarcodesStruct)
		} else {
			total = c.gatherRandom(sampleBarcode, countedBarcodesStruct)
		}

		// After the gathering is finished, the final count is printed
		fmt.Printf("\rTotal: %v\nWriting...\n", total)
		outFileName := outpath + today + "_" + sampleBarcodes.Conversion[sampleBarcode] + "_counts.csv"
		file, err := os.Create(outFileName)
		if err != nil {
			log.Fatal(err)
		}
		_, writeErr := file.WriteString(c.sampleOut.String())
		if writeErr != nil {
			log.Fatal(err)
		}
		c.sampleOut.Reset()
	}
	fmt.Println()
	// If merge is called, write the merge file
	if c.merge {
		mergeFileName := outpath + today + "_counts.all.csv"
		mergeFile, mergeErr := os.Create(mergeFileName)
		if mergeErr != nil {
			log.Fatal(mergeErr)
		}
		_, mergeWriteErr := mergeFile.WriteString(c.mergeOut.String())
		if mergeWriteErr != nil {
			log.Fatal(mergeWriteErr)
		}
		c.mergeOut.Reset()
	}
	if c.enrich {
		if c.merge {
			// countedBarcodesFinished holds what comma separated counted barcodes have already been done.  This is
			// used while creating the merge file so that countedBarcodes are not repeatedly counted
			for k := range c.countedBarcodesFinished {
				delete(c.countedBarcodesFinished, k)
			}

			mergeHeader := headerStart
			for i, sampleId := range sampleIds {
				if i != 0 {
					mergeHeader += ","
				}
				mergeHeader += sampleId
			}
			c.mergeOutSingle.WriteString(mergeHeader)
			c.mergeOutDouble.WriteString(mergeHeader)
		}
		for _, sampleBarcode := range c.sampleBarcodesSorted {
			fmt.Printf("Gathering for single/double enriched %v\n", sampleBarcodes.Conversion[sampleBarcode])
			c.sampleOutSingle.WriteString(sampleHeader)
			c.sampleOutDouble.WriteString(sampleHeader)

			totalSingle, totalDouble := c.gatherEnriched(sampleBarcode)

			// After the gathering is finished, the final count is printed
			fmt.Printf("\rTotal single enriched: %v\nTotal double enriched: %v\n", totalSingle, totalDouble)
			if totalSingle != 0 {
				outFileNameSingle := outpath + today + "_" + sampleBarcodes.Conversion[sampleBarcode] + "_counts.Single.csv"
				file, err := os.Create(outFileNameSingle)
				if err != nil {
					log.Fatal(err)
				}
				_, writeErr := file.WriteString(c.sampleOutSingle.String())
				if writeErr != nil {
					log.Fatal(err)
				}
				c.sampleOutSingle.Reset()
			}
			if totalDouble != 0 {
				outFileNameDouble := outpath + today + "_" + sampleBarcodes.Conversion[sampleBarcode] + "_counts.Double.csv"
				file, err := os.Create(outFileNameDouble)
				if err != nil {
					log.Fatal(err)
				}
				_, writeErr := file.WriteString(c.sampleOutDouble.String())
				if writeErr != nil {
					log.Fatal(err)
				}
				c.sampleOutDouble.Reset()
			}
		}
		fmt.Println()
		// If merge is called, write the merge file
		if c.merge {
			if c.barcodeNum > 1 {
				mergeFileNameSingle := outpath + today + "_counts.all.Single.csv"
				mergeFile, mergeErr := os.Create(mergeFileNameSingle)
				if mergeErr != nil {
					log.Fatal(mergeErr)
				}
				_, mergeWriteErr := mergeFile.WriteString(c.mergeOutSingle.String())
				if mergeWriteErr != nil {
					log.Fatal(mergeWriteErr)
				}
				c.mergeOutSingle.Reset()
			}
			if c.barcodeNum > 2 {
				mergeFileNameDouble := outpath + today + "_counts.all.Double.csv"
				mergeFile, mergeErr := os.Create(mergeFileNameDouble)
				if mergeErr != nil {
					log.Fatal(mergeErr)
				}
				_, mergeWriteErr := mergeFile.WriteString(c.mergeOutDouble.String())
				if mergeWriteErr != nil {
					log.Fatal(mergeWriteErr)
				}
				c.mergeOutDouble.Reset()
			}
		}
	}
}

// gatherCounts is a method for gathering all counts into a comma separated string which, when written to a file,
// will create a csv file.  It returns the total number of different countedBarcodes to record to stdout later
func (c *Counts) gatherCounts(sampleBarcode string, countedBarcodesStruct input.CountedBarcodes) int {
	total := 0
	for countedBarcodes, count := range c.NoRandom[sampleBarcode] {
		total++
		var convertedBarcodes string
		if countedBarcodesStruct.Included {
			convertedBarcodes = convertCounted(countedBarcodes, countedBarcodesStruct)
		} else {
			convertedBarcodes = countedBarcodes
		}
		c.sampleOut.WriteString("\n" + convertedBarcodes + "," + strconv.Itoa(count))
		if c.merge {
			if _, ok := c.countedBarcodesFinished[countedBarcodes]; !ok {
				mergeRow := "\n" + convertedBarcodes
				for _, sampleBarcode := range c.sampleBarcodesSorted {
					mergeRow += "," + strconv.Itoa(c.NoRandom[sampleBarcode][countedBarcodes])
				}
				c.mergeOut.WriteString(mergeRow)
				c.countedBarcodesFinished[countedBarcodes] = true
			}
		}
		if c.enrich {
			c.addEnrichment(sampleBarcode, convertedBarcodes, count)
		}
		if total%10000 == 0 {
			fmt.Printf("\rTotal: %v", total)
		}
	}
	return total
}

// gatherRandom gathers counts when a random barcode is used.  It finds the number of random barcodes per sample:countedBarcodes and
// used this for the count.  It returns the total number of different countedBarcodes to record to stdout later
func (c *Counts) gatherRandom(sampleBarcode string, countedBarcodesStruct input.CountedBarcodes) int {
	total := 0
	for countedBarcodes, randomBarcodesMap := range c.Random[sampleBarcode] {
		count := len(randomBarcodesMap)
		total++
		var convertedBarcodes string
		if countedBarcodesStruct.Included {
			convertedBarcodes = convertCounted(countedBarcodes, countedBarcodesStruct)
		} else {
			convertedBarcodes = countedBarcodes
		}
		c.sampleOut.WriteString("\n" + convertedBarcodes + "," + strconv.Itoa(count))
		if c.merge {
			if _, ok := c.countedBarcodesFinished[countedBarcodes]; !ok {
				mergeRow := "\n" + convertedBarcodes
				for _, sampleBarcode := range c.sampleBarcodesSorted {
					sampleCount := len(c.Random[sampleBarcode][countedBarcodes])
					mergeRow += "," + strconv.Itoa(sampleCount)
				}
				c.mergeOut.WriteString(mergeRow)
				c.countedBarcodesFinished[countedBarcodes] = true
			}
		}
		if c.enrich {
			c.addEnrichment(sampleBarcode, convertedBarcodes, count)
		}
		if total%10000 == 0 {
			fmt.Printf("\rTotal: %v", total)
		}
	}
	return total
}

// convertCounted splits the countedBarcodes string by the ','s, converts the DNA barcode to barcode ID,
// which could be a SMILES string for DEL or whatever identifier is used.  It then combines it back to a
// comma separated string of converted barcodes
func convertCounted(countedBarcodes string, countedBarcodesStruct input.CountedBarcodes) string {
	var convertedBarcodes string
	for i, countedBarcode := range strings.Split(countedBarcodes, ",") {
		if len(convertedBarcodes) != 0 {
			convertedBarcodes += ","
		}
		convertedBarcodes += countedBarcodesStruct.Conversion[i][countedBarcode]
	}
	return convertedBarcodes
}

func (c *Counts) addEnrichment(sampleBarcode string, convertedBarcodes string, count int) {
	barcodesSplit := strings.Split(convertedBarcodes, ",")
	c.barcodeNum = len(barcodesSplit)
	if c.barcodeNum > 1 {
		for i, barcode := range barcodesSplit {
			var barcodeString string
			for j := 0; j < c.barcodeNum; j++ {
				if j != 0 {
					barcodeString += ","
				}
				if i == j {
					barcodeString += barcode
				}
			}
			c.single[sampleBarcode][barcodeString] += count
		}
	}
	if c.barcodeNum > 2 {
		for firstBarcodeIndex := 0; firstBarcodeIndex < (c.barcodeNum - 1); firstBarcodeIndex++ {
			for nextBarcodeAdd := 1; nextBarcodeAdd < (c.barcodeNum - firstBarcodeIndex); nextBarcodeAdd++ {
				var doubleBarcodeString string
				for columnIndex := 0; columnIndex < c.barcodeNum; columnIndex++ {
					if columnIndex != 0 {
						doubleBarcodeString += ","
					}
					if columnIndex == firstBarcodeIndex {
						doubleBarcodeString += barcodesSplit[firstBarcodeIndex]
					}
					if columnIndex == (firstBarcodeIndex + nextBarcodeAdd) {
						doubleBarcodeString += barcodesSplit[(firstBarcodeIndex + nextBarcodeAdd)]
					}
				}
				c.double[sampleBarcode][doubleBarcodeString] += count
			}
		}
	}
}

func (c *Counts) gatherEnriched(sampleBarcode string) (int, int) {
	var totalSingle, totalDouble int
	if len(c.single[sampleBarcode]) != 0 {
		for convertedBarcodes, count := range c.single[sampleBarcode] {
			totalSingle++
			c.sampleOutSingle.WriteString("\n" + convertedBarcodes + "," + strconv.Itoa(count))
			if c.merge {
				if _, ok := c.countedBarcodesFinished[convertedBarcodes]; !ok {
					mergeRow := "\n" + convertedBarcodes
					for _, sampleBarcode := range c.sampleBarcodesSorted {
						mergeRow += "," + strconv.Itoa(c.single[sampleBarcode][convertedBarcodes])
					}
					c.mergeOutSingle.WriteString(mergeRow)
					c.countedBarcodesFinished[convertedBarcodes] = true
				}
			}
			if totalSingle%10000 == 0 {
				fmt.Printf("\rTotal: %v", totalSingle)
			}
		}
	}
	if len(c.double[sampleBarcode]) != 0 {
		for convertedBarcodes, count := range c.double[sampleBarcode] {
			totalDouble++
			c.sampleOutDouble.WriteString("\n" + convertedBarcodes + "," + strconv.Itoa(count))
			if c.merge {
				if _, ok := c.countedBarcodesFinished[convertedBarcodes]; !ok {
					mergeRow := "\n" + convertedBarcodes
					for _, sampleBarcode := range c.sampleBarcodesSorted {
						mergeRow += "," + strconv.Itoa(c.single[sampleBarcode][convertedBarcodes])
					}
					c.mergeOutDouble.WriteString(mergeRow)
					c.countedBarcodesFinished[convertedBarcodes] = true
				}
			}
			if totalDouble%10000 == 0 {
				fmt.Printf("\rTotal: %v", totalDouble)
			}
		}
	}
	return totalSingle, totalDouble
}

type ParseErrors struct {
	correct     int
	constant    int
	sample      int
	counted     int
	duplicate   int
	correctMu   sync.Mutex
	constantMu  sync.Mutex
	sampleMu    sync.Mutex
	countedMu   sync.Mutex
	duplicateMu sync.Mutex
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

func (p *ParseErrors) AddDuplicateError() {
	p.duplicateMu.Lock()
	p.duplicate++
	p.duplicateMu.Unlock()
}

func (p *ParseErrors) Print() {
	fmt.Printf("Correctly matched sequences: %v\n"+
		"Constant region errrors:     %v\n"+
		"Sample barcode errors:       %v\n"+
		"Counted barcode errors:      %v\n"+
		"Duplicates:                  %v\n\n",
		p.correct, p.constant, p.sample, p.counted, p.duplicate)
}

type MaxBarcodeErrorsAllowed struct {
	Sample       int
	sampleSize   int
	Counted      int
	countedSizes []int
	Constant     int
	constantSize int
}

// NewMaxErrors creates a MaxBarcodeErrorsAllowed struct which includes how many errors are allowed per sequence barcode.
// If --max-errors flags are used, this number will be used for the number of allowed sequence errors.  Otherwise
// 20% of the length of each barcode is used.
func NewMaxErrors(sample int, counted int, constant int, format input.SequenceFormat) MaxBarcodeErrorsAllowed {
	var maxErrors MaxBarcodeErrorsAllowed
	maxErrors.sampleSize = format.SampleSize
	maxErrors.countedSizes = format.CountedBarcodesSizes
	maxErrors.constantSize = format.ConstantSize
	if sample == -1 {
		maxErrors.Sample = format.SampleSize / 5
	} else {
		maxErrors.Sample = sample
	}

	if counted == -1 {
		var averageCountedSize int
		for _, countedSize := range format.CountedBarcodesSizes {
			averageCountedSize += countedSize
		}
		averageCountedSize /= len(format.CountedBarcodesSizes)
		maxErrors.Counted = averageCountedSize / 5
	} else {
		maxErrors.Counted = counted
	}

	if constant == -1 {
		maxErrors.Constant = format.ConstantSize / 5
	} else {
		maxErrors.Constant = constant
	}
	return maxErrors
}

func (m MaxBarcodeErrorsAllowed) Print() {
	fmt.Printf("-BARCODE INFO-\n"+
		"Constant region size: %v\n"+
		"Maximum mismatches allowed per sequence: %v\n"+
		"--------------------------------------------------------------\n"+
		"Sample barcode size: %v\n"+
		"Maximum mismatches allowed per sequence: %v\n"+
		"--------------------------------------------------------------\n"+
		"Barcode sizes: %v\n"+
		"Maximum mismatches allowed per barcode sequence: %v\n"+
		"--------------------------------------------------------------\n\n",
		m.constantSize, m.Constant, m.sampleSize, m.Sample, m.countedSizes, m.Counted)

}
