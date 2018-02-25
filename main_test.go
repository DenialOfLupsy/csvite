package main

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

func TestSortcsv(t *testing.T) {
	csv1 := "a,2\nb,1\n1,3"
	tests := []struct {
		columns []int
		mode    Mode
		out     string
	}{
		{
			[]int{0},
			ALPHABETICALLY,
			"1,3\na,2\nb,1\n",
		},
		{
			[]int{1},
			ALPHABETICALLY,
			"b,1\na,2\n1,3\n",
		},
	}

	for _, test := range tests {
		str := strings.NewReader(csv1)
		csvr := csv.NewReader(str)
		buf := bytes.NewBuffer(nil)
		csvw := csv.NewWriter(buf)
		sortcsv(csvw, csvr, test.mode, test.columns...)

		output := string(buf.Bytes())

		if output != test.out {
			t.Errorf("Expected: %s, but got %s", test.out, output)
		}
	}

}
