package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBenchmarkDataValidation(t *testing.T) {
	valid := benchmarkData{
		Schema:     "go-system-one-benchmarks-v1",
		Comparison: []comparison{{Label: "current", RepresentativeMS: 2, MinMS: 1, MaxMS: 3, Basis: "median"}},
		Workloads:  []workload{{Label: "baseline", MedianMS: 2, MinMS: 1, MaxMS: 3}},
	}
	if err := validateBenchmarkData(valid); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.Comparison = []comparison{{Label: "current", RepresentativeMS: 4, MinMS: 1, MaxMS: 3, Basis: "median"}}
	if err := validateBenchmarkData(invalid); err == nil {
		t.Fatal("out-of-range representative accepted")
	}
}

func TestSampleStatistics(t *testing.T) {
	values := []float64{1, 2, 3, 4}
	if got := median(values); got != 2.5 {
		t.Fatalf("median=%g", got)
	}
	if got := nearestRank(values, .95); got != 4 {
		t.Fatalf("p95=%g", got)
	}
	valid := sampleData{N: 4, Min: 1, Median: 2.5, P95: 4, P99: 4, Max: 4, Samples: values}
	if err := validateSamples(valid); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.Samples = []float64{2, 1, 3, 4}
	if err := validateSamples(invalid); err == nil {
		t.Fatal("unsorted samples accepted")
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	data := benchmarkData{
		Schema:     "go-system-one-benchmarks-v1",
		Comparison: []comparison{{Label: "Current Q6-staged Go/PTX", RepresentativeMS: 81, MinMS: 80, MaxMS: 82, Basis: "median"}},
		Workloads:  []workload{{Label: "Baseline", MedianMS: 81, MinMS: 80, MaxMS: 82}},
	}
	samples := sampleData{N: 4, Min: 80, Median: 81.5, P95: 83, P99: 83, Max: 83, Samples: []float64{80, 81, 82, 83}}
	for name, output := range map[string]string{
		"distribution": renderDistribution(samples),
		"comparison":   renderComparison(data),
		"workloads":    renderWorkloads(data),
	} {
		if !strings.HasPrefix(output, `<svg xmlns="http://www.w3.org/2000/svg"`) || !strings.Contains(output, "prefers-color-scheme:dark") || !strings.HasSuffix(output, "</svg>") {
			t.Fatalf("invalid %s SVG", name)
		}
	}
	if renderDistribution(samples) != renderDistribution(samples) {
		t.Fatal("distribution render changed across calls")
	}
	comparison := renderComparison(data)
	if !strings.Contains(comparison, "From prototype to hand-tuned Go") || !strings.Contains(comparison, "Single-boolean fixture") || !strings.Contains(comparison, `class="good"`) {
		t.Fatal("comparison chart does not describe the implementation sequence")
	}
}

func TestCollectRejectsDecisionDrift(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		urgent := calls != 2
		_, _ = fmt.Fprintf(w, `{"results":[{"decision":{"urgent":%t}}],"timings":{"total_ms":%d}}`, urgent, calls)
	}))
	defer server.Close()

	dir := t.TempDir()
	requestPath := filepath.Join(dir, "request.json")
	if err := os.WriteFile(requestPath, []byte(`{"contexts":["fixture"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	err := collect([]string{"-url", server.URL, "-request", requestPath, "-out", filepath.Join(dir, "samples.json"), "-warmup", "0", "-n", "2"})
	if err == nil || !strings.Contains(err.Error(), "decision changed") {
		t.Fatalf("error=%v", err)
	}
}

func TestCollect(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("request method=%s content-type=%s", r.Method, r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"decision":{"urgent":true}}],"timings":{"total_ms":` + string(rune('0'+calls)) + `}}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	requestPath := filepath.Join(dir, "request.json")
	outPath := filepath.Join(dir, "samples.json")
	if err := os.WriteFile(requestPath, []byte(`{"contexts":["fixture"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := collect([]string{"-url", server.URL, "-request", requestPath, "-out", outPath, "-warmup", "1", "-n", "3", "-timeout", time.Second.String()}); err != nil {
		t.Fatal(err)
	}
	var got sampleData
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := sampleData{N: 3, Min: 2, Median: 3, P95: 4, P99: 4, Max: 4, Samples: []float64{2, 3, 4}, Decisions: []map[string]any{{"urgent": true}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
}
