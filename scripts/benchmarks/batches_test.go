package main

import (
	"strings"
	"testing"
)

func batchExample() batchCell {
	c := batchCell{Size: 10, Rows: 512, Status: "complete", Min: 100, Median: 101, Max: 102}
	for _, v := range []float64{102, 100, 101} {
		c.Samples = append(c.Samples, struct {
			MS float64 `json:"handler_ms"`
		}{v})
	}
	return c
}
func TestBatchChartExcludesIncompleteCells(t *testing.T) {
	c := batchExample()
	s := batchSweep{Revision: strings.Repeat("a", 40), Mode: "automatic", Cases: []batchCell{c, {Size: 50, Status: "interrupted", Median: 999}}}
	got, err := renderAutomaticBatches(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "50 entries") || !strings.Contains(got, "10 entries") || !strings.Contains(got, "0.101 s") || !strings.Contains(got, "prefers-color-scheme") {
		t.Fatal("invalid completed-cell chart")
	}
	again, err := renderAutomaticBatches(s)
	if err != nil || again != got {
		t.Fatal("nondeterministic render")
	}
	s.Cases[0].Median = 999
	if _, err := renderAutomaticBatches(s); err == nil {
		t.Fatal("bad summary accepted")
	}
	s.Cases[0] = c
	s.Cases = append(s.Cases, c)
	if _, err := renderAutomaticBatches(s); err == nil {
		t.Fatal("duplicate accepted")
	}
}
func TestBatchComparisonRequiresPairs(t *testing.T) {
	packed := batchExample()
	serial := batchExample()
	serial.Rows = 0
	if _, err := renderMultiFieldComparison([]batchCell{packed}, strings.Repeat("a", 40)); err == nil {
		t.Fatal("unpaired comparison")
	}
	got, err := renderMultiFieldComparison([]batchCell{packed, serial}, strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "1.00×") || !strings.Contains(got, "serial") {
		t.Fatal("missing labels")
	}
}
