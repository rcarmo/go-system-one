package main

import (
	"bufio"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestLongContextRecheckKeepsOriginalRequests(t *testing.T) {
	const root = "../../docs/benchmarks/data/"
	var report struct {
		N       int    `json:"n"`
		Valid   int    `json:"strict_valid"`
		Correct int    `json:"n_correct"`
		Binary  string `json:"binary_sha256"`
		Cases   []struct {
			ID      string          `json:"task_id"`
			Request json.RawMessage `json:"request"`
			Status  int             `json:"status"`
			MS      float64         `json:"http_ms"`
			Before  struct {
				Temperature int `json:"temperature_c"`
			} `json:"before"`
			MaxTemperature int `json:"max_temperature_c"`
			Tokens         int `json:"total_visible_tokens"`
			Scored         struct {
				Valid, Correct bool
				Strict         bool `json:"strict_valid"`
			} `json:"scored"`
		} `json:"cases"`
	}
	if err := readJSON(root+"long-context-20260923/recheck.json", &report); err != nil {
		t.Fatal(err)
	}
	if report.N != 36 || report.Valid != 36 || report.Correct != 27 || len(report.Binary) != 64 || len(report.Cases) != 36 {
		t.Fatal("invalid recheck header")
	}
	f, err := os.Open(root + "jevbench-public-20260923/raw-evidence.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	originals := map[string]json.RawMessage{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 65536), 1<<20)
	for scanner.Scan() {
		var r struct {
			ID       string `json:"task_id"`
			Evidence struct {
				Status  int             `json:"http_status"`
				Request json.RawMessage `json:"request"`
			} `json:"evidence"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if r.Evidence.Status == 400 {
			originals[r.ID] = r.Evidence.Request
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	correct := 0
	for _, c := range report.Cases {
		if seen[c.ID] || c.Status != 200 || !c.Scored.Valid || !c.Scored.Strict || c.MS <= 0 || c.Before.Temperature > 55 || c.MaxTemperature >= 83 || c.Tokens <= 2048 || c.Tokens > 3937 {
			t.Fatalf("invalid case %s", c.ID)
		}
		seen[c.ID] = true
		if c.Scored.Correct {
			correct++
		}
		var got, want any
		if json.Unmarshal(c.Request, &got) != nil || json.Unmarshal(originals[c.ID], &want) != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("request changed %s", c.ID)
		}
	}
	if correct != report.Correct || len(seen) != len(originals) {
		t.Fatal("incomplete or incorrect subset")
	}
}
