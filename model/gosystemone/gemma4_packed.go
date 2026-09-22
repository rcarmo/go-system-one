package gosystemone

import (
	"context"
	"fmt"

	"github.com/rcarmo/go-system-one/model"
)

// ScoreSplitContexts executes one shared decision branch for independent
// contexts. PackedTokenRows=0 retains the scalar reference for comparison.
func (s *Gemma4NVIDIAScorer) ScoreSplitContexts(ctx context.Context, shared []int, contexts [][]int, branch Branch, allowCache bool) ([][]float32, error) {
	if s == nil || s.Model == nil || s.GPU == nil || ctx == nil || len(shared) == 0 || len(contexts) == 0 || len(contexts) > MaxContexts {
		return nil, fmt.Errorf("invalid packed decision scorer/input")
	}
	budget := s.PackedTokenRows
	if budget < 0 || budget > model.Gemma4PackedRows {
		return nil, fmt.Errorf("packed token rows must be 0..%d", model.Gemma4PackedRows)
	}
	paths := make([][]int, len(contexts))
	fallback := budget == 0
	for i, tokens := range contexts {
		if len(tokens) == 0 {
			return nil, fmt.Errorf("empty context %d", i)
		}
		prompt := append(append([]int(nil), shared...), tokens...)
		if _, err := validateGemma4ScoringInput(ctx, s.Model, prompt, []Branch{branch}, nil); err != nil {
			return nil, err
		}
		paths[i] = append(append([]int(nil), tokens...), branch.Tokens...)
		fallback = fallback || len(paths[i]) > budget
	}
	out := make([][]float32, len(contexts))
	if fallback {
		for i, tokens := range contexts {
			scores, err := s.ScoreSplitContext(ctx, shared, tokens, []Branch{branch}, allowCache)
			if err != nil {
				return nil, err
			}
			out[i] = scores[0]
		}
		return out, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := s.cached
	if !allowCache || !cacheCanReuse(shared, s.cachedTokens, len(shared), s.cachedTrunkCap, prefix != nil) {
		if s.cached != nil {
			s.cached.Close()
			s.cached = nil
		}
		var err error
		prefix, err = s.GPU.PrefillPreparedCapacity(ctx, shared, len(shared), 1, 1)
		if err != nil {
			return nil, err
		}
		if allowCache {
			s.cached = prefix
			s.cachedTokens = append([]int(nil), shared...)
			s.cachedTrunkCap = len(shared)
		} else {
			defer prefix.Close()
		}
	}
	for start := 0; start < len(paths); {
		end, rows := start, 0
		for end < len(paths) && len(paths[end]) <= budget-rows {
			rows += len(paths[end])
			end++
		}
		candidates := make([][]int, end-start)
		for i := range candidates {
			candidates[i] = branch.CandidateTokens
		}
		scores, err := s.GPU.ScorePrefixedPacked(ctx, prefix, paths[start:end], candidates)
		if err != nil {
			return nil, fmt.Errorf("packed contexts [%d,%d): %w", start, end, err)
		}
		copy(out[start:end], scores)
		start = end
	}
	return out, nil
}

// ScoreSplitContextTrees packs complete context/branch groups. Every branch in
// a group sees its context but no sibling branch. Larger groups fall back.
func (s *Gemma4NVIDIAScorer) ScoreSplitContextTrees(ctx context.Context, shared []int, contexts [][]int, branches []Branch, allowCache bool) ([][][]float32, error) {
	if s == nil || s.GPU == nil || s.Model == nil || ctx == nil || len(contexts) < 1 || len(contexts) > MaxContexts || len(branches) < 1 || len(shared) < 1 {
		return nil, fmt.Errorf("invalid tree batch input")
	}
	if len(branches) == 1 {
		rows, err := s.ScoreSplitContexts(ctx, shared, contexts, branches[0], allowCache)
		if err != nil {
			return nil, err
		}
		out := make([][][]float32, len(rows))
		for i := range rows {
			out[i] = [][]float32{rows[i]}
		}
		return out, nil
	}
	budget := s.PackedTokenRows
	if budget < 0 || budget > model.Gemma4PackedRows {
		return nil, fmt.Errorf("invalid packed token rows")
	}
	cost := 0
	paths, selected := make([][]int, len(branches)), make([][]int, len(branches))
	for j, b := range branches {
		cost += len(b.Tokens)
		paths[j] = b.Tokens
		selected[j] = b.CandidateTokens
	}
	fallback := budget == 0 || len(branches) > 256
	for _, c := range contexts {
		prompt := append(append([]int(nil), shared...), c...)
		if len(c) == 0 {
			return nil, fmt.Errorf("empty context")
		}
		if _, err := validateGemma4ScoringInput(ctx, s.Model, prompt, branches, nil); err != nil {
			return nil, err
		}
		fallback = fallback || len(c)+cost > budget
	}
	out := make([][][]float32, len(contexts))
	if fallback {
		for i, c := range contexts {
			v, err := s.ScoreSplitContext(ctx, shared, c, branches, allowCache)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := s.cached
	if !allowCache || !cacheCanReuse(shared, s.cachedTokens, len(shared), s.cachedTrunkCap, prefix != nil) {
		if s.cached != nil {
			s.cached.Close()
			s.cached = nil
		}
		var err error
		prefix, err = s.GPU.PrefillPreparedCapacity(ctx, shared, len(shared), 1, 1)
		if err != nil {
			return nil, err
		}
		if allowCache {
			s.cached = prefix
			s.cachedTokens = append([]int(nil), shared...)
			s.cachedTrunkCap = len(shared)
		} else {
			defer prefix.Close()
		}
	}
	for start := 0; start < len(contexts); {
		end, rows := start, 0
		for end < len(contexts) && len(contexts[end])+cost <= budget-rows && (end-start+1)*len(branches) <= 256 {
			rows += len(contexts[end]) + cost
			end++
		}
		scores, err := s.GPU.ScorePrefixedTreeGroups(ctx, prefix, contexts[start:end], paths, selected)
		if err != nil {
			return nil, err
		}
		copy(out[start:end], scores)
		start = end
	}
	return out, nil
}
