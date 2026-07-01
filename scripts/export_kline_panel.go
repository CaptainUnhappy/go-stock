package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"go-stock/backend/appdata"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go.uber.org/zap"
)

type stockBasicFile struct {
	Data struct {
		Fields []string        `json:"fields"`
		Items  [][]interface{} `json:"items"`
	} `json:"data"`
}

type stockItem struct {
	TsCode   string
	Symbol   string
	Name     string
	Industry string
}

func main() {
	maxStocks := flag.Int("max-stocks", 0, "0 means all eligible A shares")
	limit := flag.Int("limit", 520, "K-line bars per stock")
	workers := flag.Int("workers", 8, "concurrent workers")
	outPath := flag.String("out", "outputs/kline_signal_summary/kline_panel.csv", "output CSV path")
	flag.Parse()

	quietLogger()
	dbPath := strings.TrimSpace(os.Getenv("GO_STOCK_DB"))
	if dbPath == "" {
		var err error
		dbPath, err = appdata.DefaultDBPath()
		if err != nil {
			panic(err)
		}
	}
	db.Init(dbPath)
	data.InitAnalyzeSentiment()

	universe, err := loadUniverse("build/stock_basic.json", *maxStocks)
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0755); err != nil {
		panic(err)
	}
	f, err := os.Create(*outPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()
	header := []string{"date", "open", "close", "high", "low", "volume", "amount", "pct_chg", "change", "amplitude", "turnover", "ts_code", "symbol", "name", "industry", "source"}
	if err := writer.Write(header); err != nil {
		panic(err)
	}

	jobs := make(chan stockItem)
	rows := make(chan []string, 4096)
	var done int32
	var usable int32
	var mu sync.Mutex

	var writerWG sync.WaitGroup
	writerWG.Add(1)
	go func() {
		defer writerWG.Done()
		for row := range rows {
			mu.Lock()
			_ = writer.Write(row)
			mu.Unlock()
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for stock := range jobs {
				rs := data.FetchKLineWithFallback(stock.TsCode, stock.Name, "101", *limit, "")
				n := 0
				if rs != nil && rs.Data != nil {
					for _, k := range *rs.Data {
						rows <- []string{
							k.Day,
							k.Open,
							k.Close,
							k.High,
							k.Low,
							k.Volume,
							k.Amount,
							k.ChangePercent,
							k.ChangeValue,
							k.Amplitude,
							k.TurnoverRate,
							stock.TsCode,
							stock.Symbol,
							stock.Name,
							stock.Industry,
							rs.Source,
						}
						n++
					}
				}
				if n >= 180 {
					atomic.AddInt32(&usable, 1)
				}
				cur := atomic.AddInt32(&done, 1)
				if cur%50 == 0 || int(cur) == len(universe) {
					fmt.Printf("exported %d/%d, usable=%d\n", cur, len(universe), atomic.LoadInt32(&usable))
				}
			}
		}()
	}

	for _, stock := range universe {
		jobs <- stock
	}
	close(jobs)
	wg.Wait()
	close(rows)
	writerWG.Wait()
	fmt.Printf("wrote %s\n", *outPath)
}

func quietLogger() {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("GO_STOCK_CLI_VERBOSE")), "true") {
		return
	}
	logger.Logger = zap.NewNop()
	logger.SugaredLogger = logger.Logger.Sugar()
}

func loadUniverse(path string, maxStocks int) ([]stockItem, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload stockBasicFile
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	fieldIndex := map[string]int{}
	for i, name := range payload.Data.Fields {
		fieldIndex[name] = i
	}
	prefixes := []string{"000", "001", "002", "003", "300", "301", "600", "601", "603", "605", "688"}
	var out []stockItem
	for _, row := range payload.Data.Items {
		get := func(name string) string {
			idx, ok := fieldIndex[name]
			if !ok || idx >= len(row) || row[idx] == nil {
				return ""
			}
			return fmt.Sprint(row[idx])
		}
		tsCode := get("ts_code")
		symbol := get("symbol")
		name := get("name")
		if get("list_status") != "L" {
			continue
		}
		if strings.Contains(strings.ToUpper(name), "ST") || strings.Contains(name, "退") {
			continue
		}
		okPrefix := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(symbol, prefix) {
				okPrefix = true
				break
			}
		}
		if !okPrefix {
			continue
		}
		out = append(out, stockItem{TsCode: tsCode, Symbol: symbol, Name: name, Industry: get("industry")})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TsCode < out[j].TsCode })
	if maxStocks > 0 && maxStocks < len(out) {
		step := float64(len(out)) / float64(maxStocks)
		sampled := make([]stockItem, 0, maxStocks)
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
