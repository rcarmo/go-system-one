package webui

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// TestBrowserServer is a loopback-only fixture for Playwright. Normal Go tests
// skip it; browser/playwright.config.ts enables it explicitly.
func TestBrowserServer(t *testing.T) {
	if os.Getenv("WEBUI_BROWSER_TEST") != "1" {
		t.Skip("Playwright fixture only")
	}
	mux := http.NewServeMux()
	RegisterGoSystemOne(mux, GoSystemOneConfig{ModelID: "fixture-model", Backend: "fixture", Device: "synthetic", MaxContexts: 256, MaxFields: 32, MaxCandidates: 255})
	mux.HandleFunc("/v1/decision", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Contexts []string `json:"contexts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Contexts) == 0 {
			http.Error(w, "bad fixture input", http.StatusBadRequest)
			return
		}
		results := make([]any, len(body.Contexts))
		for i := range body.Contexts {
			urgent := i == 0
			urgentProbability := .875
			severity := "high"
			severityProbabilities := []float64{.10, .20, .70}
			if !urgent {
				urgentProbability = .80
				severity = "low"
				severityProbabilities = []float64{.75, .20, .05}
			}
			results[i] = map[string]any{
				"decision": map[string]any{"urgent": urgent, "severity": severity},
				"fields": map[string]any{
					"urgent": map[string]any{"value": urgent, "probability": urgentProbability, "candidates": []any{
						map[string]any{"value": true, "probability": map[bool]float64{true: .875, false: .20}[urgent], "selected": urgent},
						map[string]any{"value": false, "probability": map[bool]float64{true: .125, false: .80}[urgent], "selected": !urgent},
					}, "scored_nodes": 1, "tree": true},
					"severity": map[string]any{"value": severity, "probability": severityProbabilities[map[string]int{"low": 0, "medium": 1, "high": 2}[severity]], "candidates": []any{
						map[string]any{"value": "low", "probability": severityProbabilities[0], "selected": severity == "low"},
						map[string]any{"value": "medium", "probability": severityProbabilities[1], "selected": severity == "medium"},
						map[string]any{"value": "high", "probability": severityProbabilities[2], "selected": severity == "high"},
					}, "scored_nodes": 1, "tree": true},
				},
				"usage": map[string]int{"context_tokens": 4 + i, "scored_rows": 5},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object":  "decision",
			"model":   "fixture-model",
			"results": results,
			"timings": map[string]any{"total_ms": 3.5, "prefill_ms": 1.0, "scoring_ms": 2.5, "per_decision_ms": 1.75, "rounds": 1},
		})
	})
	mux.HandleFunc("/v1/systemone", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			State     json.RawMessage `json:"state"`
			Questions map[string]struct {
				Type string `json:"type"`
			} `json:"questions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.State) == 0 || len(body.Questions) == 0 {
			http.Error(w, "bad fixture input", http.StatusBadRequest)
			return
		}
		answers := make(map[string]any)
		for name, q := range body.Questions {
			switch q.Type {
			case "noul":
				answers[name] = map[string]any{"type": "noul", "noul": .875}
			case "choice":
				answers[name] = map[string]any{"type": "choice", "choice": "operations", "confidence": .70, "probabilities": map[string]float64{"operations": .85, "billing": .15}}
			case "score":
				answers[name] = map[string]any{"type": "score", "score": 1.30, "confidence": .65, "probabilities": map[string]float64{"0": .20, "1": .30, "2": .50}, "legend": map[string]string{"0": "Routine request", "1": "Degraded service", "2": "Total outage"}}
			default:
				http.Error(w, "unsupported fixture question", http.StatusBadRequest)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"model": "fixture-model", "answers": answers, "usage": map[string]int{"input_tokens": 176, "output_tokens": 120}})
	})
	listener, err := net.Listen("tcp", "127.0.0.1:18181")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("browser fixture: %v", err)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	_ = srv.Close()
}
