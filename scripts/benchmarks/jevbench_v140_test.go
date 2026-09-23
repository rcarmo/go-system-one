package main

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestJevBenchV140PublicEvidence(t *testing.T) {
	const root = "../../docs/benchmarks/data/jevbench-v140-public-20260923/"
	type metrics struct {
		Attempted int     `json:"n_attempted"`
		Correct   int     `json:"n_correct"`
		Valid     int     `json:"n_valid"`
		Accuracy  float64 `json:"accuracy"`
	}
	var report struct {
		All        metrics            `json:"all_public"`
		Tiers      map[string]metrics `json:"per_tier"`
		Score      *float64           `json:"official_jevbench_score"`
		Rank       *int               `json:"rank"`
		Provenance struct {
			Revision string `json:"service_revision"`
			Suite    string `json:"suite_revision"`
			Binary   string `json:"binary_sha256"`
		} `json:"provenance"`
	}
	if err := readJSON(root+"summary.json", &report); err != nil {
		t.Fatal(err)
	}
	if report.Provenance.Revision != "b18ee0d4748bac436999aa72c000e06406c3cce6" || report.Provenance.Suite != "2fa63fa3226cb369795525ed011800f57dcbd894" || len(report.Provenance.Binary) != 64 || report.Score != nil || report.Rank != nil {
		t.Fatal("identity or ranking mismatch")
	}
	f, err := os.Open(root + "records.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	counts := map[string]metrics{}
	seen := map[string]bool{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 65536), 1<<20)
	for scanner.Scan() {
		var r struct {
			ID             string `json:"task_id"`
			Tier           string `json:"tier"`
			Status         string `json:"status"`
			Code           int    `json:"status_code"`
			Correct, Valid bool
			Strict         bool `json:"strict_valid"`
			Renormalized   bool
			Probs          map[string]float64
			MS             float64 `json:"latency_s"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if seen[r.ID] || r.Status != "ok" || r.Code != 200 || !r.Valid || !r.Strict || r.Renormalized || r.MS <= 0 {
			t.Fatalf("invalid record %s", r.ID)
		}
		seen[r.ID] = true
		sum := 0.0
		for _, p := range r.Probs {
			if p < 0 || p > 1 {
				t.Fatal("probability out of range")
			}
			sum += p
		}
		if !near(sum, 1) {
			t.Fatal("probability sum")
		}
		m := counts[r.Tier]
		m.Attempted++
		m.Valid++
		if r.Correct {
			m.Correct++
		}
		counts[r.Tier] = m
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	correct := 0
	for tier, want := range map[string][2]int{"easy": {48, 48}, "original": {72, 71}, "hard": {111, 77}} {
		m := counts[tier]
		if m.Attempted != want[0] || m.Correct != want[1] {
			t.Fatalf("wrong count %s", tier)
		}
		r := report.Tiers[tier]
		if r.Attempted != m.Attempted || r.Valid != m.Valid || r.Correct != m.Correct || !near(r.Accuracy, float64(m.Correct)/float64(m.Attempted)) {
			t.Fatal("aggregate mismatch")
		}
		correct += m.Correct
	}
	if len(seen) != 231 || correct != 196 || report.All.Attempted != 231 || report.All.Correct != 196 || !near(report.All.Accuracy, 196.0/231) {
		t.Fatal("pooled mismatch")
	}
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"84.85% public-subset accuracy (196/231)", "https://github.com/fstandhartinger/jevbench/tree/v1.4.0", "docs/benchmarks/jevbench-v140-public.md"} {
		if !strings.Contains(string(readme), s) {
			t.Fatalf("README missing %q", s)
		}
	}
}
