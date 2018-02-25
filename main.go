package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

var headers []string
var firstrow []string
var colcount int

var noHeader = flag.Bool("nh", false, "Specify when there are no column headers")
var delimiter = flag.String("d", ",", "The character to use as delimiter")
var infile = flag.String("i", "file.csv", "The input file")
var selnum = flag.String("selnum", "", `Comma separated column numbers to print. 0 represents all the columns`)
var selhead = flag.String("selhead", "", `Comma separated column names to print. Cannot be used with -nh`)
var delnum = flag.String("delnum", "", `Comma separated column numbers not to print.`)
var delhead = flag.String("delhead", "", `Comma separated column names not to print. Cannot be used with -nh`)

func main() {

	flag.Parse()

	file, err := os.Open(*infile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	r := csv.NewReader(file)        // r è un csv
	r.Comma = rune((*delimiter)[0]) // rune = char, assegno il valore del separatore

	headers, err = r.Read()
	if err != nil {
		log.Fatal(err)
	}
	colcount = len(headers)
	if *noHeader {
		firstrow = headers
		headers = nil
	}

	w := csv.NewWriter(os.Stdout)
	//stampare gli headr se ci sono

	switch {
	case *selnum != "":
		SelectColumnsByIndex(w, r, *selnum, false)

	case *selhead != "":
		SelectColumnsByName(w, r, *selhead, false)

	case *delnum != "":
		SelectColumnsByIndex(w, r, *delnum, true)

	case *delhead != "":
		SelectColumnsByName(w, r, *delhead, true)

	default:
		log.Println("No option specified")
		flag.PrintDefaults()
		return
	}

}

// SelectColumnsByName
func SelectColumnsByName(w *csv.Writer, r *csv.Reader, columns string, invert bool) {
	splitted := strings.Split(columns, ",")
	indexes := make([]int, len(splitted))

	if headers == nil {
		log.Fatal(errors.New("Title specified, but the file doesn't contain headers"))
	}
	for i, s := range splitted {
		found := false
		for j, h := range headers {
			if s == h {
				indexes[i] = j
				found = true
				break
			}
		}
		if !found {
			log.Fatal(fmt.Errorf("Column name not found : %s", s))
		}
	}

	if invert {
		indexes = complement(indexes, colcount)
	}
	SelectColumns(w, r, indexes)
}

// SelectColumnsByIndex
func SelectColumnsByIndex(w *csv.Writer, r *csv.Reader, columns string, invert bool) {
	splitted := strings.Split(columns, ",")
	indexes := make([]int, len(splitted))

	for i, s := range splitted {
		var err error
		indexes[i], err = strconv.Atoi(s)
		indexes[i]-- // index from 1 not from 0
		if err != nil {
			log.Fatal(err)
		}
		if indexes[i] < -1 {
			log.Fatal(fmt.Errorf("Column index out of range : %d", indexes[i]+1))
		}
		if invert && indexes[i] == -1 {
			log.Fatal(errors.New("Cannot delete all columns"))
		}
	}

	if invert {
		indexes = complement(indexes, colcount)
	}
	SelectColumns(w, r, indexes)
}

// SelectColumns
func SelectColumns(w *csv.Writer, r *csv.Reader, indexes []int) {

	var row []string

	defer w.Flush()

ReadLoop:
	for {
		var err error
		var record []string

		if firstrow != nil {
			record = firstrow
			firstrow = nil
		} else {
			record, err = r.Read()
			switch err {
			case io.EOF:
				break ReadLoop
			case nil:
			default:
				log.Println(err)
				return
			}
		}

		for _, s := range indexes {
			if s > len(record) {
				log.Fatal(fmt.Errorf("Column index out of range : %d", s+1))
			}

			if s == -1 {
				row = append(row, record...)
				continue
			}
			row = append(row, record[s])
		}

		err = w.Write(row)
		if err != nil {
			log.Fatal(err)
		}

		row = row[:0]
	}
}

func complement(subset []int, length int) []int {
	var compl = make([]int, 0, length-len(subset))
	exist := false
	for i := 0; i < length; i++ {
		for _, ss := range subset {
			if i == ss {
				exist = true
				break
			}
		}
		if !exist {
			compl = append(compl, i)
		}
		exist = false
	}
	return compl
}

type alphabetically struct {
	csv [][]string
	c   int
}

func (a alphabetically) Len() int {
	return len(a.csv)
}

func (a alphabetically) Less(i, j int) bool {
	return a.csv[i][a.c] < a.csv[j][a.c]
}

func (a alphabetically) Swap(i, j int) {
	a.csv[j], a.csv[i] = a.csv[i], a.csv[j]
}

type Mode int

const (
	ALPHABETICALLY Mode = iota
	NUMERICALLY
	VERSION
)

func sortcsv(w *csv.Writer, r *csv.Reader, m Mode, col ...int) {
	var csv [][]string
	if firstrow != nil {
		csv = append(csv, firstrow)
		firstrow = nil
	}

	tmp, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	csv = append(csv, tmp...)

	alpha := alphabetically{csv, col[0]}
	sort.Sort(alpha)
	w.WriteAll(csv)
}
