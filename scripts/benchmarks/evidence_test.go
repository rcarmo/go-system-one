package main

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

// Audit derived chart summaries against the frozen raw evidence. Rendering a
// deterministic SVG alone would not catch a stale summary or mixed revisions.
func TestCurrentChartEvidenceMatchesRawRuns(t *testing.T) {
	root := "../../docs/benchmarks/data/"
	var data benchmarkData
	if err := readJSON(root+"current.json", &data); err != nil {
		t.Fatal(err)
	}
	if err := validateBenchmarkData(data); err != nil {
		t.Fatal(err)
	}
	type identity struct {
		Revision string `json:"revision"`
		Binary   string `json:"binary_sha256"`
		Model    string `json:"model_sha256"`
	}
	var expected identity
	if err := readJSON(root+"q6-staged-warm.json", &expected); err != nil {
		t.Fatal(err)
	}
	if expected.Revision != data.Fixture.Revision || len(expected.Binary) != 64 || expected.Model != data.Fixture.ModelSHA256 {
		t.Fatal("current fixture identity mismatch")
	}
	for _, name := range []string{"q6-staged-workloads.json", "q6-staged-batches.json", "q6-staged-serial.json"} {
		var got identity
		if err := readJSON(root+name, &got); err != nil {
			t.Fatal(err)
		}
		if got != expected {
			t.Fatalf("%s identity=%+v want=%+v", name, got, expected)
		}
	}
	var warm sampleData
	if err := readJSON(root+"q6-staged-warm.json", &warm); err != nil {
		t.Fatal(err)
	}
	if err := validateSamples(warm); err != nil {
		t.Fatal(err)
	}
	last := data.Comparison[len(data.Comparison)-1]
	if !strings.HasPrefix(last.Label, "Current") || last.RepresentativeMS != warm.Median || last.MinMS != warm.Min || last.MaxMS != warm.Max {
		t.Fatal("current comparison does not match warm samples")
	}
	var workloads struct {
		Cases []struct {
			sampleData
			Label string `json:"label"`
		}
	}
	if err := readJSON(root+"q6-staged-workloads.json", &workloads); err != nil {
		t.Fatal(err)
	}
	if len(workloads.Cases) != len(data.Workloads) {
		t.Fatal("workload count mismatch")
	}
	for i, c := range workloads.Cases {
		if err := validateSamples(c.sampleData); err != nil {
			t.Fatal(err)
		}
		want := data.Workloads[i]
		if c.Label != want.Label || c.Median != want.MedianMS || c.Min != want.MinMS || c.Max != want.MaxMS {
			t.Fatalf("workload %d does not match samples", i)
		}
	}
	type rawCell struct {
		batchCell
		Request json.RawMessage `json:"request"`
	}
	type rawRun struct {
		Revision string    `json:"revision"`
		Binary   string    `json:"binary_sha256"`
		Cases    []rawCell `json:"cases"`
	}
	var paired, automatic, serial rawRun
	for name, dst := range map[string]*rawRun{"q6-staged-paired.json": &paired, "q6-staged-batches.json": &automatic, "q6-staged-serial.json": &serial} {
		if err := readJSON(root+name, dst); err != nil {
			t.Fatal(err)
		}
		if dst.Revision != expected.Revision || dst.Binary != expected.Binary {
			t.Fatalf("%s has different identity", name)
		}
	}
	if len(paired.Cases) != 2*len(serial.Cases) {
		t.Fatal("paired result count mismatch")
	}
	for _, c := range paired.Cases {
		if err := validateBatchCell(c.batchCell); err != nil {
			t.Fatal(err)
		}
		reference := automatic.Cases
		if c.Rows == 0 {
			reference = serial.Cases
		}
		found := false
		for _, original := range reference {
			if original.Size == c.Size {
				found = true
				if c.Status != "complete" || !reflect.DeepEqual(c.Samples, original.Samples) || c.Median != original.Median {
					t.Fatal("paired timings differ from raw run")
				}
				var a, b any
				if json.Unmarshal(c.Request, &a) != nil || json.Unmarshal(original.Request, &b) != nil || !reflect.DeepEqual(a, b) {
					t.Fatal("paired request differs from raw run")
				}
			}
		}
		if !found {
			t.Fatal("paired case absent from raw run")
		}
	}
	for _, s := range serial.Cases {
		found := false
		for _, a := range automatic.Cases {
			if a.Size == s.Size {
				found = true
				var ar, sr any
				if json.Unmarshal(a.Request, &ar) != nil || json.Unmarshal(s.Request, &sr) != nil || !reflect.DeepEqual(ar, sr) {
					t.Fatal("serial/automatic input mismatch")
				}
			}
		}
		if !found {
			t.Fatal("serial case missing automatic partner")
		}
	}
}

func TestDistributionHandlesConstantAndSingleSample(t *testing.T) {
	for _, samples := range [][]float64{{80}, {80, 80, 80}} {
		d := sampleData{N: len(samples), Min: 80, Max: 80, Median: 80, P95: 80, P99: 80, Samples: samples}
		if err := validateSamples(d); err != nil {
			t.Fatal(err)
		}
		got := renderDistribution(d)
		if strings.Contains(got, "NaN") || strings.Contains(got, "Inf") {
			t.Fatal("non-finite chart coordinate")
		}
	}
	if near(math.NaN(), 1) {
		t.Fatal("NaN treated as matching evidence")
	}
}
