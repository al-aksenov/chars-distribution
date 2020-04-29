package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

var (
	dirPath    = "data"
	workersNum = 4
	bufferSize = 4
	maxProcs   = 0
)

func main() {

	log.Printf("Start research %s", dirPath)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("directory %s not found\n", dirPath)
		return
	}

	runtime.GOMAXPROCS(maxProcs)
	filesChan := make(chan string, bufferSize)
	resultsChan := make(chan []uint64)
	wg := &sync.WaitGroup{}
	for i := 0; i < workersNum; i++ {
		wg.Add(1)
		go fileWorker(filesChan, resultsChan, wg)
	}
	filesProcessing(files, dirPath, filesChan)
	close(filesChan)

	go func(wg *sync.WaitGroup, resultsChan chan []uint64) {
		wg.Wait()
		close(resultsChan)
	}(wg, resultsChan)

	distribution := totalDistribution(resultsChan)

	barPlot(distribution)

	log.Println("Quit")
}

func filesProcessing(flist []os.FileInfo, currentPath string, filesChan chan<- string) {
	for _, f := range flist {
		newPath := path.Join(currentPath, f.Name())
		if f.IsDir() {
			files, err := ioutil.ReadDir(newPath)
			if err != nil {
				fmt.Printf("cannot read %s\n", newPath)
				continue
			}
			filesProcessing(files, newPath, filesChan)
		} else {
			filesChan <- newPath
		}
	}
}

func fileWorker(in <-chan string, out chan<- []uint64, wg *sync.WaitGroup) {
	defer wg.Done()
	distribution := make([]uint64, 256)
	for fname := range in {
		file, err := os.Open(fname)
		if err != nil {
			fmt.Printf("cannot open %s\n", fname)
			continue
		}
		rd := bufio.NewReader(file)
		for b, err := rd.ReadByte(); err == nil; b, err = rd.ReadByte() {
			distribution[b]++
		}
		file.Close()
	}
	out <- distribution
}

func totalDistribution(resultsChan <-chan []uint64) []uint64 {
	result := make([]uint64, 256)
	for d := range resultsChan {
		for i, val := range d {
			result[i] += val
		}
	}
	return result
}

func barPlot(d []uint64) {

	n := 128
	values := make(plotter.Values, n)
	for i := 0; i < n; i++ {
		values[i] = float64(d[i])
	}
	asciiSigns := make([]string, n)
	for i := 0; i < n; i++ {
		asciiSigns[i] = strings.Trim(strconv.QuoteToASCII(string(byte(i))), "\"")
	}
	w := vg.Points(5)

	for i := 0; i < 2; i++ {
		p, err := plot.New()
		if err != nil {
			fmt.Println("cannot create new plot")
			return
		}
		p.Title.Text = fmt.Sprintf("%v - %v ascii chars distribution", i*64, (i+1)*64-1)
		bar, err := plotter.NewBarChart(values[i*64:(i+1)*64], w)
		if err != nil {
			fmt.Println("cannot create bar chart")
			return
		}
		bar.Color = plotutil.Color(2)
		bar.LineStyle.Width = vg.Length(0)
		p.Add(bar)
		p.NominalX(asciiSigns[i*64 : (i+1)*64]...)
		// p.X.Tick.Label.Rotation = 1.57
		fname := fmt.Sprintf("barchart_%v.png", i)
		if err := p.Save(17*vg.Inch, 5*vg.Inch, fname); err != nil {
			fmt.Println("cannot save png file")
			return
		}
	}
}
