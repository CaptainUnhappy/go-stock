package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type signalBacktestStock struct {
	TsCode   string
	Symbol   string
	Name     string
	Industry string
	Bars     kLineBars
	Amount   []float64
	Turnover []float64
	DateIdx  map[string]int
}

type signalBacktestFeature struct {
	Stock          *signalBacktestStock
	Idx            int
	Date           string
	Close          float64
	Total          int
	Bullish        int
	Bearish        int
	Oscillating    int
	Neutral        int
	BullishPct     float64
	BearishPct     float64
	OscillatingPct float64
	SignalSpread   float64
	TrendBull      int
	TrendBear      int
	MomentumBull   int
	MomentumBear   int
	VolumeBull     int
	VolumeBear     int
	Ret5           float64
	Ret20          float64
	DistMA20       float64
	DistMA60       float64
	VolRatio5      float64
	AmountMA20     float64
	Turnover       float64
	Score          float64
}

type signalBacktestMarket struct {
	AvgBullishPct float64
	AvgBearishPct float64
	AvgSpread     float64
	StrongPct     float64
	BearDominant  float64
}

type signalBacktestVariant struct {
	Name             string
	MinMarketStrong  float64
	MaxMarketBear    float64
	MinMarketSpread  float64
	MinSignalSpread  float64
	MinBullishPct    float64
	MaxBearishPct    float64
	MinOscillating   float64
	MinTrendBull     int
	MinMomentumBull  int
	MinAmountMA20    float64
	MinTurnover      float64
	MaxTurnover      float64
	MinDistMA20      float64
	MaxDistMA20      float64
	MinDistMA60      float64
	MinRet5          float64
	MaxRet5          float64
	MinRet20         float64
	MaxRet20         float64
	MinVolRatio      float64
	MaxVolRatio      float64
	ScoreMode        string
}

type signalBacktestTrade struct {
	SignalDate   string  `json:"signal_date"`
	EntryDate    string  `json:"entry_date"`
	ExitDate     string  `json:"exit_date"`
	PortfolioRet float64 `json:"portfolio_ret"`
	Names        string  `json:"names"`
	Codes        string  `json:"codes"`
	Count        int     `json:"count"`
	NAV          float64 `json:"nav"`
}

type signalBacktestStats struct {
	Variant              string  `json:"variant"`
	Periods              int     `json:"periods"`
	FinalNAV             float64 `json:"final_nav"`
	AnnualReturn         float64 `json:"annual_return"`
	MaxDrawdown          float64 `json:"max_drawdown"`
	WinRate              float64 `json:"win_rate"`
	AvgPeriodReturn      float64 `json:"avg_period_return"`
	BenchmarkFinalNAV    float64 `json:"benchmark_final_nav"`
	BenchmarkMaxDrawdown float64 `json:"benchmark_max_drawdown"`
	LatestCandidates     int     `json:"latest_candidates"`
}

func TestBacktestKlineSignalStrategies(t *testing.T) {
	if strings.TrimSpace(os.Getenv("GO_STOCK_BACKTEST_KLINE_SIGNALS")) != "true" {
		t.Skip("set GO_STOCK_BACKTEST_KLINE_SIGNALS=true to run exact kline signal backtest")
	}
	input := strings.TrimSpace(os.Getenv("GO_STOCK_BACKTEST_PANEL"))
	if input == "" {
		input = filepath.Join("..", "..", "outputs", "kline_signal_summary", "kline_panel_all.csv")
	}
	stocks, dates, err := loadSignalBacktestPanel(input)
	if err != nil {
		t.Fatal(err)
	}
	completeDate := latestCompleteSignalDate(dates)
	t.Logf("stocks=%d dates=%d complete=%s", len(stocks), len(dates), completeDate)

	variants := []signalBacktestVariant{
		{
			Name: "strong_pullback", MinMarketStrong: 0.012, MaxMarketBear: 0.50, MinMarketSpread: -0.28,
			MinBullishPct: 0.50, MaxBearishPct: 0.22, MinSignalSpread: 0.26, MinTrendBull: 8, MinMomentumBull: 4,
			MinAmountMA20: 80_000_000, MinTurnover: 0.006, MaxTurnover: 0.18,
			MinDistMA20: -0.035, MaxDistMA20: 0.055, MinDistMA60: -0.05,
			MinRet5: -0.045, MaxRet5: 0.075, MinRet20: -0.10, MaxRet20: 0.24,
			MinVolRatio: 0.80, MaxVolRatio: 2.30, ScoreMode: "pullback",
		},
		{
			Name: "low_turn_confirm", MinMarketStrong: 0.006, MaxMarketBear: 0.54, MinMarketSpread: -0.35,
			MinBullishPct: 0.34, MaxBearishPct: 0.36, MinSignalSpread: -0.02, MinOscillating: 0.20, MinTrendBull: 4, MinMomentumBull: 3,
			MinAmountMA20: 60_000_000, MinTurnover: 0.006, MaxTurnover: 0.20,
			MinDistMA20: -0.105, MaxDistMA20: 0.025, MinDistMA60: -0.18,
			MinRet5: -0.12, MaxRet5: 0.035, MinRet20: -0.22, MaxRet20: 0.12,
			MinVolRatio: 0.70, MaxVolRatio: 1.90, ScoreMode: "turn",
		},
		{
			Name: "trend_continuation", MinMarketStrong: 0.018, MaxMarketBear: 0.48, MinMarketSpread: -0.24,
			MinBullishPct: 0.56, MaxBearishPct: 0.18, MinSignalSpread: 0.36, MinTrendBull: 10, MinMomentumBull: 5,
			MinAmountMA20: 100_000_000, MinTurnover: 0.006, MaxTurnover: 0.22,
			MinDistMA20: -0.005, MaxDistMA20: 0.095, MinDistMA60: 0.00,
			MinRet5: -0.025, MaxRet5: 0.10, MinRet20: 0.00, MaxRet20: 0.35,
			MinVolRatio: 0.85, MaxVolRatio: 2.80, ScoreMode: "trend",
		},
	}

	outDir := filepath.Join("..", "..", "outputs", "kline_signal_summary", "exact_backtest")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	stats := make([]signalBacktestStats, 0, len(variants))
	latestRows := map[string][]signalBacktestFeature{}
	for _, variant := range variants {
		trades, latestCandidates := runSignalVariantBacktest(stocks, dates, completeDate, variant, 5, 5, 5)
		equity := tradesToEquity(trades)
		bench := benchmarkEquity(stocks, dates, completeDate, trades)
		stat := calcSignalBacktestStats(variant.Name, trades, equity, bench, 5, len(latestCandidates))
		stats = append(stats, stat)
		latestRows[variant.Name] = latestCandidates
		writeSignalBacktestTrades(t, filepath.Join(outDir, variant.Name+"_trades.csv"), trades)
		writeSignalBacktestCandidates(t, filepath.Join(outDir, variant.Name+"_latest_candidates.csv"), latestCandidates)
		t.Logf("%s final=%.4f ann=%.2f%% mdd=%.2f%% win=%.2f%% latest=%d", variant.Name, stat.FinalNAV, stat.AnnualReturn*100, stat.MaxDrawdown*100, stat.WinRate*100, stat.LatestCandidates)
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].FinalNAV > stats[j].FinalNAV })
	raw, _ := json.MarshalIndent(stats, "", "  ")
	if err := os.WriteFile(filepath.Join(outDir, "strategy_stats.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func loadSignalBacktestPanel(path string) ([]*signalBacktestStock, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(records) < 2 {
		return nil, nil, fmt.Errorf("empty panel: %s", path)
	}
	header := map[string]int{}
	for i, name := range records[0] {
		header[name] = i
	}
	byCode := map[string]*signalBacktestStock{}
	dateSet := map[string]bool{}
	get := func(row []string, name string) string {
		idx, ok := header[name]
		if !ok || idx >= len(row) {
			return ""
		}
		return row[idx]
	}
	for _, row := range records[1:] {
		code := get(row, "ts_code")
		if code == "" {
			continue
		}
		openV, okOpen := parsePanelFloat(get(row, "open"))
		closeV, okClose := parsePanelFloat(get(row, "close"))
		highV, okHigh := parsePanelFloat(get(row, "high"))
		lowV, okLow := parsePanelFloat(get(row, "low"))
		if !okOpen || !okClose || !okHigh || !okLow {
			continue
		}
		stock := byCode[code]
		if stock == nil {
			stock = &signalBacktestStock{TsCode: code, Symbol: get(row, "symbol"), Name: get(row, "name"), Industry: get(row, "industry"), DateIdx: map[string]int{}}
			byCode[code] = stock
		}
		day := get(row, "date")
		vol, _ := parsePanelFloat(get(row, "volume"))
		amount, _ := parsePanelFloat(get(row, "amount"))
		turnover, _ := parsePanelFloat(get(row, "turnover"))
		stock.DateIdx[day] = len(stock.Bars.days)
		stock.Bars.days = append(stock.Bars.days, day)
		stock.Bars.open = append(stock.Bars.open, openV)
		stock.Bars.close = append(stock.Bars.close, closeV)
		stock.Bars.high = append(stock.Bars.high, highV)
		stock.Bars.low = append(stock.Bars.low, lowV)
		stock.Bars.volume = append(stock.Bars.volume, vol)
		stock.Amount = append(stock.Amount, amount)
		stock.Turnover = append(stock.Turnover, turnover/100.0)
		dateSet[day] = true
	}
	stocks := make([]*signalBacktestStock, 0, len(byCode))
	for _, stock := range byCode {
		if len(stock.Bars.close) >= 260 {
			stocks = append(stocks, stock)
		}
	}
	sort.Slice(stocks, func(i, j int) bool { return stocks[i].TsCode < stocks[j].TsCode })
	dates := make([]string, 0, len(dateSet))
	for day := range dateSet {
		dates = append(dates, day)
	}
	sort.Strings(dates)
	return stocks, dates, nil
}

func parsePanelFloat(value string) (float64, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(value, 64)
	return v, err == nil && !math.IsNaN(v) && !math.IsInf(v, 0)
}

func latestCompleteSignalDate(dates []string) string {
	if len(dates) == 0 {
		return ""
	}
	today := time.Now().Format("2006-01-02")
	last := dates[len(dates)-1]
	if last >= today && len(dates) >= 2 {
		return dates[len(dates)-2]
	}
	return last
}

func runSignalVariantBacktest(stocks []*signalBacktestStock, dates []string, completeDate string, variant signalBacktestVariant, topN, holdDays, stepDays int) ([]signalBacktestTrade, []signalBacktestFeature) {
	var trades []signalBacktestTrade
	nav := 1.0
	start := 260
	latestCandidates := []signalBacktestFeature{}
	for di := start; di < len(dates); di += stepDays {
		day := dates[di]
		if day > completeDate {
			break
		}
		features, market := buildSignalFeaturesForDate(stocks, day, 250)
		candidates := make([]signalBacktestFeature, 0)
		for _, feature := range features {
			if signalVariantPass(feature, market, variant) {
				feature.Score = scoreSignalFeature(feature, variant)
				candidates = append(candidates, feature)
			}
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
		if day == completeDate {
			latestCandidates = append(latestCandidates, candidates...)
		}
		pick := candidates
		if len(pick) > topN {
			pick = pick[:topN]
		}
		trade := signalBacktestTrade{SignalDate: day, Names: "cash"}
		if len(pick) > 0 {
			rets := make([]float64, 0, len(pick))
			names := make([]string, 0, len(pick))
			codes := make([]string, 0, len(pick))
			entryDate, exitDate := "", ""
			for _, feature := range pick {
				ret, ent, exit, ok := simulateSignalTrade(feature.Stock, feature.Idx, holdDays)
				if !ok {
					continue
				}
				rets = append(rets, ret)
				names = append(names, feature.Stock.Name)
				codes = append(codes, feature.Stock.TsCode)
				if entryDate == "" {
					entryDate, exitDate = ent, exit
				}
			}
			if len(rets) > 0 {
				trade.PortfolioRet = avgFloat(rets)
				trade.EntryDate = entryDate
				trade.ExitDate = exitDate
				trade.Names = strings.Join(names, " ")
				trade.Codes = strings.Join(codes, " ")
				trade.Count = len(rets)
			}
		}
		nav *= 1 + trade.PortfolioRet
		trade.NAV = nav
		trades = append(trades, trade)
	}
	return trades, latestCandidates
}

func buildSignalFeaturesForDate(stocks []*signalBacktestStock, day string, limit int) ([]signalBacktestFeature, signalBacktestMarket) {
	features := make([]signalBacktestFeature, 0, len(stocks))
	for _, stock := range stocks {
		idx, ok := stock.DateIdx[day]
		if !ok || idx < 180 || idx+6 >= len(stock.Bars.close) {
			continue
		}
		start := idx - limit + 1
		if start < 0 {
			start = 0
		}
		bars := kLineBars{
			days:   stock.Bars.days[start : idx+1],
			open:   stock.Bars.open[start : idx+1],
			close:  stock.Bars.close[start : idx+1],
			high:   stock.Bars.high[start : idx+1],
			low:    stock.Bars.low[start : idx+1],
			volume: stock.Bars.volume[start : idx+1],
		}
		signals := evaluateKLineSignals(bars)
		if len(signals) == 0 {
			continue
		}
		feature := summarizeSignalFeature(stock, idx, signals)
		if feature.Total > 0 {
			features = append(features, feature)
		}
	}
	market := signalBacktestMarket{}
	if len(features) == 0 {
		return features, market
	}
	bearDominant := 0
	for _, feature := range features {
		market.AvgBullishPct += feature.BullishPct
		market.AvgBearishPct += feature.BearishPct
		market.AvgSpread += feature.SignalSpread
		if feature.BullishPct >= 0.55 && feature.BearishPct <= 0.15 {
			market.StrongPct++
		}
		if feature.Bearish > feature.Bullish && feature.Bearish > feature.Oscillating {
			bearDominant++
		}
	}
	denom := float64(len(features))
	market.AvgBullishPct /= denom
	market.AvgBearishPct /= denom
	market.AvgSpread /= denom
	market.StrongPct /= denom
	market.BearDominant = float64(bearDominant) / denom
	return features, market
}

func summarizeSignalFeature(stock *signalBacktestStock, idx int, signals []kLineSignal) signalBacktestFeature {
	f := signalBacktestFeature{Stock: stock, Idx: idx, Date: stock.Bars.days[idx], Close: stock.Bars.close[idx], Total: len(signals)}
	for _, signal := range signals {
		switch signal.Signal {
		case "bullish":
			f.Bullish++
			switch signal.Group {
			case "趋势":
				f.TrendBull++
			case "动量":
				f.MomentumBull++
			case "量价":
				f.VolumeBull++
			}
		case "bearish":
			f.Bearish++
			switch signal.Group {
			case "趋势":
				f.TrendBear++
			case "动量":
				f.MomentumBear++
			case "量价":
				f.VolumeBear++
			}
		case "oscillating":
			f.Oscillating++
		case "neutral":
			f.Neutral++
		}
	}
	f.BullishPct = ratio(f.Bullish, f.Total)
	f.BearishPct = ratio(f.Bearish, f.Total)
	f.OscillatingPct = ratio(f.Oscillating, f.Total)
	f.SignalSpread = f.BullishPct - f.BearishPct
	closeV := stock.Bars.close
	if idx >= 5 && closeV[idx-5] > 0 {
		f.Ret5 = closeV[idx]/closeV[idx-5] - 1
	}
	if idx >= 20 && closeV[idx-20] > 0 {
		f.Ret20 = closeV[idx]/closeV[idx-20] - 1
	}
	if ma20, ok := trailingAvg(closeV, idx, 20); ok && ma20 > 0 {
		f.DistMA20 = closeV[idx]/ma20 - 1
	}
	if ma60, ok := trailingAvg(closeV, idx, 60); ok && ma60 > 0 {
		f.DistMA60 = closeV[idx]/ma60 - 1
	}
	if avgVol, ok := trailingAvg(stock.Bars.volume, idx-1, 5); ok && avgVol > 0 {
		f.VolRatio5 = stock.Bars.volume[idx] / avgVol
	}
	f.AmountMA20, _ = trailingAvg(stock.Amount, idx, 20)
	f.Turnover = stock.Turnover[idx]
	return f
}

func trailingAvg(values []float64, idx int, period int) (float64, bool) {
	if idx < period-1 || idx >= len(values) {
		return 0, false
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		v := values[idx-i]
		if !finite(v) {
			return 0, false
		}
		sum += v
	}
	return sum / float64(period), true
}

func signalVariantPass(f signalBacktestFeature, m signalBacktestMarket, v signalBacktestVariant) bool {
	if m.StrongPct < v.MinMarketStrong || m.AvgBearishPct > v.MaxMarketBear || m.AvgSpread < v.MinMarketSpread {
		return false
	}
	return f.Total >= 40 &&
		f.BullishPct >= v.MinBullishPct &&
		f.BearishPct <= v.MaxBearishPct &&
		f.SignalSpread >= v.MinSignalSpread &&
		f.OscillatingPct >= v.MinOscillating &&
		f.TrendBull >= v.MinTrendBull &&
		f.MomentumBull >= v.MinMomentumBull &&
		f.AmountMA20 >= v.MinAmountMA20 &&
		f.Turnover >= v.MinTurnover && f.Turnover <= v.MaxTurnover &&
		f.DistMA20 >= v.MinDistMA20 && f.DistMA20 <= v.MaxDistMA20 &&
		f.DistMA60 >= v.MinDistMA60 &&
		f.Ret5 >= v.MinRet5 && f.Ret5 <= v.MaxRet5 &&
		f.Ret20 >= v.MinRet20 && f.Ret20 <= v.MaxRet20 &&
		f.VolRatio5 >= v.MinVolRatio && f.VolRatio5 <= v.MaxVolRatio
}

func scoreSignalFeature(f signalBacktestFeature, v signalBacktestVariant) float64 {
	trendSpread := float64(f.TrendBull-f.TrendBear) / 18.0
	momSpread := float64(f.MomentumBull-f.MomentumBear) / 14.0
	volBalance := 1 - math.Min(math.Abs(f.VolRatio5-1.15)/1.5, 1)
	switch v.ScoreMode {
	case "turn":
		retRepair := 1 - math.Min(math.Abs(f.Ret5+0.045)/0.12, 1)
		maRepair := 1 - math.Min(math.Abs(f.DistMA20+0.035)/0.10, 1)
		return 0.25*retRepair + 0.24*maRepair + 0.18*f.OscillatingPct + 0.16*f.BullishPct + 0.10*volBalance + 0.07*f.SignalSpread
	case "trend":
		maSweet := 1 - math.Min(math.Abs(f.DistMA20-0.035)/0.10, 1)
		retSweet := 1 - math.Min(math.Abs(f.Ret5-0.035)/0.10, 1)
		return 0.34*f.SignalSpread + 0.20*trendSpread + 0.16*momSpread + 0.14*maSweet + 0.10*retSweet + 0.06*volBalance
	default:
		maSweet := 1 - math.Min(math.Abs(f.DistMA20-0.005)/0.07, 1)
		retSweet := 1 - math.Min(math.Abs(f.Ret5-0.015)/0.08, 1)
		return 0.30*f.SignalSpread + 0.20*trendSpread + 0.16*momSpread + 0.14*maSweet + 0.10*retSweet + 0.10*volBalance
	}
}

func simulateSignalTrade(stock *signalBacktestStock, signalIdx int, holdDays int) (float64, string, string, bool) {
	entryIdx := signalIdx + 1
	if entryIdx >= len(stock.Bars.open) {
		return 0, "", "", false
	}
	entryOpen := stock.Bars.open[entryIdx]
	if entryOpen <= 0 {
		return 0, "", "", false
	}
	stopPrice := entryOpen * (1 - 0.055)
	takePrice := entryOpen * (1 + 0.095)
	exitIdx := entryIdx + holdDays
	if exitIdx >= len(stock.Bars.close) {
		return 0, "", "", false
	}
	exitPrice := stock.Bars.close[exitIdx]
	for i := entryIdx + 1; i <= exitIdx; i++ {
		if stock.Bars.low[i] <= stopPrice {
			exitIdx = i
			exitPrice = stopPrice
			break
		}
		if stock.Bars.high[i] >= takePrice {
			exitIdx = i
			exitPrice = takePrice
			break
		}
	}
	return netSignalReturn(entryOpen, exitPrice), stock.Bars.days[entryIdx], stock.Bars.days[exitIdx], true
}

func netSignalReturn(entryOpen, exitPrice float64) float64 {
	commission := 2.5 / 10000.0
	slippage := 5.0 / 10000.0
	stampTax := 5.0 / 10000.0
	transferFee := 0.2 / 10000.0
	buyPrice := entryOpen * (1 + slippage)
	sellPrice := exitPrice * (1 - slippage)
	return (sellPrice*(1-commission-stampTax-transferFee))/(buyPrice*(1+commission+transferFee)) - 1
}

func avgFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func tradesToEquity(trades []signalBacktestTrade) []float64 {
	equity := make([]float64, len(trades))
	for i, trade := range trades {
		equity[i] = trade.NAV
	}
	return equity
}

func benchmarkEquity(stocks []*signalBacktestStock, dates []string, completeDate string, trades []signalBacktestTrade) []float64 {
	if len(trades) == 0 {
		return nil
	}
	tradeDates := map[string]bool{}
	for _, trade := range trades {
		tradeDates[trade.SignalDate] = true
	}
	nav := 1.0
	out := make([]float64, 0, len(trades))
	for _, day := range dates {
		if day > completeDate {
			break
		}
		if !tradeDates[day] {
			continue
		}
		var rets []float64
		for _, stock := range stocks {
			idx, ok := stock.DateIdx[day]
			if !ok || idx == 0 {
				continue
			}
			prev := stock.Bars.close[idx-1]
			if prev > 0 {
				rets = append(rets, stock.Bars.close[idx]/prev-1)
			}
		}
		nav *= 1 + avgFloat(rets)
		out = append(out, nav)
	}
	return out
}

func calcSignalBacktestStats(name string, trades []signalBacktestTrade, equity []float64, bench []float64, stepDays int, latest int) signalBacktestStats {
	stat := signalBacktestStats{Variant: name, Periods: len(trades), LatestCandidates: latest}
	if len(trades) == 0 || len(equity) == 0 {
		return stat
	}
	stat.FinalNAV = equity[len(equity)-1]
	years := math.Max(float64(len(trades)*stepDays)/252.0, 1.0/252.0)
	stat.AnnualReturn = math.Pow(stat.FinalNAV, 1/years) - 1
	stat.MaxDrawdown = maxDrawdownFloat(equity)
	wins, sum := 0, 0.0
	for _, trade := range trades {
		if trade.PortfolioRet > 0 {
			wins++
		}
		sum += trade.PortfolioRet
	}
	stat.WinRate = float64(wins) / float64(len(trades))
	stat.AvgPeriodReturn = sum / float64(len(trades))
	if len(bench) > 0 {
		stat.BenchmarkFinalNAV = bench[len(bench)-1]
		stat.BenchmarkMaxDrawdown = maxDrawdownFloat(bench)
	}
	return stat
}

func maxDrawdownFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	peak := values[0]
	mdd := 0.0
	for _, v := range values {
		if v > peak {
			peak = v
		}
		if peak > 0 {
			dd := v/peak - 1
			if dd < mdd {
				mdd = dd
			}
		}
	}
	return mdd
}

func writeSignalBacktestTrades(t *testing.T, path string, trades []signalBacktestTrade) {
	t.Helper()
	rows := [][]string{{"signal_date", "entry_date", "exit_date", "portfolio_ret", "names", "codes", "count", "nav"}}
	for _, trade := range trades {
		rows = append(rows, []string{trade.SignalDate, trade.EntryDate, trade.ExitDate, fmt.Sprintf("%.8f", trade.PortfolioRet), trade.Names, trade.Codes, strconv.Itoa(trade.Count), fmt.Sprintf("%.8f", trade.NAV)})
	}
	writeCSV(t, path, rows)
}

func writeSignalBacktestCandidates(t *testing.T, path string, candidates []signalBacktestFeature) {
	t.Helper()
	if len(candidates) > 50 {
		candidates = candidates[:50]
	}
	rows := [][]string{{"date", "ts_code", "name", "close", "score", "bullish", "bearish", "oscillating", "neutral", "bullish_pct", "bearish_pct", "signal_spread", "trend_bull", "momentum_bull", "ret5", "ret20", "dist_ma20", "vol_ratio5", "turnover", "amount_ma20"}}
	for _, f := range candidates {
		rows = append(rows, []string{
			f.Date, f.Stock.TsCode, f.Stock.Name, fmt.Sprintf("%.2f", f.Close), fmt.Sprintf("%.6f", f.Score),
			strconv.Itoa(f.Bullish), strconv.Itoa(f.Bearish), strconv.Itoa(f.Oscillating), strconv.Itoa(f.Neutral),
			fmt.Sprintf("%.4f", f.BullishPct), fmt.Sprintf("%.4f", f.BearishPct), fmt.Sprintf("%.4f", f.SignalSpread),
			strconv.Itoa(f.TrendBull), strconv.Itoa(f.MomentumBull), fmt.Sprintf("%.4f", f.Ret5), fmt.Sprintf("%.4f", f.Ret20),
			fmt.Sprintf("%.4f", f.DistMA20), fmt.Sprintf("%.4f", f.VolRatio5), fmt.Sprintf("%.4f", f.Turnover), fmt.Sprintf("%.0f", f.AmountMA20),
		})
	}
	writeCSV(t, path, rows)
}
