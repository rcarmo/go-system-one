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
			results[i] = map[string]any{
				"decision": map[string]any{"urgent": i == 0},
				"fields": map[string]any{
					"urgent": map[string]any{"value": i == 0, "probability": .875, "scored_nodes": 1, "tree": true},
				},
				"usage": map[string]int{"context_tokens": 4, "scored_rows": 2},
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
