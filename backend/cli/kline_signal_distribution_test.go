package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"go-stock/backend/appdata"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go.uber.org/zap"
)

type signalDistributionStockBasic struct {
	Data struct {
		Fields []string        `json:"fields"`
		Items  [][]interface{} `json:"items"`
	} `json:"data"`
}

type signalDistributionStock struct {
	TsCode string
	Symbol string
	Name   string
}

type signalDistributionCountBucket map[string]int

func TestExportKlineSignalDistribution(t *testing.T) {
	if strings.TrimSpace(os.Getenv("GO_STOCK_EXPORT_KLINE_SIGNAL_DISTRIBUTION")) != "true" {
		t.Skip("set GO_STOCK_EXPORT_KLINE_SIGNAL_DISTRIBUTION=true to export kline signal distribution")
	}
	logger.Logger = zap.NewNop()
	logger.SugaredLogger = logger.Logger.Sugar()
	dbPath := strings.TrimSpace(os.Getenv("GO_STOCK_DB"))
	if dbPath == "" {
		var err error
		dbPath, err = appdata.DefaultDBPath()
		if err != nil {
			t.Fatal(err)
		}
	}
	db.Init(dbPath)
	data.InitAnalyzeSentiment()

	limit := envInt("GO_STOCK_SIGNAL_LIMIT", 250)
	workers := envInt("GO_STOCK_SIGNAL_WORKERS", 12)
	maxStocks := envInt("GO_STOCK_SIGNAL_MAX_STOCKS", 0)
	outDir := filepath.Join("..", "..", "outputs", "kline_signal_summary")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	universe, err := loadSignalDistributionUniverse(filepath.Join("..", "..", "build", "stock_basic.json"), maxStocks)
	if err != nil {
		t.Fatal(err)
	}

	type stockResult struct {
		Stock   signalDistributionStock
		Summary kLineSignalSummary
		OK      bool
	}
	jobs := make(chan signalDistributionStock)
	results := make(chan stockResult, workers*2)
	var done int32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for stock := range jobs {
				summary, ok := buildKLineSignalSummary(stock.TsCode, "day", limit, "qfq")
				results <- stockResult{Stock: stock, Summary: summary, OK: ok}
				cur := atomic.AddInt32(&done, 1)
				if cur%100 == 0 || int(cur) == len(universe) {
					t.Logf("signals %d/%d", cur, len(universe))
				}
			}
		}()
	}
	go func() {
		for _, stock := range universe {
			jobs <- stock
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	signalLabels := []string{"bullish", "bearish", "oscillating", "neutral"}
	labelText := map[string]string{"bullish": "看多", "bearish": "看空", "oscillating": "震荡", "neutral": "中性"}
	indicatorCounts := map[string]signalDistributionCountBucket{}
	groupCounts := map[string]signalDistributionCountBucket{}
	totalCounts := signalDistributionCountBucket{}
	stockRows := [][]string{{"ts_code", "symbol", "name", "date", "close", "total", "bullish", "bearish", "oscillating", "neutral", "bullish_pct", "bearish_pct", "oscillating_pct", "neutral_pct", "dominant", "source"}}
	okCount := 0
	for result := range results {
		if !result.OK {
			continue
		}
		okCount++
		summary := result.Summary
		dominant := dominantSignal(summary)
		stockRows = append(stockRows, []string{
			result.Stock.TsCode,
			result.Stock.Symbol,
			result.Stock.Name,
			summary.LatestDay,
			fmt.Sprintf("%.2f", summary.LatestClose),
			strconv.Itoa(summary.Total),
			strconv.Itoa(summary.Bullish),
			strconv.Itoa(summary.Bearish),
			strconv.Itoa(summary.Oscillating),
			strconv.Itoa(summary.Neutral),
			fmt.Sprintf("%.4f", ratio(summary.Bullish, summary.Total)),
			fmt.Sprintf("%.4f", ratio(summary.Bearish, summary.Total)),
			fmt.Sprintf("%.4f", ratio(summary.Oscillating, summary.Total)),
			fmt.Sprintf("%.4f", ratio(summary.Neutral, summary.Total)),
			labelText[dominant],
			summary.Source,
		})
		totalCounts["bullish"] += summary.Bullish
		totalCounts["bearish"] += summary.Bearish
		totalCounts["oscillating"] += summary.Oscillating
		totalCounts["neutral"] += summary.Neutral
		for _, signal := range summary.Signals {
			if indicatorCounts[signal.Name] == nil {
				indicatorCounts[signal.Name] = signalDistributionCountBucket{}
			}
			if groupCounts[signal.Group] == nil {
				groupCounts[signal.Group] = signalDistributionCountBucket{}
			}
			indicatorCounts[signal.Name][signal.Signal]++
			groupCounts[signal.Group][signal.Signal]++
		}
	}

	writeCSV(t, filepath.Join(outDir, "signal_distribution_by_stock.csv"), stockRows)
	writeDistributionCSV(t, filepath.Join(outDir, "signal_distribution_by_indicator.csv"), indicatorCounts, signalLabels)
	writeDistributionCSV(t, filepath.Join(outDir, "signal_distribution_by_group.csv"), groupCounts, signalLabels)
	summary := map[string]any{
		"universe":     len(universe),
		"computed":     okCount,
		"limit":        limit,
		"total_counts": totalCounts,
		"total_ratio": map[string]float64{
			"bullish":     ratio(totalCounts["bullish"], totalSignals(totalCounts)),
			"bearish":     ratio(totalCounts["bearish"], totalSignals(totalCounts)),
			"oscillating": ratio(totalCounts["oscillating"], totalSignals(totalCounts)),
			"neutral":     ratio(totalCounts["neutral"], totalSignals(totalCounts)),
		},
	}
	raw, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(filepath.Join(outDir, "signal_distribution_summary.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func ratio(count, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(count) / float64(total)
}

func totalSignals(counts map[string]int) int {
	return counts["bullish"] + counts["bearish"] + counts["oscillating"] + counts["neutral"]
}

func dominantSignal(summary kLineSignalSummary) string {
	values := map[string]int{
		"bullish":     summary.Bullish,
		"bearish":     summary.Bearish,
		"oscillating": summary.Oscillating,
		"neutral":     summary.Neutral,
	}
	order := []string{"bullish", "bearish", "oscillating", "neutral"}
	best := order[0]
	for _, key := range order[1:] {
		if values[key] > values[best] {
			best = key
		}
	}
	return best
}

func writeCSV(t *testing.T, path string, rows [][]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			t.Fatal(err)
		}
	}
}

func writeDistributionCSV(t *testing.T, path string, counts map[string]signalDistributionCountBucket, signalLabels []string) {
	t.Helper()
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := [][]string{{"name", "total", "bullish", "bearish", "oscillating", "neutral", "bullish_pct", "bearish_pct", "oscillating_pct", "neutral_pct"}}
	for _, key := range keys {
		total := 0
		for _, signal := range signalLabels {
			total += counts[key][signal]
		}
		row := []string{key, strconv.Itoa(total)}
		for _, signal := range signalLabels {
			row = append(row, strconv.Itoa(counts[key][signal]))
		}
		for _, signal := range signalLabels {
			row = append(row, fmt.Sprintf("%.4f", ratio(counts[key][signal], total)))
		}
		rows = append(rows, row)
	}
	writeCSV(t, path, rows)
}

func loadSignalDistributionUniverse(path string, maxStocks int) ([]signalDistributionStock, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload signalDistributionStockBasic
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	fieldIndex := map[string]int{}
	for i, name := range payload.Data.Fields {
		fieldIndex[name] = i
	}
	prefixes := []string{"000", "001", "002", "003", "300", "301", "600", "601", "603", "605", "688"}
	var out []signalDistributionStock
	for _, row := range payload.Data.Items {
		get := func(name string) string {
			idx, ok := fieldIndex[name]
			if !ok || idx >= len(row) || row[idx] == nil {
				return ""
			}
			return fmt.Sprint(row[idx])
		}
		symbol := get("symbol")
		name := get("name")
		if get("list_status") != "L" || strings.Contains(strings.ToUpper(name), "ST") || strings.Contains(name, "退") {
			continue
		}
		okPrefix := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(symbol, prefix) {
				okPrefix = true
				break
			}
		}
		if okPrefix {
			out = append(out, signalDistributionStock{TsCode: get("ts_code"), Symbol: symbol, Name: name})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TsCode < out[j].TsCode })
	if maxStocks > 0 && maxStocks < len(out) {
		step := float64(len(out)) / float64(maxStocks)
		sampled := make([]signalDistributionStock, 0, maxStocks)
		for i := 0; i < maxStocks; i++ {
			idx := int(float64(i) * step)
			if idx >= len(out) {
				idx = len(out) - 1
			}
			sampled = append(sampled, out[idx])
		}
		return sampled, nil
	}
	return out, nil
}
