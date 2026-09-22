package main

import (
	"encoding/json"
	"math"
	"testing"

	gso "github.com/rcarmo/go-system-one/model/gosystemone"
)

func example(logits []float32, p float64) trial {
	winner := json.RawMessage(`true`)
	if p < .5 {
		winner = json.RawMessage(`false`)
	}
	return trial{Logits: [][][]float32{{logits}}, Response: gso.Response{Results: []gso.Result{{Fields: map[string]gso.FieldResult{"urgent": {Value: winner, Candidates: []gso.CandidateResult{{Value: json.RawMessage(`true`), Probability: p}, {Value: json.RawMessage(`false`), Probability: 1 - p}}}}}}}}
}
func TestCompareCommonShift(t *testing.T) {
	a, b := example([]float32{1, 2}, .25), example([]float32{11, 12}, .25)
	c, err := compare(a, b, []float64{.5, .9})
	if err != nil {
		t.Fatal(err)
	}
	if c.Raw.Max != 10 || c.Raw.RMS != 10 || c.Centred.Max != 0 || c.ChangedFields != 0 || c.Fields[0].Probability.Max != 0 {
		t.Fatalf("%+v", c)
	}
}
func TestCompareNearTieCrossing(t *testing.T) {
	c, err := compare(example([]float32{1, 1.01}, .49), example([]float32{1.01, 1}, .51), []float64{.5, .9})
	if err != nil {
		t.Fatal(err)
	}
	if c.ChangedFields != 1 || len(c.Fields[0].ThresholdCrossings) != 2 || math.Abs(c.Fields[0].SerialMargin-.02) > 1e-12 || math.Abs(c.Fields[0].Probability.Max-.02) > 1e-12 {
		t.Fatalf("%+v", c)
	}
}
func TestCompareMalformed(t *testing.T) {
	a := example([]float32{1, 2}, .25)
	for _, b := range []trial{{}, example([]float32{1}, .25), example([]float32{1, float32(math.NaN())}, .25)} {
		if _, err := compare(a, b, nil); err == nil {
			t.Fatal("malformed accepted")
		}
	}
}
