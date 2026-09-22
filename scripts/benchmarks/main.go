package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

type benchmarkData struct {
	Schema     string `json:"schema"`
	Fixture    fixture
	Comparison []comparison `json:"comparison"`
	Workloads  []workload   `json:"workloads"`
}

type fixture struct {
	Revision                string `json:"revision"`
	Description             string `json:"description"`
	Model                   string `json:"model"`
	ModelSHA256             string `json:"model_sha256"`
	Device                  string `json:"device"`
	Driver                  string `json:"driver"`
	ResidentProjectionBytes int64  `json:"resident_projection_bytes"`
}

type comparison struct {
	Label            string  `json:"label"`
	RepresentativeMS float64 `json:"representative_ms"`
	MinMS            float64 `json:"min_ms"`
	MaxMS            float64 `json:"max_ms"`
	Basis            string  `json:"basis"`
}

type workload struct {
	Label    string  `json:"label"`
	MedianMS float64 `json:"median_ms"`
	MinMS    float64 `json:"min_ms"`
	MaxMS    float64 `json:"max_ms"`
}

type sampleData struct {
	Revision  string           `json:"revision,omitempty"`
	N         int              `json:"n"`
	Min       float64          `json:"min"`
	Median    float64          `json:"median"`
	P95       float64          `json:"p95"`
	P99       float64          `json:"p99"`
	Max       float64          `json:"max"`
	Samples   []float64        `json:"samples"`
	Decisions []map[string]any `json:"decisions,omitempty"`
}

type responseTiming struct {
	Results []struct {
		Decision map[string]any `json:"decision"`
	} `json:"results"`
	Timings struct {
		TotalMS float64 `json:"total_ms"`
	} `json:"timings"`
}

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: %s {render|collect} [options]", os.Args[0])
	}
	var err error
	switch os.Args[1] {
	case "render":
		err = render(os.Args[2:])
	case "collect":
		err = collect(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fatalf("%v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "benchmarks: "+format+"\n", args...)
	os.Exit(1)
}

func render(args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	dataPath := fs.String("data", "docs/benchmarks/data/current.json", "comparison/workload JSON")
	samplesPath := fs.String("samples", "docs/benchmarks/data/q6-staged-warm.json", "warm sample JSON")
	outDir := fs.String("out", "docs/benchmarks", "SVG output directory")
	batchPath := fs.String("batch-data", "docs/benchmarks/data/q6-staged-batches.json", "automatic multi-field batch sweep")
	pairedPath := fs.String("paired-data", "docs/benchmarks/data/q6-staged-paired.json", "paired multi-field comparison")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var data benchmarkData
	if err := readJSON(*dataPath, &data); err != nil {
		return err
	}
	if err := validateBenchmarkData(data); err != nil {
		return err
	}
	var samples sampleData
	if err := readJSON(*samplesPath, &samples); err != nil {
		return err
	}
	if err := validateSamples(samples); err != nil {
		return err
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}
	var sweep batchSweep
	if err := readJSON(*batchPath, &sweep); err != nil {
		return err
	}
	batchChart, err := renderAutomaticBatches(sweep)
	if err != nil {
		return err
	}
	var paired struct {
		Revision string      `json:"revision"`
		Cases    []batchCell `json:"cases"`
	}
	if err := readJSON(*pairedPath, &paired); err != nil {
		return err
	}
	pairedChart, err := renderMultiFieldComparison(paired.Cases, paired.Revision)
	if err != nil {
		return err
	}
	outputs := map[string]string{
		"automatic-batches.svg":     batchChart,
		"multifield-comparison.svg": pairedChart,
		"warm-latency.svg":          renderDistribution(samples),
		"latency-comparison.svg":    renderComparison(data),
		"workload-matrix.svg":       renderWorkloads(data),
	}
	for name, content := range outputs {
		if err := writeAtomic(filepath.Join(*outDir, name), []byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func collect(args []string) error {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	url := fs.String("url", "http://127.0.0.1:8080/v1/decision", "decision endpoint")
	requestPath := fs.String("request", "docs/benchmarks/request.json", "request JSON")
	outPath := fs.String("out", "dist/benchmarks/nvidia-http.json", "sample output JSON")
	warmup := fs.Int("warmup", 1, "warm-up requests")
	n := fs.Int("n", 100, "measured requests")
	timeout := fs.Duration("timeout", 60*time.Second, "per-request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *warmup < 0 || *n <= 0 || *n > 10000 || *timeout <= 0 {
		return fmt.Errorf("invalid warmup=%d n=%d timeout=%s", *warmup, *n, *timeout)
	}
	body, err := os.ReadFile(*requestPath)
	if err != nil {
		return err
	}
	if !json.Valid(body) {
		return fmt.Errorf("request is not valid JSON")
	}
	client := &http.Client{Timeout: *timeout}
	for i := 0; i < *warmup; i++ {
		if _, _, err := request(client, *url, body); err != nil {
			return fmt.Errorf("warmup %d: %w", i+1, err)
		}
	}
	samples := make([]float64, 0, *n)
	var decisions []map[string]any
	for i := 0; i < *n; i++ {
		total, gotDecisions, err := request(client, *url, body)
		if err != nil {
			return fmt.Errorf("request %d: %w", i+1, err)
		}
		if i == 0 {
			decisions = gotDecisions
		} else if !reflect.DeepEqual(gotDecisions, decisions) {
			return fmt.Errorf("request %d decision changed: got=%v want=%v", i+1, gotDecisions, decisions)
		}
		samples = append(samples, total)
	}
	sort.Float64s(samples)
	result := sampleData{
		N:         len(samples),
		Min:       samples[0],
		Median:    median(samples),
		P95:       nearestRank(samples, .95),
		P99:       nearestRank(samples, .99),
		Max:       samples[len(samples)-1],
		Samples:   samples,
		Decisions: decisions,
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return writeAtomic(*outPath, encoded)
}

func request(client *http.Client, url string, body []byte) (float64, []map[string]any, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, 2<<20)
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(limited)
		return 0, nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	var decoded responseTiming
	if err := json.NewDecoder(limited).Decode(&decoded); err != nil {
		return 0, nil, err
	}
	if len(decoded.Results) == 0 || decoded.Timings.TotalMS <= 0 || math.IsNaN(decoded.Timings.TotalMS) || math.IsInf(decoded.Timings.TotalMS, 0) {
		return 0, nil, fmt.Errorf("invalid decision response")
	}
	decisions := make([]map[string]any, len(decoded.Results))
	for i := range decoded.Results {
		decisions[i] = decoded.Results[i].Decision
	}
	return decoded.Timings.TotalMS, decisions, nil
}

func validateBenchmarkData(data benchmarkData) error {
	if data.Schema != "go-system-one-benchmarks-v1" || len(data.Comparison) == 0 || len(data.Workloads) == 0 {
		return fmt.Errorf("invalid benchmark data contract")
	}
	validRange := func(minimum, representative, maximum float64) bool {
		return minimum > 0 && minimum <= representative && representative <= maximum && !math.IsNaN(minimum) && !math.IsNaN(representative) && !math.IsNaN(maximum) && !math.IsInf(minimum, 0) && !math.IsInf(representative, 0) && !math.IsInf(maximum, 0)
	}
	for i, item := range data.Comparison {
		if strings.TrimSpace(item.Label) == "" || strings.TrimSpace(item.Basis) == "" || !validRange(item.MinMS, item.RepresentativeMS, item.MaxMS) {
			return fmt.Errorf("invalid comparison row %d", i)
		}
	}
	for i, item := range data.Workloads {
		if strings.TrimSpace(item.Label) == "" || !validRange(item.MinMS, item.MedianMS, item.MaxMS) {
			return fmt.Errorf("invalid workload row %d", i)
		}
	}
	return nil
}

func validateSamples(data sampleData) error {
	if data.N != len(data.Samples) || data.N == 0 {
		return fmt.Errorf("sample count n=%d len=%d", data.N, len(data.Samples))
	}
	for i, value := range data.Samples {
		if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("invalid sample %d", i)
		}
		if i > 0 && value < data.Samples[i-1] {
			return fmt.Errorf("samples are not sorted")
		}
	}
	if !near(data.Min, data.Samples[0]) || !near(data.Max, data.Samples[len(data.Samples)-1]) || !near(data.Median, median(data.Samples)) || !near(data.P95, nearestRank(data.Samples, .95)) || !near(data.P99, nearestRank(data.Samples, .99)) {
		return errors.New("sample summary does not match samples")
	}
	return nil
}

func near(a, b float64) bool { return math.Abs(a-b) <= 1e-9 }

func median(values []float64) float64 {
	n := len(values)
	if n%2 == 1 {
		return values[n/2]
	}
	return (values[n/2-1] + values[n/2]) / 2
}

func nearestRank(values []float64, percentile float64) float64 {
	index := int(math.Ceil(percentile*float64(len(values)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func chartStart(title, subtitle string, width, height int) *strings.Builder {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-labelledby="title desc" viewBox="0 0 %d %d">`, width, height)
	fmt.Fprintf(&b, `<title id="title">%s</title><desc id="desc">%s</desc>`, html.EscapeString(title), html.EscapeString(subtitle))
	b.WriteString(`<style>
:root{--text:#1a2a40;--muted:#637b92;--grid:#c9d7e4;--surface:#e8eff6;--primary:#2b6cb0;--accent:#c05050;--good:#2a7a3a;--amber:#c87020}
@media(prefers-color-scheme:dark){:root{--text:#e8d8d0;--muted:#a88f80;--grid:#59463c;--surface:#2a1e18;--primary:#e08050;--accent:#50b0a0;--good:#60c870;--amber:#e0a840}}
text{fill:var(--text);font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}.muted{fill:var(--muted)}.grid{stroke:var(--grid);stroke-width:1}.axis{stroke:var(--text);stroke-width:1.2}.primary{fill:var(--primary)}.accent{fill:var(--accent)}.good{fill:var(--good)}.amber{fill:var(--amber)}.surface{fill:var(--surface)}
</style>`)
	fmt.Fprintf(&b, `<text x="40" y="42" font-size="24" font-weight="700">%s</text>`, html.EscapeString(title))
	fmt.Fprintf(&b, `<text class="muted" x="40" y="67" font-size="13">%s</text>`, html.EscapeString(subtitle))
	return &b
}

func renderDistribution(data sampleData) string {
	const width, height = 960, 480
	b := chartStart("Single-boolean warm latency", fmt.Sprintf("%d sequential requests after one warm-up · RTX 3060 · source %s", data.N, shortRevision(data.Revision)), width, height)
	left, top, plotW, plotH := 72.0, 105.0, 840.0, 285.0
	minV := math.Floor(data.Min)
	maxV := math.Ceil(data.Max)
	if maxV == minV {
		maxV++
	}
	for tick := minV; tick <= maxV; tick++ {
		y := top + plotH - (tick-minV)/(maxV-minV)*plotH
		fmt.Fprintf(b, `<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/><text class="muted" x="62" y="%.1f" text-anchor="end" font-size="11">%.0f</text>`, left, y, left+plotW, y, y+4, tick)
	}
	fmt.Fprintf(b, `<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/><line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`, left, top, left, top+plotH, left, top+plotH, left+plotW, top+plotH)
	points := make([]string, len(data.Samples))
	for i, value := range data.Samples {
		x := left + float64(i)/float64(max(1, len(data.Samples)-1))*plotW
		y := top + plotH - (value-minV)/(maxV-minV)*plotH
		points[i] = fmt.Sprintf("%.2f,%.2f", x, y)
	}
	fmt.Fprintf(b, `<polyline fill="none" stroke="var(--primary)" stroke-width="3" points="%s"/>`, strings.Join(points, " "))
	for _, marker := range []struct {
		label string
		value float64
		class string
	}{{"median", data.Median, "good"}, {"p95", data.P95, "amber"}, {"p99", data.P99, "accent"}} {
		y := top + plotH - (marker.value-minV)/(maxV-minV)*plotH
		fmt.Fprintf(b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--%s)" stroke-width="1.5" stroke-dasharray="6 5"/><text class="%s" x="%.1f" y="%.1f" font-size="12">%s %.2f ms</text>`, left, y, left+plotW, y, marker.class, marker.class, left+8, y-6, marker.label, marker.value)
	}
	fmt.Fprintf(b, `<text class="muted" x="%.1f" y="420" text-anchor="middle" font-size="12">request rank (sorted)</text>`, left+plotW/2)
	fmt.Fprintf(b, `<text class="muted" x="18" y="%.1f" text-anchor="middle" transform="rotate(-90 18 %.1f)" font-size="12">milliseconds</text>`, top+plotH/2, top+plotH/2)
	fmt.Fprintf(b, `<text class="muted" x="40" y="455" font-size="11">min %.2f · median %.2f · p95 %.2f · p99 %.2f · max %.2f ms</text>`, data.Min, data.Median, data.P95, data.P99, data.Max)
	b.WriteString(`</svg>`)
	return b.String()
}

func renderComparison(data benchmarkData) string {
	const width = 1080
	height := 180 + len(data.Comparison)*70
	b := chartStart("From prototype to hand-tuned Go", "Single-boolean fixture · historical stages plus current source "+shortRevision(data.Fixture.Revision), width, height)
	bottom := 105.0 + float64(len(data.Comparison))*70
	left, top, plotW := 280.0, 105.0, 650.0
	maxV := 0.0
	for _, item := range data.Comparison {
		maxV = math.Max(maxV, item.MaxMS)
	}
	maxV = math.Ceil(maxV/100) * 100
	for tick := 0.0; tick <= maxV; tick += 100 {
		x := left + tick/maxV*plotW
		fmt.Fprintf(b, `<line class="grid" x1="%.1f" y1="90" x2="%.1f" y2="%.1f"/><text class="muted" x="%.1f" y="%.1f" text-anchor="middle" font-size="11">%.0f</text>`, x, x, bottom, x, bottom+24, tick)
	}
	for i, item := range data.Comparison {
		y := top + float64(i)*70
		barW := item.RepresentativeMS / maxV * plotW
		class := "primary"
		if strings.HasPrefix(item.Label, "Current") {
			class = "good"
		} else if strings.Contains(item.Label, "llama") {
			class = "amber"
		}
		fmt.Fprintf(b, `<text x="268" y="%.1f" text-anchor="end" font-size="13">%s</text>`, y+20, html.EscapeString(item.Label))
		fmt.Fprintf(b, `<rect class="%s" x="%.1f" y="%.1f" width="%.1f" height="32" rx="5"/>`, class, left, y, barW)
		fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="13" font-weight="700">%.2f ms</text>`, left+barW+8, y+21, item.RepresentativeMS)
		if item.MaxMS > item.MinMS {
			x1, x2 := left+item.MinMS/maxV*plotW, left+item.MaxMS/maxV*plotW
			fmt.Fprintf(b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--text)" stroke-width="2"/><line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--text)"/><line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--text)"/>`, x1, y+38, x2, y+38, x1, y+34, x1, y+42, x2, y+34, x2, y+42)
		}
	}
	fmt.Fprintf(b, `<text class="muted" x="%.1f" y="%.1f" text-anchor="middle" font-size="12">milliseconds · llama.cpp: worker time; Go: handler time</text>`, left+plotW/2, bottom+50)
	b.WriteString(`</svg>`)
	return b.String()
}

func renderWorkloads(data benchmarkData) string {
	const width, height = 1060, 600
	b := chartStart("Current warm workloads", "Five warm samples per case · log scale · exact requests recorded · source "+shortRevision(data.Fixture.Revision), width, height)
	left, top, plotW := 255.0, 105.0, 720.0
	minV, maxV := 64.0, 2048.0
	logMin, logSpan := math.Log2(minV), math.Log2(maxV)-math.Log2(minV)
	toX := func(value float64) float64 { return left + (math.Log2(value)-logMin)/logSpan*plotW }
	for tick := minV; tick <= maxV; tick *= 2 {
		x := toX(tick)
		fmt.Fprintf(b, `<line class="grid" x1="%.1f" y1="90" x2="%.1f" y2="540"/><text class="muted" x="%.1f" y="565" text-anchor="middle" font-size="11">%.0f</text>`, x, x, x, tick)
	}
	for i, item := range data.Workloads {
		y := top + float64(i)*42
		x1, xm, x2 := toX(item.MinMS), toX(item.MedianMS), toX(item.MaxMS)
		fmt.Fprintf(b, `<text x="242" y="%.1f" text-anchor="end" font-size="12">%s</text>`, y+4, html.EscapeString(item.Label))
		fmt.Fprintf(b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--primary)" stroke-width="5" stroke-linecap="round"/>`, x1, y, x2, y)
		fmt.Fprintf(b, `<circle class="good" cx="%.1f" cy="%.1f" r="6"/><text class="muted" x="%.1f" y="%.1f" font-size="11">%.2f ms</text>`, xm, y, math.Min(x2+10, 986), y+4, item.MedianMS)
	}
	fmt.Fprintf(b, `<text class="muted" x="%.1f" y="590" text-anchor="middle" font-size="12">milliseconds (log₂ scale)</text>`, left+plotW/2)
	b.WriteString(`</svg>`)
	return b.String()
}
