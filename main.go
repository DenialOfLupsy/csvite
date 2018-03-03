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
var infile = flag.String("i", "", "The input file, optional if file is last parameter")
var selnum = flag.String("selnum", "", `Comma separated column numbers to print. 0 represents all the columns`)
var selhead = flag.String("selhead", "", `Comma separated column names to print. Cannot be used with -nh`)
var delnum = flag.String("delnum", "", `Comma separated column numbers not to print`)
var delhead = flag.String("delhead", "", `Comma separated column names not to print. Cannot be used with -nh`)
var sortnum = flag.String("sortnum", "", `Comma separated column numbers to sort by`)
var sorthead = flag.String("sorthead", "", `Comma separated column names to sort by. Cannot be used with -nh`)
var sortmode = flag.String("sortmode", "a", `a to sort alphabetically, n to sort numerically, v to sort according to semver`)

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: csvite [OPTIONS]... [FILE]")
		fmt.Println("Example: csvite -nh -sortnum 3 -sortmode v file.csv\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	var input io.Reader
	var err error

	if *infile == "" {
		*infile = flag.Arg(0)
		if len(flag.Args()) > 1 {
			fmt.Println("Too many arguments, unflagged file must be last parameter")
			os.Exit(2)
		}
	} else {
		if len(flag.Args()) > 0 {
			fmt.Println("Too many arguments, unflagged file must be last parameter")
			os.Exit(2)
		}
	}
	if *infile != "" {
		file, err := os.Open(*infile)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		input = file
	} else {
		stat, err := os.Stdin.Stat()
		if err != nil || (stat.Mode()&os.ModeNamedPipe == 0) {
			fmt.Println("No input file specified")
			os.Exit(2)
		}
		input = os.Stdin
	}

	r := csv.NewReader(input)       // r è un csv
	r.Comma = rune((*delimiter)[0]) // rune = char, assegno il valore del separatore
	r.LazyQuotes = true

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
		SelectColumnsByIndex(w, r, *selnum, SELECT)

	case *selhead != "":
		SelectColumnsByName(w, r, *selhead, SELECT)

	case *delnum != "":
		SelectColumnsByIndex(w, r, *delnum, DELETE)

	case *delhead != "":
		SelectColumnsByName(w, r, *delhead, DELETE)

	case *sortnum != "":
		SelectColumnsByIndex(w, r, *sortnum, SORT)

	case *sorthead != "":
		SelectColumnsByName(w, r, *sorthead, SORT)

	default:
		log.Println("No option specified")
		flag.PrintDefaults()
		return
	}

}

type Action int

const (
	SELECT Action = iota
	DELETE
	SORT
)

// SelectColumnsByName
func SelectColumnsByName(w *csv.Writer, r *csv.Reader, columns string, action Action) {
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

	if action == SORT {
		chooseMode(w, r, indexes)
		return
	}

	if action == DELETE {
		indexes = complement(indexes, colcount)
	}
	SelectColumns(w, r, indexes)
}

// SelectColumnsByIndex
func SelectColumnsByIndex(w *csv.Writer, r *csv.Reader, columns string, action Action) {
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
		if action != SELECT && indexes[i] == -1 {
			log.Fatal(errors.New("Cannot delete/sort all columns"))
		}
	}
	//fmt.Println("-- 144 --")
	if action == SORT {
		chooseMode(w, r, indexes)
		return
	}

	if action == DELETE {
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

type sortable struct {
	csv [][]string
	c   int
	m   Mode
}

func (a sortable) Len() int {
	return len(a.csv)
}
func (a sortable) Less(i, j int) bool {
	switch a.m {
	case ALPHABETICALLY:
		return a.csv[i][a.c] < a.csv[j][a.c]

	case NUMERICALLY:
		n1, _ := strconv.Atoi(a.csv[i][a.c])
		n2, _ := strconv.Atoi(a.csv[j][a.c])
		return n1 < n2

	case VERSION:
		splitted1 := strings.Split(a.csv[i][a.c], ".")
		splitted2 := strings.Split(a.csv[j][a.c], ".")
		var shorter int
		//determine the shortest one
		if len(splitted1) <= len(splitted2) {
			shorter = len(splitted1)
		} else {
			shorter = len(splitted2)
		}
		var sp1, sp2 []int
		//conversion to int
		for _, s1 := range splitted1 {
			ll, _ := strconv.Atoi(s1)
			sp1 = append(sp1, ll)
		}
		for _, s2 := range splitted2 {
			ll, _ := strconv.Atoi(s2)
			sp2 = append(sp2, ll)
		}

		for k := 0; k < shorter; k++ {
			switch {
			case sp1[k] < sp2[k]:
				return true
			case sp1[k] > sp2[k]:
				return false
			}
		}
		if len(sp1) < len(sp2) {
			return true
		}
		return false

	}
	panic("Invalid mode specified")
}

func (a sortable) Swap(i, j int) {
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
	sort.Sort(sortable{csv, col[0], m})

	for _, c := range col[1:] {
		sort.Stable(sortable{csv, c, m})
	}

	w.WriteAll(csv)
}

func chooseMode(w *csv.Writer, r *csv.Reader, indexes []int) {
	var mode Mode
	switch {
	case strings.HasPrefix(strings.ToLower(*sortmode), "a"):
		mode = ALPHABETICALLY
	case strings.HasPrefix(strings.ToLower(*sortmode), "n"):
		mode = NUMERICALLY
	case strings.HasPrefix(strings.ToLower(*sortmode), "v"):
		mode = VERSION
	}
	sortcsv(w, r, mode, indexes...)
}
