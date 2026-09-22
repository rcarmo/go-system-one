package main

import (
	"fmt"
	"math"
	"sort"
)

type batchCell struct {
	Size    int     `json:"size"`
	Rows    int     `json:"rows"`
	Status  string  `json:"status"`
	Median  float64 `json:"median_ms"`
	Min     float64 `json:"min_ms"`
	Max     float64 `json:"max_ms"`
	Samples []struct {
		MS float64 `json:"handler_ms"`
	} `json:"samples"`
}
type batchSweep struct {
	Revision string      `json:"revision"`
	Mode     string      `json:"mode"`
	Cases    []batchCell `json:"cases"`
}

func validateBatchCell(c batchCell) error {
	if c.Size < 1 || c.Size > 256 || len(c.Samples) != 3 {
		return fmt.Errorf("batch size %d needs three samples", c.Size)
	}
	times := make([]float64, len(c.Samples))
	for i, s := range c.Samples {
		if s.MS <= 0 || math.IsNaN(s.MS) || math.IsInf(s.MS, 0) {
			return fmt.Errorf("invalid batch sample")
		}
		times[i] = s.MS
	}
	sort.Float64s(times)
	if !near(c.Min, times[0]) || !near(c.Median, median(times)) || !near(c.Max, times[len(times)-1]) {
		return fmt.Errorf("batch size %d summary differs from samples", c.Size)
	}
	return nil
}
func completedBatches(s batchSweep) ([]batchCell, error) {
	if s.Mode != "automatic" || len(s.Revision) != 40 {
		return nil, fmt.Errorf("invalid automatic sweep provenance")
	}
	seen := map[int]bool{}
	var out []batchCell
	for _, c := range s.Cases {
		if c.Size < 1 || c.Size > 256 || seen[c.Size] {
			return nil, fmt.Errorf("invalid/duplicate sweep size %d", c.Size)
		}
		seen[c.Size] = true
		switch c.Status {
		case "complete":
			if err := validateBatchCell(c); err != nil {
				return nil, err
			}
			out = append(out, c)
		case "interrupted", "aborted", "running": // Preserve incomplete observations, never chart their partial median.
		default:
			return nil, fmt.Errorf("unrecognised batch status %q", c.Status)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no complete batch cells")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Size < out[j].Size })
	return out, nil
}

func renderAutomaticBatches(s batchSweep) (string, error) {
	cells, err := completedBatches(s)
	if err != nil {
		return "", err
	}
	const width = 1040
	height := 165 + 70*len(cells)
	b := chartStart("Automatic multi-field batching", "RTX 3060 · one boolean + three-choice enum · three warm samples per size", width, height)
	left, plotW := 180., 710.
	maxMS := cells[0].Max
	for _, c := range cells {
		maxMS = math.Max(maxMS, c.Max)
	}
	maxMS = math.Ceil(maxMS/500) * 500
	bottom := 105. + 70.*float64(len(cells))
	for tick := 0.; tick <= maxMS; tick += maxMS / 4 {
		x := left + tick/maxMS*plotW
		fmt.Fprintf(b, `<line class="grid" x1="%.1f" y1="94" x2="%.1f" y2="%.1f"/><text class="muted" x="%.1f" y="%.1f" text-anchor="middle" font-size="12">%.2f s</text>`, x, x, bottom, x, bottom+22, tick/1000)
	}
	for i, c := range cells {
		y := 105. + 70.*float64(i)
		w := c.Median / maxMS * plotW
		fmt.Fprintf(b, `<text x="165" y="%.1f" text-anchor="end" font-size="14">%d entries</text><rect class="good" x="%.1f" y="%.1f" width="%.1f" height="28" rx="4"/><text x="%.1f" y="%.1f" font-size="13">%.3f s</text>`, y+20, c.Size, left, y, w, left+w+8, y+20, c.Median/1000)
		x1, x2 := left+c.Min/maxMS*plotW, left+c.Max/maxMS*plotW
		fmt.Fprintf(b, `<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/><text class="muted" x="%.1f" y="%.1f" font-size="12">%.2f entries/s</text>`, x1, y+34, x2, y+34, left, y+52, 1000*float64(c.Size)/c.Median)
	}
	fmt.Fprintf(b, `<text class="muted" x="40" y="%d" font-size="12">Handler median with min–max line; completed measurements only; source %s</text></svg>`, height-12, s.Revision[:7])
	return b.String(), nil
}

func renderMultiFieldComparison(cells []batchCell) (string, error) {
	grouped := map[int]map[int]batchCell{}
	for _, c := range cells {
		if err := validateBatchCell(c); err != nil {
			return "", err
		}
		if c.Rows != 0 && c.Rows != 512 {
			return "", fmt.Errorf("unexpected paired row budget")
		}
		if grouped[c.Size] == nil {
			grouped[c.Size] = map[int]batchCell{}
		}
		if _, ok := grouped[c.Size][c.Rows]; ok {
			return "", fmt.Errorf("duplicate paired result")
		}
		grouped[c.Size][c.Rows] = c
	}
	sizes := make([]int, 0, len(grouped))
	maxMS := 0.
	for size, g := range grouped {
		if len(g) != 2 {
			return "", fmt.Errorf("unpaired size %d", size)
		}
		sizes = append(sizes, size)
		for _, c := range g {
			maxMS = math.Max(maxMS, c.Max)
		}
	}
	if len(sizes) == 0 {
		return "", fmt.Errorf("empty comparison")
	}
	sort.Ints(sizes)
	maxMS = math.Ceil(maxMS/1000) * 1000
	height := 170 + 110*len(sizes)
	b := chartStart("Multi-field: serial versus packed", "Earlier paired run · identical requests/binary · three warm samples · RTX 3060", 1040, height)
	left, plotW := 200., 690.
	bottom := 100. + 110.*float64(len(sizes))
	for tick := 0.; tick <= maxMS; tick += maxMS / 4 {
		x := left + tick/maxMS*plotW
		fmt.Fprintf(b, `<line class="grid" x1="%.1f" y1="95" x2="%.1f" y2="%.1f"/><text class="muted" x="%.1f" y="%.1f" text-anchor="middle" font-size="12">%.2f s</text>`, x, x, bottom, x, bottom+22, tick/1000)
	}
	for i, size := range sizes {
		y := 105. + 110.*float64(i)
		for j, rows := range []int{0, 512} {
			c := grouped[size][rows]
			class, label := "primary", "serial"
			if rows > 0 {
				class, label = "good", "packed"
			}
			yy := y + 36.*float64(j)
			w := c.Median / maxMS * plotW
			fmt.Fprintf(b, `<text x="185" y="%.1f" text-anchor="end" font-size="13">%d entries · %s</text><rect class="%s" x="%.1f" y="%.1f" width="%.1f" height="26" rx="4"/><text x="%.1f" y="%.1f" font-size="13">%.3f s</text>`, yy+18, size, label, class, left, yy, w, left+w+8, yy+18, c.Median/1000)
		}
		fmt.Fprintf(b, `<text class="muted" x="200" y="%.1f" font-size="12">%.2f× faster</text>`, y+88, grouped[size][0].Median/grouped[size][512].Median)
	}
	b.WriteString(fmt.Sprintf(`<text class="muted" x="40" y="%d" font-size="12">Fixed two-field workload; precision trade-off is documented separately; no extrapolated batches.</text></svg>`, height-12))
	return b.String(), nil
}
