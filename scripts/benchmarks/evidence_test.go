package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
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
	if err := readJSON(root+"q5-chunk512-warm.json", &expected); err != nil {
		t.Fatal(err)
	}
	if expected.Revision != data.Fixture.Revision || len(expected.Binary) != 64 || expected.Model != data.Fixture.ModelSHA256 {
		t.Fatal("current fixture identity mismatch")
	}
	for _, name := range []string{"q5-chunk512-workloads.json", "q5-chunk512-batches.json", "q5-chunk512-serial.json"} {
		var got identity
		if err := readJSON(root+name, &got); err != nil {
			t.Fatal(err)
		}
		if got != expected {
			t.Fatalf("%s identity=%+v want=%+v", name, got, expected)
		}
	}
	var warm sampleData
	if err := readJSON(root+"q5-chunk512-warm.json", &warm); err != nil {
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
	if err := readJSON(root+"q5-chunk512-workloads.json", &workloads); err != nil {
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
	for name, dst := range map[string]*rawRun{"q5-chunk512-paired.json": &paired, "q5-chunk512-batches.json": &automatic, "q5-chunk512-serial.json": &serial} {
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
	for _, check := range []struct {
		run   rawRun
		sizes []int
	}{{automatic, []int{1, 10, 25, 50, 100}}, {serial, []int{1, 10}}} {
		if len(check.run.Cases) != len(check.sizes) {
			t.Fatal("incomplete sweep")
		}
		for i, cell := range check.run.Cases {
			if cell.Size != check.sizes[i] || cell.Status != "complete" {
				t.Fatal("wrong or incomplete sweep cell")
			}
			if err := validateBatchCell(cell.batchCell); err != nil {
				t.Fatal(err)
			}
		}
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

func TestTypeSafeEvidence(t *testing.T) {
	var run struct {
		Schema, Revision string
		Binary           string `json:"binary_sha256"`
		Model            string `json:"model_sha256"`
		Cases            []struct {
			sampleData
			Label, Status string
			Request       json.RawMessage
			RequestHash   string `json:"request_sha256"`
			Trials        []struct {
				HTTPMS         float64 `json:"http_ms"`
				Before         struct{ Temperature float64 }
				MaxTemperature float64 `json:"max_temperature_c"`
				Response       struct {
					Answers map[string]struct {
						Type          string
						Noul          *float64
						Choice        string
						Score         *float64
						Confidence    *float64
						Probabilities map[string]float64
						Legend        map[string]json.RawMessage
					}
				}
			}
		}
	}
	if err := readJSON("../../docs/benchmarks/data/typesafe-594ba47.json", &run); err != nil {
		t.Fatal(err)
	}
	if run.Schema != "go-system-one-typesafe-benchmark-v1" || run.Revision != "594ba476bb33d7a38b8be482687b075ecdb2238d" || len(run.Binary) != 64 || len(run.Model) != 64 || len(run.Cases) != 4 {
		t.Fatal("invalid TypeSafe provenance")
	}
	for i, c := range run.Cases {
		if c.Label != []string{"Noul", "Choice", "Score", "Noul + choice + score"}[i] || c.Status != "complete" || c.N != 5 || len(c.Trials) != 5 {
			t.Fatal("incomplete TypeSafe workload")
		}
		if err := validateSamples(c.sampleData); err != nil {
			t.Fatal(err)
		}
		var compact bytes.Buffer
		if err := json.Compact(&compact, c.Request); err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(compact.Bytes())
		if hex.EncodeToString(hash[:]) != c.RequestHash {
			t.Fatal("request hash mismatch")
		}
		var request struct {
			Questions map[string]struct {
				Type     string
				Criteria json.RawMessage
			}
		}
		if err := json.Unmarshal(c.Request, &request); err != nil {
			t.Fatal(err)
		}
		var times []float64
		for _, trial := range c.Trials {
			times = append(times, trial.HTTPMS)
			if trial.Before.Temperature > 55 || trial.MaxTemperature >= 83 {
				t.Fatal("thermal guard breached")
			}
			if len(trial.Response.Answers) != len(request.Questions) {
				t.Fatal("missing answers")
			}
			for name, q := range request.Questions {
				a := trial.Response.Answers[name]
				if a.Type != q.Type {
					t.Fatal("wrong answer type")
				}
				if q.Type == "noul" {
					if a.Noul == nil || *a.Noul < 0 || *a.Noul > 1 || a.Confidence != nil {
						t.Fatal("invalid noul")
					}
					continue
				}
				if a.Confidence == nil || *a.Confidence < 0 || *a.Confidence > 1 {
					t.Fatal("invalid confidence")
				}
				sum := 0.0
				for _, p := range a.Probabilities {
					if p < 0 || p > 1 {
						t.Fatal("invalid probability")
					}
					sum += p
				}
				if !near(sum, 1) {
					t.Fatal("unnormalised probabilities")
				}
				if q.Type == "choice" {
					var criteria map[string]json.RawMessage
					if err := json.Unmarshal(q.Criteria, &criteria); err != nil {
						t.Fatal(err)
					}
					if len(a.Probabilities) != len(criteria) {
						t.Fatal("choice count mismatch")
					}
					for key := range criteria {
						if _, ok := a.Probabilities[key]; !ok {
							t.Fatal("missing choice")
						}
					}
					if _, ok := criteria[a.Choice]; !ok {
						t.Fatal("unknown choice")
					}
				} else {
					var levels []json.RawMessage
					if err := json.Unmarshal(q.Criteria, &levels); err != nil {
						t.Fatal(err)
					}
					if len(a.Probabilities) != len(levels) || len(a.Legend) != len(levels) {
						t.Fatal("score levels mismatch")
					}
					expectation := 0.0
					for j, level := range levels {
						key := fmt.Sprint(j)
						expectation += float64(j) * a.Probabilities[key]
						var got, want any
						if json.Unmarshal(a.Legend[key], &got) != nil || json.Unmarshal(level, &want) != nil || !reflect.DeepEqual(got, want) {
							t.Fatal("legend mismatch")
						}
					}
					if a.Score == nil || !near(*a.Score, expectation) {
						t.Fatal("score is not expectation")
					}
				}
			}
		}
		sort.Float64s(times)
		if !reflect.DeepEqual(times, c.Samples) {
			t.Fatal("TypeSafe trials differ from summary")
		}
	}
}
