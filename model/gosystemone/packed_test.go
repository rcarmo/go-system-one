package gosystemone

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/rcarmo/go-system-one/loader/tokenizer"
	"github.com/rcarmo/go-system-one/model"
)

type batchFake struct {
	fixedScorer
	batchCalls int
	bad        bool
}

func (s *batchFake) ScoreSplitContexts(_ context.Context, _ []int, contexts [][]int, branch Branch, _ bool) ([][]float32, error) {
	s.batchCalls++
	if s.bad {
		return nil, nil
	}
	out := make([][]float32, len(contexts))
	for i := range out {
		out[i] = make([]float32, len(branch.CandidateTokens))
		for j, token := range branch.CandidateTokens {
			out[i][j] = float32(token)
		}
	}
	return out, nil
}

func TestEnginePackedMatchesScalar(t *testing.T) {
	s := &batchFake{}
	e := &Engine{Tokenizer: runeTokenizer{}, Scorer: s, BOSToken: 2}
	r := Request{Schema: json.RawMessage(`{"urgent":{"type":"boolean","description":"urgent"}}`), Contexts: []string{"a", "longer context", "z"}, Mode: ModeTree}
	got, err := e.Decide(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	want, err := (&Engine{Tokenizer: runeTokenizer{}, Scorer: &fixedScorer{}, BOSToken: 2}).Decide(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if s.batchCalls != 1 || s.calls != 0 || !reflect.DeepEqual(got.Results, want.Results) || got.Usage != want.Usage || got.Timings.Rounds != want.Timings.Rounds {
		t.Fatalf("batch mismatch: got=%+v want=%+v calls=%d/%d", got, want, s.batchCalls, s.calls)
	}
	s.bad = true
	if _, err := e.Decide(context.Background(), r); err == nil {
		t.Fatal("malformed batch accepted")
	}
}

func TestEnginePackedFallback(t *testing.T) {
	for _, request := range []Request{
		{Schema: json.RawMessage(`{"urgent":{"type":"boolean","description":"classification"}}`), Contexts: []string{"one"}, Mode: ModeTree},
		{Schema: json.RawMessage(`{"a":{"type":"boolean","description":"classification"},"b":{"type":"boolean","description":"classification"}}`), Contexts: []string{"one", "two"}, Mode: ModeTree},
		{Schema: json.RawMessage(`{"severity":{"type":"enum","description":"classification","choices":["low","medium","high"]}}`), Contexts: []string{"one", "two"}, Mode: ModeAuto, TreeMax: 2},
	} {
		s := &batchFake{}
		if _, err := (&Engine{Tokenizer: runeTokenizer{}, Scorer: s, BOSToken: 2}).Decide(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		if s.batchCalls != 0 || s.calls == 0 {
			t.Fatalf("fallback calls batch=%d scalar=%d", s.batchCalls, s.calls)
		}
	}
}

func TestGoSystemOnePackedReleasedModelMatchesSerial(t *testing.T) {
	path, dir := os.Getenv("GO_SYSTEM_ONE_MODEL"), os.Getenv("GO_SYSTEM_ONE_TOKENIZER_DIR")
	if path == "" || dir == "" {
		t.Skip("set GO_SYSTEM_ONE_MODEL and GO_SYSTEM_ONE_TOKENIZER_DIR")
	}
	tok, err := tokenizer.LoadWithConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	m, err := model.LoadGemma4GGUFAsLlama(path)
	if err != nil {
		t.Fatal(err)
	}
	m.Tok = tok
	g, err := model.NewGemma4NVIDIA(m)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	s := &Gemma4NVIDIAScorer{Model: m, GPU: g}
	defer s.Close()
	e := &Engine{Tokenizer: tok, Scorer: s, BOSToken: m.Config.BOSTokenID}
	r := Request{Instructions: "Answer from the context.", Schema: json.RawMessage(`{"urgent":{"type":"boolean","description":"Does this need urgent handling?"}}`), Contexts: []string{
		"The production service is down for every customer.",
		"The planned maintenance completed successfully. All services are healthy.",
		"There is an active fire in the server room.",
		"Please update the documentation next month.",
		"An expired certificate has blocked all payments since this morning. Customers cannot check out.",
		"The routine nightly backup finished without errors.",
	}, Mode: ModeTree}
	want, err := e.Decide(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := CompileSchema(r.Schema, r.Instructions)
	if err != nil {
		t.Fatal(err)
	}
	field, err := CompileField(tok, schema.Inputs[0], ModeTree, DefaultTreeMax)
	if err != nil {
		t.Fatal(err)
	}
	shared := append([]int{m.Config.BOSTokenID}, tok.Encode(e.renderShared(schema.SystemText))...)
	contexts := make([][]int, len(r.Contexts))
	branch := Branch{Tokens: append(append([]int(nil), field.Suffix...), field.Nodes[0].Prefix...), CandidateTokens: field.Nodes[0].Options}
	for i, text := range r.Contexts {
		contexts[i] = tok.Encode(e.renderContext(text))
	}
	s.PackedTokenRows = 0
	serialLogits, err := s.ScoreSplitContexts(context.Background(), shared, contexts, branch, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, budget := range []int{128, 256, 512} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			s.PackedTokenRows = budget
			got, err := e.Decide(context.Background(), r)
			if err != nil {
				t.Fatal(err)
			}
			maxDiff := 0.0
			for i, result := range got.Results {
				if !reflect.DeepEqual(result.Decision, want.Results[i].Decision) {
					t.Fatalf("entry %d decision mismatch", i)
				}
				for j, candidate := range result.Fields["urgent"].Candidates {
					d := math.Abs(candidate.Probability - want.Results[i].Fields["urgent"].Candidates[j].Probability)
					maxDiff = math.Max(maxDiff, d)
					if d > 1e-6 {
						t.Fatalf("entry %d candidate %d probability diff %g", i, j, d)
					}
				}
			}
			packedLogits, err := s.ScoreSplitContexts(context.Background(), shared, contexts, branch, true)
			if err != nil {
				t.Fatal(err)
			}
			maxLogit := 0.0
			for i := range serialLogits {
				for j, expected := range serialLogits[i] {
					d := math.Abs(float64(packedLogits[i][j] - expected))
					maxLogit = math.Max(maxLogit, d)
					if d > math.Max(.01, .005*math.Abs(float64(expected))) {
						t.Fatalf("logit[%d][%d] got=%g want=%g delta=%g", i, j, packedLogits[i][j], expected, d)
					}
				}
			}
			// Same checkpoint kernels must not depend on sibling order. The
			// earlier attention race passed saturated-probability checks alone.
			if !reflect.DeepEqual(packedLogits, serialLogits) {
				t.Fatal("packed logits differ from serial")
			}
			reversed := make([][]int, len(contexts))
			for i := range contexts {
				reversed[i] = contexts[len(contexts)-1-i]
			}
			again, err := s.ScoreSplitContexts(context.Background(), shared, reversed, branch, true)
			if err != nil {
				t.Fatal(err)
			}
			for i := range again {
				if !reflect.DeepEqual(again[i], serialLogits[len(again)-1-i]) {
					t.Fatalf("reordered context %d changed logits", i)
				}
			}
			t.Logf("packed rows=%d max probability delta=%g max logit delta=%g", budget, maxDiff, maxLogit)
		})
	}
}

type treeFake struct {
	fixedScorer
	callsTree int
	bad       int
}

func (s *treeFake) ScoreSplitContextTrees(_ context.Context, _ []int, contexts [][]int, branches []Branch, _ bool) ([][][]float32, error) {
	s.callsTree++
	out := make([][][]float32, len(contexts))
	for i := range out {
		out[i] = make([][]float32, len(branches))
		for j, b := range branches {
			out[i][j] = make([]float32, len(b.CandidateTokens))
			for k, tok := range b.CandidateTokens {
				out[i][j][k] = float32(tok)
			}
		}
	}
	if s.bad == 1 {
		return out[:len(out)-1], nil
	}
	if s.bad == 2 {
		out[0] = nil
	}
	if s.bad == 3 {
		out[0][0] = nil
	}
	return out, nil
}
func TestEngineBatchTreeMatchesScalar(t *testing.T) {
	r := Request{Schema: json.RawMessage(`{"urgent":{"type":"boolean","description":"urgency"},"severity":{"type":"enum","description":"severity","choices":["critical incident","routine maintenance","routine request"]}}`), Contexts: []string{"one", "different longer context"}, Mode: ModeTree}
	s := &treeFake{}
	e := &Engine{Tokenizer: runeTokenizer{}, Scorer: s, BOSToken: 2}
	got, err := e.Decide(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	want, err := (&Engine{Tokenizer: runeTokenizer{}, Scorer: &fixedScorer{}, BOSToken: 2}).Decide(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if s.callsTree != 1 || s.calls != 0 || !reflect.DeepEqual(got.Results, want.Results) || got.Usage != want.Usage || got.Timings.Rounds != want.Timings.Rounds {
		t.Fatal("tree/scalar mismatch")
	}
	for _, bad := range []int{1, 2, 3} {
		s.bad = bad
		if _, err := e.Decide(context.Background(), r); err == nil {
			t.Fatal("malformed tree scores accepted")
		}
	}
}
