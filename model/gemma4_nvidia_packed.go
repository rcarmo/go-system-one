package model

import (
	"context"
	"fmt"
	"math"

	nvidia "github.com/rcarmo/go-system-one/backends/nvidia/runtime"
	gemmacfg "github.com/rcarmo/go-system-one/model/gemma"
)

// Gemma4PackedRows bounds total real token rows in a packed transformer pass.
const Gemma4PackedRows = 512

type gemma4PrefillSegment struct{ offset, length int }

type gemma4DeviceWork struct {
	buffers []*nvidia.Buffer
	err     error
}

func (w *gemma4DeviceWork) alloc(elements int) *nvidia.Buffer {
	if w.err != nil {
		return nil
	}
	var b *nvidia.Buffer
	b, w.err = nvidia.Malloc(elements)
	if w.err == nil {
		w.buffers = append(w.buffers, b)
	}
	return b
}

func (w *gemma4DeviceWork) free() {
	for _, b := range w.buffers {
		b.Free()
	}
}

// deviceView borrows storage; the returned view must never be freed.
func deviceView(b *nvidia.Buffer, offset, count int) *nvidia.Buffer {
	return &nvidia.Buffer{Ptr: b.Ptr + nvidia.CUdeviceptr(offset*4), Size: count * 4}
}

func (g *Gemma4NVIDIA) prefillAttentionSegments(out, q, k, v, rope *nvidia.Buffer, arena *gemma4NVIDIAKVArena, layer, rows, pos0, window, hd, rot int, segments []gemma4PrefillSegment) error {
	m := g.model
	kvHeads := gemmacfg.LayerKVHeads(m.Config, layer)
	qDim, kvDim := m.Config.NumHeads*hd, kvHeads*hd
	if len(segments) == 0 {
		segments = []gemma4PrefillSegment{{length: rows}}
	}
	for _, segment := range segments {
		qq := deviceView(q, segment.offset*qDim, segment.length*qDim)
		kk := deviceView(k, segment.offset*kvDim, segment.length*kvDim)
		vv := deviceView(v, segment.offset*kvDim, segment.length*kvDim)
		oo := deviceView(out, segment.offset*qDim, segment.length*qDim)
		if err := nvidia.RoPEPartialSequenceBuffer(qq, rope, segment.length, pos0, m.Config.NumHeads, hd, rot); err != nil {
			return err
		}
		if err := nvidia.RoPEPartialSequenceBuffer(kk, rope, segment.length, pos0, kvHeads, hd, rot); err != nil {
			return err
		}
		// Only prefix KV survives across segments. Each attention launch reads
		// its own causal suffix before the next segment replaces that suffix.
		if err := arena.appendTrunkRows(layer, pos0, segment.length, kk, vv); err != nil {
			return err
		}
		if err := nvidia.CausalBatchAttentionBuffer(oo, qq, arena.trunkK[layer], arena.trunkV[layer], segment.length, pos0, pos0+segment.length, window, m.Config.NumHeads, kvHeads, hd, attentionScale(m.Config, hd)); err != nil {
			return err
		}
	}
	return nil
}

// ScorePrefixedPacked scores independent token sequences after one immutable
// prefix. Projections run across all real token rows; attention stays isolated.
// The caller chunks to Gemma4PackedRows and owns the prefix lifetime.
func (g *Gemma4NVIDIA) ScorePrefixedPacked(ctx context.Context, prefix *Gemma4NVIDIAContext, paths, candidates [][]int) ([][]float32, error) {
	if g == nil || ctx == nil {
		return nil, fmt.Errorf("nil NVIDIA packed scorer/context")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || prefix == nil || prefix.closed || prefix.owner != g || prefix.arena == nil {
		return nil, fmt.Errorf("invalid NVIDIA packed prefix")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(paths) == 0 || len(paths) > 256 || len(paths) != len(candidates) {
		return nil, fmt.Errorf("invalid NVIDIA packed paths/candidates")
	}
	segments := make([]gemma4PrefillSegment, len(paths))
	rows, longest := 0, 0
	for i, path := range paths {
		if len(path) == 0 || len(path) > Gemma4PackedRows-rows || len(candidates[i]) == 0 || len(prefix.tokens)+len(path) > 2048 {
			return nil, fmt.Errorf("invalid NVIDIA packed path %d", i)
		}
		for _, ids := range [][]int{path, candidates[i]} {
			for _, id := range ids {
				if id < 0 || id >= g.model.Config.VocabSize {
					return nil, fmt.Errorf("packed path %d token outside vocabulary", i)
				}
			}
		}
		segments[i] = gemma4PrefillSegment{rows, len(path)}
		rows += len(path)
		longest = max(longest, len(path))
	}
	flat := make([]int, 0, rows)
	for _, path := range paths {
		flat = append(flat, path...)
	}
	// One private arena holds the immutable prefix plus a reusable suffix,
	// not a copy of the entire prefix per context.
	arena, err := prefix.arena.cloneTrunk(max(prefix.arena.trunkLen, len(prefix.tokens)+longest), 1, 1)
	if err != nil {
		return nil, err
	}
	defer arena.free()
	h := g.model.Config.HiddenSize
	terminal, err := nvidia.Malloc(len(paths) * h)
	if err != nil {
		return nil, err
	}
	defer terminal.Free()
	// Drain submitted work on cancellation/error before releasing owned buffers.
	defer nvidia.SyncAll()
	if err := g.runPrefillRows(ctx, flat, len(prefix.tokens), arena, nil, terminal, segments); err != nil {
		return nil, err
	}
	out := make([][]float32, len(paths))
	for i := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		out[i], err = g.finishSelectedDevice(deviceView(terminal, i*h, h), candidates[i])
		if err != nil {
			return nil, err
		}
	}
	return out, ctx.Err()
}

func (g *Gemma4NVIDIA) transformSelectedLogits(logits []float32, tokens []int) {
	applyLlamaFinalLogitSoftcap(logits, g.model.Config.FinalLogitSoftcapping)
	for i, token := range tokens {
		for _, suppressed := range g.model.SuppressTokens {
			if token == suppressed {
				logits[i] = float32(math.Inf(-1))
				break
			}
		}
	}
}
