package cli

import (
	"context"
	"fmt"
	"math"
	"strings"

	"go-stock/backend/data"
	"go-stock/backend/util"
)

type kLineSignal struct {
	Name   string
	Signal string
	Group  string
}

type kLineSignalRow struct {
	Indicator string `md:"指标"`
	Signal    string `md:"信号"`
	Group     string `md:"分组"`
}

type kLineSignalSummary struct {
	Total       int
	Bullish     int
	Bearish     int
	Oscillating int
	Neutral     int
	Signals     []kLineSignal
	Source      string
	LatestDay   string
	LatestClose float64
	AdjustFlag  string
}

func runKlineSignals(_ *toolRunner) Handler {
	return func(_ context.Context, args map[string]any) (string, error) {
		normalizeCommonArgs(args)
		code, err := requiredString(args, "stockCode")
		if err != nil {
			return "", err
		}
		kLineType := optionalString(args, "kLineType", optionalString(args, "period", "day"))
		limit := optionalInt(args, "limit", 250)
		adjustFlag, err := kLineAdjustFlag(args, kLineType)
		if err != nil {
			return "", err
		}
		if data.IsTHSIndexCode(code) {
			if raw := strings.TrimSpace(optionalString(args, "adjustFlag", "")); raw != "" && !strings.EqualFold(raw, "none") && raw != "0" {
				return "", fmt.Errorf(".TI 指数没有复权语义；请省略 --adjust 或使用 --adjust none")
			}
			adjustFlag = "none"
		}
		summary, ok, fetchErr := buildKLineSignalSummaryWithError(code, kLineType, limit, adjustFlag)
		if fetchErr != "" {
			return "", fmt.Errorf("%s：%s", code, fetchErr)
		}
		if !ok {
			return fmt.Sprintf("%s：未获取到可计算指标信号的 K 线数据，请检查股票代码、周期或增加 --limit。", code), nil
		}
		return formatKLineSignalSummary(code, kLineType, summary), nil
	}
}

func buildKLineSignalSummary(code, kLineType string, limit int, adjustFlag string) (kLineSignalSummary, bool) {
	summary, ok, _ := buildKLineSignalSummaryWithError(code, kLineType, limit, adjustFlag)
	return summary, ok
}

func buildKLineSignalSummaryWithError(code, kLineType string, limit int, adjustFlag string) (kLineSignalSummary, bool, string) {
	if limit < 180 {
		limit = 180
	}
	result := data.FetchKLineWithFallback(code, "", data.NormalizeKLineType(kLineType), limit, "", adjustFlag)
	if strings.TrimSpace(result.Error) != "" {
		return kLineSignalSummary{}, false, result.Error
	}
	if result.Data == nil || len(*result.Data) < 2 {
		return kLineSignalSummary{}, false, ""
	}
	bars := parseKLineBars(*result.Data)
	if len(bars.close) < 2 {
		return kLineSignalSummary{}, false, ""
	}
	signals := evaluateKLineSignals(bars)
	if len(signals) == 0 {
		return kLineSignalSummary{}, false, ""
	}
	summary := kLineSignalSummary{
		Total:       len(signals),
		Signals:     signals,
		Source:      strings.TrimSpace(result.Source),
		LatestDay:   bars.days[len(bars.days)-1],
		LatestClose: bars.close[len(bars.close)-1],
		AdjustFlag:  adjustFlag,
	}
	for _, signal := range signals {
		switch signal.Signal {
		case "bullish":
			summary.Bullish++
		case "bearish":
			summary.Bearish++
		case "oscillating":
			summary.Oscillating++
		case "neutral":
			summary.Neutral++
		}
	}
	return summary, true, ""
}

func formatKLineSignalSummary(code, kLineType string, summary kLineSignalSummary) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s %s 指标信号汇总\n\n", code, kLineType))
	b.WriteString(fmt.Sprintf("- 最新K线：%s，收盘价：%.2f\n", cliValueOrDash(summary.LatestDay), summary.LatestClose))
	b.WriteString(fmt.Sprintf("- 共 %d 项：看多 %d (%d%%)，看空 %d (%d%%)，震荡 %d (%d%%)，中性 %d (%d%%)\n",
		summary.Total,
		summary.Bullish, signalPercent(summary.Bullish, summary.Total),
		summary.Bearish, signalPercent(summary.Bearish, summary.Total),
		summary.Oscillating, signalPercent(summary.Oscillating, summary.Total),
		summary.Neutral, signalPercent(summary.Neutral, summary.Total),
	))
	if summary.Source != "" {
		b.WriteString("- 数据源：" + summary.Source + "\n")
	}
	if summary.AdjustFlag != "" {
		b.WriteString("- 复权：" + kLineAdjustLabel(summary.AdjustFlag) + "\n")
	}
	b.WriteString("\n## 指标标签\n\n")
	for i, signal := range summary.Signals {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString("`")
		b.WriteString(signal.Name)
		b.WriteString(":")
		b.WriteString(signalLabel(signal.Signal))
		b.WriteString("`")
	}
	rows := make([]kLineSignalRow, 0, len(summary.Signals))
	for _, signal := range summary.Signals {
		rows = append(rows, kLineSignalRow{
			Indicator: signal.Name,
			Signal:    signalLabel(signal.Signal),
			Group:     signal.Group,
		})
	}
	b.WriteString("\n\n")
	b.WriteString(util.MarkdownTableWithTitle("指标明细", rows))
	return strings.TrimSpace(b.String())
}

func signalPercent(count, total int) int {
	if total <= 0 {
		return 0
	}
	return int(math.Round(float64(count) / float64(total) * 100))
}

func signalLabel(signal string) string {
	switch signal {
	case "bullish":
		return "看多"
	case "bearish":
		return "看空"
	case "oscillating":
		return "震荡"
	case "neutral":
		return "中性"
	default:
		return signal
	}
}

type kLineBars struct {
	days   []string
	open   []float64
	close  []float64
	high   []float64
	low    []float64
	volume []float64
}

func parseKLineBars(list []data.KLineData) kLineBars {
	bars := kLineBars{}
	for _, item := range list {
		open, okOpen := parseKLineFloat(item.Open)
		closeValue, okClose := parseKLineFloat(item.Close)
		high, okHigh := parseKLineFloat(item.High)
		low, okLow := parseKLineFloat(item.Low)
		if !okOpen || !okClose || !okHigh || !okLow {
			continue
		}
		volume, _ := parseKLineFloat(item.Volume)
		bars.days = append(bars.days, item.Day)
		bars.open = append(bars.open, open)
		bars.close = append(bars.close, closeValue)
		bars.high = append(bars.high, high)
		bars.low = append(bars.low, low)
		bars.volume = append(bars.volume, volume)
	}
	return bars
}

func evaluateKLineSignals(b kLineBars) []kLineSignal {
	var signals []kLineSignal
	add := func(name, group, signal string) {
		if signal != "" {
			signals = append(signals, kLineSignal{Name: name, Group: group, Signal: signal})
		}
	}
	n := len(b.close)
	c := b.close[n-1]

	ma5, ma10, ma20, ma60 := kSMA(b.close, 5), kSMA(b.close, 10), kSMA(b.close, 20), kSMA(b.close, 60)
	if v5, v10, v20, v60, ok := last4(ma5, ma10, ma20, ma60); ok {
		switch {
		case v5 > v10 && v10 > v20 && v20 > v60:
			add("MA", "趋势", "bullish")
		case v5 < v10 && v10 < v20 && v20 < v60:
			add("MA", "趋势", "bearish")
		case (v5 > v20 && v10 < v60) || (v5 < v20 && v10 > v60):
			add("MA", "趋势", "oscillating")
		default:
			add("MA", "趋势", "neutral")
		}
	}
	if v12, v21, ok := last2(kEMA(b.close, 12), kEMA(b.close, 21)); ok {
		add("EMA", "趋势", compareSignal(v12, v21, "bullish", "bearish", "neutral"))
	}
	if upper, mid, lower := kBollinger(b.close, 20, 2); true {
		if u, m, l, ok := last3(upper, mid, lower); ok {
			switch {
			case c > u:
				add("BOLL", "趋势", "bullish")
			case c < l:
				add("BOLL", "趋势", "bearish")
			case c > m:
				add("BOLL", "趋势", "oscillating")
			default:
				add("BOLL", "趋势", "neutral")
			}
		}
	}
	if v, ok := lastValue(kVWAP(b.high, b.low, b.close, b.volume, 20)); ok {
		add("VWAP", "趋势", compareSignal(c, v, "bullish", "bearish", "neutral"))
	}
	if v, ok := lastValue(kDEMA(b.close, 21)); ok {
		add("DEMA", "趋势", compareSignal(c, v, "bullish", "bearish", "neutral"))
	}
	if v, ok := lastValue(kTEMA(b.close, 21)); ok {
		add("TEMA", "趋势", compareSignal(c, v, "bullish", "bearish", "neutral"))
	}
	if kama := kKAMA(b.close, 10, 2, 30); true {
		if v, pv, ok := lastPrev(kama); ok {
			switch {
			case c > v && v > pv:
				add("KAMA", "趋势", "bullish")
			case c < v && v < pv:
				add("KAMA", "趋势", "bearish")
			default:
				add("KAMA", "趋势", "neutral")
			}
		}
	}
	if hull := kHullMA(b.close, 9); true {
		if v, pv, ok := lastPrev(hull); ok {
			add("HullMA", "趋势", compareSignal(v, pv, "bullish", "bearish", "neutral"))
		}
	}
	if upper, _, lower := kKeltner(b.high, b.low, b.close, 20, 10, 1.5); true {
		if u, l, ok := last2(upper, lower); ok {
			switch {
			case c > u:
				add("Keltner", "趋势", "bullish")
			case c < l:
				add("Keltner", "趋势", "bearish")
			default:
				add("Keltner", "趋势", "oscillating")
			}
		}
	}
	if _, dir := kSupertrend(b.high, b.low, b.close, 10, 3); true {
		if d, ok := lastValue(dir); ok {
			add("SuperTrend", "趋势", directionSignal(d))
		}
	}
	if tenkan, kijun, spanA, senkouB := kIchimoku(b.high, b.low, b.close); true {
		if t, k, a, sb, ok := last4(tenkan, kijun, spanA, senkouB); ok {
			cloudTop, cloudBot := math.Max(a, sb), math.Min(a, sb)
			switch {
			case c > cloudTop && t > k:
				add("Ichimoku", "趋势", "bullish")
			case c < cloudBot && t < k:
				add("Ichimoku", "趋势", "bearish")
			case c >= cloudBot && c <= cloudTop:
				add("Ichimoku", "趋势", "oscillating")
			default:
				add("Ichimoku", "趋势", "neutral")
			}
		}
	}
	if _, dir := kSAR(b.high, b.low, b.close, 0.02, 0.2); true {
		if d, ok := lastValue(dir); ok {
			add("SAR", "趋势", directionSignal(d))
		}
	}
	if upper, _, lower := kDonchian(b.high, b.low, 20); true {
		if u, l, ok := last2(upper, lower); ok {
			switch {
			case c >= u:
				add("Donchian", "趋势", "bullish")
			case c <= l:
				add("Donchian", "趋势", "bearish")
			default:
				add("Donchian", "趋势", "oscillating")
			}
		}
	}
	if jaw, teeth, lips := kAlligator(b.high, b.low, b.close); true {
		if j, t, l, ok := last3(jaw, teeth, lips); ok {
			switch {
			case l > t && t > j:
				add("Alligator", "趋势", "bullish")
			case l < t && t < j:
				add("Alligator", "趋势", "bearish")
			default:
				add("Alligator", "趋势", "oscillating")
			}
		}
	}
	if zzDir := kZigZagDirections(b.high, b.low, b.close, 5); len(zzDir) > 0 {
		lastDir := 0
		for i := len(zzDir) - 1; i >= 0; i-- {
			if zzDir[i] == 1 || zzDir[i] == -1 {
				lastDir = zzDir[i]
				break
			}
		}
		if lastDir == -1 {
			add("ZigZag", "趋势", "bullish")
		} else if lastDir == 1 {
			add("ZigZag", "趋势", "bearish")
		} else {
			add("ZigZag", "趋势", "neutral")
		}
	}
	if dir := kSATSDirection(b.high, b.low, b.close, b.volume); len(dir) > 0 {
		if d, ok := lastValue(dir); ok {
			add("SATS", "趋势", directionSignal(d))
		}
	}
	if pp, s1, r1 := kPivot(b.high, b.low, b.close); true {
		if p, s, r, ok := last3(pp, s1, r1); ok {
			switch {
			case c > r:
				add("Pivot", "趋势", "bullish")
			case c < s:
				add("Pivot", "趋势", "bearish")
			case c > p:
				add("Pivot", "趋势", "oscillating")
			default:
				add("Pivot", "趋势", "neutral")
			}
		}
	}
	if _, upper, lower := kVWAPBands(b.high, b.low, b.close, b.volume, 20, 2); true {
		if u, l, ok := last2(upper, lower); ok {
			switch {
			case c > u:
				add("VWAPBands", "趋势", "bullish")
			case c < l:
				add("VWAPBands", "趋势", "bearish")
			default:
				add("VWAPBands", "趋势", "oscillating")
			}
		}
	}

	if dif, dea, hist := kMACD(b.close); true {
		if d, e, h, ok := last3(dif, dea, hist); ok {
			switch {
			case d > e && h > 0:
				add("MACD", "动量", "bullish")
			case d < e && h < 0:
				add("MACD", "动量", "bearish")
			case (d > 0 && h < 0) || (d < 0 && h > 0):
				add("MACD", "动量", "oscillating")
			default:
				add("MACD", "动量", "neutral")
			}
		}
	}
	if v, ok := lastValue(kRSI(b.close, 14)); ok {
		switch {
		case v > 70 || v < 30:
			add("RSI", "动量", "oscillating")
		case v > 50:
			add("RSI", "动量", "bullish")
		default:
			add("RSI", "动量", "bearish")
		}
	}
	if k, d, j := kKDJ(b.high, b.low, b.close, 9); true {
		if kv, dv, jv, ok := last3(k, d, j); ok {
			switch {
			case jv > kv && kv > dv && kv < 80:
				add("KDJ", "动量", "bullish")
			case jv < kv && kv < dv && kv > 20:
				add("KDJ", "动量", "bearish")
			case kv > 80:
				add("KDJ", "动量", "bearish")
			case kv < 20:
				add("KDJ", "动量", "bullish")
			default:
				add("KDJ", "动量", "oscillating")
			}
		}
	}
	if v, ok := lastValue(kCCI(b.high, b.low, b.close, 20)); ok {
		switch {
		case v > 100:
			add("CCI", "动量", "bullish")
		case v < -100:
			add("CCI", "动量", "bearish")
		default:
			add("CCI", "动量", "oscillating")
		}
	}
	if v, ok := lastValue(kWilliamsR(b.high, b.low, b.close, 14)); ok {
		switch {
		case v < -80:
			add("W%R", "动量", "bullish")
		case v > -20:
			add("W%R", "动量", "bearish")
		default:
			add("W%R", "动量", "oscillating")
		}
	}
	if k, d := kStochRSI(b.close, 14, 14, 3, 3); true {
		if kv, dv, ok := last2(k, d); ok {
			switch {
			case kv < 20 && dv < 20 && kv > dv:
				add("StochRSI", "动量", "bullish")
			case kv > 80 && dv > 80 && kv < dv:
				add("StochRSI", "动量", "bearish")
			default:
				add("StochRSI", "动量", "oscillating")
			}
		}
	}
	if adx, diP, diM := kADX(b.high, b.low, b.close, 14); true {
		if a, p, m, ok := last3(adx, diP, diM); ok {
			switch {
			case a > 25 && p > m:
				add("ADX", "动量", "bullish")
			case a > 25 && p < m:
				add("ADX", "动量", "bearish")
			default:
				add("ADX", "动量", "oscillating")
			}
		}
	}
	if up, down := kAroon(b.high, b.low, 25); true {
		if u, d, ok := last2(up, down); ok {
			switch {
			case u > 70 && d < 30:
				add("Aroon", "动量", "bullish")
			case d > 70 && u < 30:
				add("Aroon", "动量", "bearish")
			default:
				add("Aroon", "动量", "oscillating")
			}
		}
	}
	if v, ok := lastValue(kCMO(b.close, 14)); ok {
		switch {
		case v > 50:
			add("CMO", "动量", "bullish")
		case v < -50:
			add("CMO", "动量", "bearish")
		default:
			add("CMO", "动量", "oscillating")
		}
	}
	if trix := kTRIX(b.close, 15); true {
		sig := kEMALeadingNull(trix, 9)
		if t, s, ok := last2(trix, sig); ok {
			add("TRIX", "动量", compareSignal(t, s, "bullish", "bearish", "neutral"))
		}
	}
	if v, ok := lastValue(kROC(b.close, 12)); ok {
		add("ROC", "动量", compareSignal(v, 0, "bullish", "bearish", "neutral"))
	}
	if cp := kCoppock(b.close); true {
		if v, pv, ok := lastPrev(cp); ok {
			switch {
			case v > 0 && pv <= 0:
				add("Coppock", "动量", "bullish")
			case v < 0:
				add("Coppock", "动量", "bearish")
			default:
				add("Coppock", "动量", "neutral")
			}
		}
	}
	if smi, sig := kSMI(b.high, b.low, b.close); true {
		if s, g, ok := last2(smi, sig); ok {
			switch {
			case s > g && s > 0:
				add("SMI", "动量", "bullish")
			case s < g && s < 0:
				add("SMI", "动量", "bearish")
			default:
				add("SMI", "动量", "oscillating")
			}
		}
	}
	if ao := kAO(b.high, b.low, 5, 34); true {
		if v, pv, ok := lastPrev(ao); ok {
			switch {
			case v > 0 && v > pv:
				add("AO", "动量", "bullish")
			case v < 0 && v < pv:
				add("AO", "动量", "bearish")
			case v > 0 && v < pv:
				add("AO", "动量", "oscillating")
			default:
				add("AO", "动量", "neutral")
			}
		}
	}

	if obv := kOBV(b.close, b.volume); true {
		if v, pv, ok := lastPrev(obv); ok {
			add("OBV", "量价", compareSignal(v, pv, "bullish", "bearish", "neutral"))
		}
	}
	if v, ok := lastValue(kMFI(b.high, b.low, b.close, b.volume, 14)); ok {
		switch {
		case v > 80 || v < 20:
			add("MFI", "量价", "oscillating")
		case v > 50:
			add("MFI", "量价", "bullish")
		default:
			add("MFI", "量价", "bearish")
		}
	}
	if v, ok := lastValue(kCMF(b.high, b.low, b.close, b.volume, 20)); ok {
		switch {
		case v > 0.05:
			add("CMF", "量价", "bullish")
		case v < -0.05:
			add("CMF", "量价", "bearish")
		default:
			add("CMF", "量价", "oscillating")
		}
	}
	if ad := kADLine(b.high, b.low, b.close, b.volume); true {
		if v, pv, ok := lastPrev(ad); ok {
			add("A/D", "量价", compareSignal(v, pv, "bullish", "bearish", "neutral"))
		}
	}
	if v, ok := lastValue(kForceIndex(b.close, b.volume, 13)); ok {
		add("FI", "量价", compareSignal(v, 0, "bullish", "bearish", "neutral"))
	}
	if co := kChaikinOsc(b.high, b.low, b.close, b.volume, 3, 10); true {
		if v, pv, ok := lastPrev(co); ok {
			switch {
			case v > 0 && v > pv:
				add("ChaikinOsc", "量价", "bullish")
			case v < 0 && v < pv:
				add("ChaikinOsc", "量价", "bearish")
			default:
				add("ChaikinOsc", "量价", "oscillating")
			}
		}
	}

	if atr := kATR(b.high, b.low, b.close, 14); true {
		if v, pv, ok := lastPrev(atr); ok {
			if v > pv {
				add("ATR", "波动", "oscillating")
			} else {
				add("ATR", "波动", "neutral")
			}
		}
	}
	if v, ok := lastValue(kCHOP(b.high, b.low, b.close, 14)); ok {
		if v > 61.8 {
			add("CHOP", "波动", "oscillating")
		} else {
			add("CHOP", "波动", "neutral")
		}
	}
	if mi := kMassIndex(b.high, b.low, 9, 9, 25); true {
		if v, pv, ok := lastPrev(mi); ok {
			if pv > 27 && v < 27 {
				add("MassIndex", "波动", "bullish")
			} else {
				add("MassIndex", "波动", "neutral")
			}
		}
	}
	if v, ok := lastValue(kUlcerIndex(b.close, 14)); ok {
		switch {
		case v < 5:
			add("UlcerIndex", "波动", "bullish")
		case v > 15:
			add("UlcerIndex", "波动", "bearish")
		default:
			add("UlcerIndex", "波动", "neutral")
		}
	}
	if bull, bear := kElderRay(b.high, b.low, b.close, 13); true {
		if bp, br, ok := last2(bull, bear); ok {
			switch {
			case bp > 0 && bp > br:
				add("ElderRay", "波动", "bullish")
			case br < 0 && br < bp:
				add("ElderRay", "波动", "bearish")
			default:
				add("ElderRay", "波动", "oscillating")
			}
		}
	}

	return signals
}

func compareSignal(a, b float64, gt, lt, eq string) string {
	if a > b {
		return gt
	}
	if a < b {
		return lt
	}
	return eq
}

func directionSignal(direction float64) string {
	if direction > 0 {
		return "bullish"
	}
	if direction < 0 {
		return "bearish"
	}
	return "neutral"
}

func nanSlice(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = math.NaN()
	}
	return out
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func lastValue(arr []float64) (float64, bool) {
	for i := len(arr) - 1; i >= 0; i-- {
		if finite(arr[i]) {
			return arr[i], true
		}
	}
	return 0, false
}

func lastPrev(arr []float64) (float64, float64, bool) {
	count := 0
	var last, prev float64
	for i := len(arr) - 1; i >= 0; i-- {
		if !finite(arr[i]) {
			continue
		}
		count++
		if count == 1 {
			last = arr[i]
		} else {
			prev = arr[i]
			return last, prev, true
		}
	}
	return 0, 0, false
}

func last2(a, b []float64) (float64, float64, bool) {
	av, okA := lastValue(a)
	bv, okB := lastValue(b)
	return av, bv, okA && okB
}

func last3(a, b, c []float64) (float64, float64, float64, bool) {
	av, okA := lastValue(a)
	bv, okB := lastValue(b)
	cv, okC := lastValue(c)
	return av, bv, cv, okA && okB && okC
}

func last4(a, b, c, d []float64) (float64, float64, float64, float64, bool) {
	av, okA := lastValue(a)
	bv, okB := lastValue(b)
	cv, okC := lastValue(c)
	dv, okD := lastValue(d)
	return av, bv, cv, dv, okA && okB && okC && okD
}

func kSMA(values []float64, period int) []float64 {
	out := nanSlice(len(values))
	for i := period - 1; i < len(values); i++ {
		sum := 0.0
		ok := true
		for j := 0; j < period; j++ {
			v := values[i-j]
			if !finite(v) {
				ok = false
				break
			}
			sum += v
		}
		if ok {
			out[i] = sum / float64(period)
		}
	}
	return out
}

func kEMA(values []float64, period int) []float64 {
	out := nanSlice(len(values))
	k := 2.0 / float64(period+1)
	ema := math.NaN()
	for i, v := range values {
		if !finite(v) {
			continue
		}
		if !finite(ema) {
			if i < period-1 {
				continue
			}
			sum := 0.0
			ok := true
			for j := i - period + 1; j <= i; j++ {
				if !finite(values[j]) {
					ok = false
					break
				}
				sum += values[j]
			}
			if !ok {
				continue
			}
			ema = sum / float64(period)
		} else {
			ema = v*k + ema*(1-k)
		}
		out[i] = ema
	}
	return out
}

func kEMALeadingNull(values []float64, period int) []float64 {
	out := nanSlice(len(values))
	k := 2.0 / float64(period+1)
	ema := math.NaN()
	sum := 0.0
	count := 0
	for i, v := range values {
		if !finite(v) {
			continue
		}
		if !finite(ema) {
			sum += v
			count++
			if count < period {
				continue
			}
			ema = sum / float64(period)
		} else {
			ema = v*k + ema*(1-k)
		}
		out[i] = ema
	}
	return out
}

func kWMA(values []float64, period int) []float64 {
	out := nanSlice(len(values))
	denom := float64(period * (period + 1) / 2)
	for i := period - 1; i < len(values); i++ {
		sum := 0.0
		ok := true
		for j := 0; j < period; j++ {
			v := values[i-period+1+j]
			if !finite(v) {
				ok = false
				break
			}
			sum += v * float64(j+1)
		}
		if ok {
			out[i] = sum / denom
		}
	}
	return out
}

func kBollinger(closes []float64, period int, mult float64) ([]float64, []float64, []float64) {
	mid := kSMA(closes, period)
	upper, lower := nanSlice(len(closes)), nanSlice(len(closes))
	for i := period - 1; i < len(closes); i++ {
		if !finite(mid[i]) {
			continue
		}
		sumSq := 0.0
		ok := true
		for j := 0; j < period; j++ {
			v := closes[i-j]
			if !finite(v) {
				ok = false
				break
			}
			d := v - mid[i]
			sumSq += d * d
		}
		if ok {
			std := math.Sqrt(sumSq / float64(period))
			upper[i] = mid[i] + mult*std
			lower[i] = mid[i] - mult*std
		}
	}
	return upper, mid, lower
}

func kOBV(closes, vols []float64) []float64 {
	if len(closes) == 0 {
		return nil
	}
	out := nanSlice(len(closes))
	obv := 0.0
	if len(vols) > 0 && finite(vols[0]) {
		obv = vols[0]
	}
	out[0] = obv
	for i := 1; i < len(closes); i++ {
		vol := 0.0
		if i < len(vols) && finite(vols[i]) {
			vol = vols[i]
		}
		if closes[i] > closes[i-1] {
			obv += vol
		} else if closes[i] < closes[i-1] {
			obv -= vol
		}
		out[i] = obv
	}
	return out
}

func kMACD(closes []float64) ([]float64, []float64, []float64) {
	ema12, ema26 := kEMA(closes, 12), kEMA(closes, 26)
	dif := nanSlice(len(closes))
	for i := range closes {
		if finite(ema12[i]) && finite(ema26[i]) {
			dif[i] = ema12[i] - ema26[i]
		}
	}
	dea := kEMALeadingNull(dif, 9)
	hist := nanSlice(len(closes))
	for i := range closes {
		if finite(dif[i]) && finite(dea[i]) {
			hist[i] = 2 * (dif[i] - dea[i])
		}
	}
	return dif, dea, hist
}

func kKDJ(highs, lows, closes []float64, period int) ([]float64, []float64, []float64) {
	n := len(closes)
	rsv, k, d, jv := nanSlice(n), nanSlice(n), nanSlice(n), nanSlice(n)
	for i := period - 1; i < n; i++ {
		hi, lo := -math.MaxFloat64, math.MaxFloat64
		for j := 0; j < period; j++ {
			hi = math.Max(hi, highs[i-j])
			lo = math.Min(lo, lows[i-j])
		}
		if hi == lo {
			rsv[i] = 50
		} else {
			rsv[i] = (closes[i] - lo) / (hi - lo) * 100
		}
	}
	pk, pd := 50.0, 50.0
	for i := 0; i < n; i++ {
		if !finite(rsv[i]) {
			continue
		}
		pk = (2*pk + rsv[i]) / 3
		pd = (2*pd + pk) / 3
		k[i], d[i], jv[i] = pk, pd, 3*pk-2*pd
	}
	return k, d, jv
}

func kRSI(closes []float64, period int) []float64 {
	out := nanSlice(len(closes))
	for i := period; i < len(closes); i++ {
		gain, loss := 0.0, 0.0
		for j := 0; j < period; j++ {
			ch := closes[i-j] - closes[i-j-1]
			if ch >= 0 {
				gain += ch
			} else {
				loss -= ch
			}
		}
		ag, al := gain/float64(period), loss/float64(period)
		if al == 0 {
			out[i] = 100
		} else {
			out[i] = 100 - 100/(1+ag/al)
		}
	}
	return out
}

func kATR(highs, lows, closes []float64, period int) []float64 {
	n := len(closes)
	out := nanSlice(n)
	if n < 2 {
		return out
	}
	tr := nanSlice(n)
	tr[0] = highs[0] - lows[0]
	for i := 1; i < n; i++ {
		tr[i] = math.Max(highs[i]-lows[i], math.Max(math.Abs(highs[i]-closes[i-1]), math.Abs(lows[i]-closes[i-1])))
	}
	sum := 0.0
	for i := 0; i < period && i < n; i++ {
		sum += tr[i]
	}
	if n >= period {
		out[period-1] = sum / float64(period)
		for i := period; i < n; i++ {
			out[i] = (out[i-1]*float64(period-1) + tr[i]) / float64(period)
		}
	}
	return out
}

func kVWAP(highs, lows, closes, vols []float64, period int) []float64 {
	out := nanSlice(len(closes))
	for i := period - 1; i < len(closes); i++ {
		sumPV, sumV := 0.0, 0.0
		for j := 0; j < period; j++ {
			idx := i - j
			tp := (highs[idx] + lows[idx] + closes[idx]) / 3
			sumPV += tp * vols[idx]
			sumV += vols[idx]
		}
		if sumV > 0 {
			out[i] = sumPV / sumV
		}
	}
	return out
}

func kMFI(highs, lows, closes, vols []float64, period int) []float64 {
	n := len(closes)
	out := nanSlice(n)
	if n < 2 {
		return out
	}
	tp, mf := make([]float64, n), make([]float64, n)
	for i := range closes {
		tp[i] = (highs[i] + lows[i] + closes[i]) / 3
		mf[i] = tp[i] * vols[i]
	}
	for i := period; i < n; i++ {
		pos, neg := 0.0, 0.0
		for j := 0; j < period; j++ {
			idx := i - j
			if tp[idx] > tp[idx-1] {
				pos += mf[idx]
			} else if tp[idx] < tp[idx-1] {
				neg += mf[idx]
			}
		}
		if neg == 0 {
			out[i] = 100
		} else {
			out[i] = 100 - 100/(1+pos/neg)
		}
	}
	return out
}

func kKAMA(closes []float64, period, fastPeriod, slowPeriod int) []float64 {
	n := len(closes)
	out := nanSlice(n)
	if n < period+1 {
		return out
	}
	fastSC, slowSC := 2.0/float64(fastPeriod+1), 2.0/float64(slowPeriod+1)
	kama := closes[period]
	out[period] = kama
	for i := period + 1; i < n; i++ {
		direction := math.Abs(closes[i] - closes[i-period])
		volatility := 0.0
		for j := 0; j < period; j++ {
			volatility += math.Abs(closes[i-j] - closes[i-j-1])
		}
		er := 0.0
		if volatility > 0 {
			er = direction / volatility
		}
		sc := math.Pow(er*(fastSC-slowSC)+slowSC, 2)
		kama += sc * (closes[i] - kama)
		out[i] = kama
	}
	return out
}

func kKeltner(highs, lows, closes []float64, emaPeriod, atrPeriod int, mult float64) ([]float64, []float64, []float64) {
	mid := kEMA(closes, emaPeriod)
	atr := kATR(highs, lows, closes, atrPeriod)
	upper, lower := nanSlice(len(closes)), nanSlice(len(closes))
	for i := range closes {
		if finite(mid[i]) && finite(atr[i]) {
			upper[i] = mid[i] + mult*atr[i]
			lower[i] = mid[i] - mult*atr[i]
		}
	}
	return upper, mid, lower
}

func kSupertrend(highs, lows, closes []float64, atrPeriod int, multiplier float64) ([]float64, []float64) {
	n := len(closes)
	atr := kATR(highs, lows, closes, atrPeriod)
	st, dir := nanSlice(n), nanSlice(n)
	prevUpper, prevLower, prevDir := math.NaN(), math.NaN(), 0.0
	for i := 0; i < n; i++ {
		if !finite(atr[i]) {
			continue
		}
		hl2 := (highs[i] + lows[i]) / 2
		rawUpper := hl2 + multiplier*atr[i]
		rawLower := hl2 - multiplier*atr[i]
		if finite(prevUpper) && rawUpper >= prevUpper && i > 0 && closes[i-1] <= prevUpper {
			rawUpper = prevUpper
		}
		if finite(prevLower) && rawLower <= prevLower && i > 0 && closes[i-1] >= prevLower {
			rawLower = prevLower
		}
		d := 1.0
		if prevDir == 1 {
			if closes[i] < rawLower {
				d = -1
			}
		} else if prevDir == -1 {
			d = -1
			if closes[i] > rawUpper {
				d = 1
			}
		}
		if d == 1 {
			st[i] = rawLower
		} else {
			st[i] = rawUpper
		}
		dir[i] = d
		prevUpper, prevLower, prevDir = rawUpper, rawLower, d
	}
	return st, dir
}

func kIchimoku(highs, lows, closes []float64) ([]float64, []float64, []float64, []float64) {
	periodHL := func(period int) []float64 {
		out := nanSlice(len(closes))
		for i := period - 1; i < len(closes); i++ {
			hi, lo := -math.MaxFloat64, math.MaxFloat64
			for j := 0; j < period; j++ {
				hi = math.Max(hi, highs[i-j])
				lo = math.Min(lo, lows[i-j])
			}
			out[i] = (hi + lo) / 2
		}
		return out
	}
	tenkan, kijun, senkouB := periodHL(9), periodHL(26), periodHL(52)
	spanA := nanSlice(len(closes))
	for i := range closes {
		if finite(tenkan[i]) && finite(kijun[i]) {
			spanA[i] = (tenkan[i] + kijun[i]) / 2
		}
	}
	return tenkan, kijun, spanA, senkouB
}

func kCCI(highs, lows, closes []float64, period int) []float64 {
	n := len(closes)
	tp := make([]float64, n)
	out := nanSlice(n)
	for i := range closes {
		tp[i] = (highs[i] + lows[i] + closes[i]) / 3
	}
	for i := period - 1; i < n; i++ {
		sum := 0.0
		for j := 0; j < period; j++ {
			sum += tp[i-j]
		}
		mean := sum / float64(period)
		meanDev := 0.0
		for j := 0; j < period; j++ {
			meanDev += math.Abs(tp[i-j] - mean)
		}
		meanDev /= float64(period)
		if meanDev > 0 {
			out[i] = (tp[i] - mean) / (0.015 * meanDev)
		}
	}
	return out
}

func kSAR(highs, lows, closes []float64, step, maxStep float64) ([]float64, []float64) {
	n := len(closes)
	sar, dir := nanSlice(n), nanSlice(n)
	if n < 2 {
		return sar, dir
	}
	isLong := closes[1] > closes[0]
	af := step
	ep := lows[1]
	prevSar := highs[0]
	if isLong {
		ep, prevSar = highs[1], lows[0]
	}
	sar[1] = prevSar
	if isLong {
		dir[1] = 1
	} else {
		dir[1] = -1
	}
	for i := 2; i < n; i++ {
		curSar := prevSar + af*(ep-prevSar)
		if isLong {
			curSar = math.Min(curSar, math.Min(lows[i-1], lows[i-2]))
			if lows[i] < curSar {
				isLong = false
				curSar = ep
				ep = lows[i]
				af = step
			} else if highs[i] > ep {
				ep = highs[i]
				af = math.Min(af+step, maxStep)
			}
		} else {
			curSar = math.Max(curSar, math.Max(highs[i-1], highs[i-2]))
			if highs[i] > curSar {
				isLong = true
				curSar = ep
				ep = highs[i]
				af = step
			} else if lows[i] < ep {
				ep = lows[i]
				af = math.Min(af+step, maxStep)
			}
		}
		sar[i] = curSar
		if isLong {
			dir[i] = 1
		} else {
			dir[i] = -1
		}
		prevSar = curSar
	}
	return sar, dir
}

func kDonchian(highs, lows []float64, period int) ([]float64, []float64, []float64) {
	n := len(highs)
	upper, mid, lower := nanSlice(n), nanSlice(n), nanSlice(n)
	for i := period - 1; i < n; i++ {
		hi, lo := -math.MaxFloat64, math.MaxFloat64
		for j := 0; j < period; j++ {
			hi = math.Max(hi, highs[i-j])
			lo = math.Min(lo, lows[i-j])
		}
		upper[i], lower[i], mid[i] = hi, lo, (hi+lo)/2
	}
	return upper, mid, lower
}

func kADX(highs, lows, closes []float64, period int) ([]float64, []float64, []float64) {
	n := len(closes)
	adx, diP, diM := nanSlice(n), nanSlice(n), nanSlice(n)
	if n < 2 {
		return adx, diP, diM
	}
	tr, plusDM, minusDM := make([]float64, n), make([]float64, n), make([]float64, n)
	tr[0] = highs[0] - lows[0]
	for i := 1; i < n; i++ {
		tr[i] = math.Max(highs[i]-lows[i], math.Max(math.Abs(highs[i]-closes[i-1]), math.Abs(lows[i]-closes[i-1])))
		upMove, downMove := highs[i]-highs[i-1], lows[i-1]-lows[i]
		if upMove > downMove && upMove > 0 {
			plusDM[i] = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDM[i] = downMove
		}
	}
	sTR, sPDM, sMDM := 0.0, 0.0, 0.0
	smoothTR, smoothPDM, smoothMDM := nanSlice(n), nanSlice(n), nanSlice(n)
	for i := 0; i < period && i < n; i++ {
		sTR += tr[i]
		sPDM += plusDM[i]
		sMDM += minusDM[i]
	}
	if n >= period {
		smoothTR[period-1], smoothPDM[period-1], smoothMDM[period-1] = sTR, sPDM, sMDM
		for i := period; i < n; i++ {
			smoothTR[i] = smoothTR[i-1] - smoothTR[i-1]/float64(period) + tr[i]
			smoothPDM[i] = smoothPDM[i-1] - smoothPDM[i-1]/float64(period) + plusDM[i]
			smoothMDM[i] = smoothMDM[i-1] - smoothMDM[i-1]/float64(period) + minusDM[i]
		}
	}
	dx := nanSlice(n)
	for i := 0; i < n; i++ {
		if finite(smoothTR[i]) && smoothTR[i] > 0 {
			diP[i] = 100 * smoothPDM[i] / smoothTR[i]
			diM[i] = 100 * smoothMDM[i] / smoothTR[i]
			sum := diP[i] + diM[i]
			if sum > 0 {
				dx[i] = 100 * math.Abs(diP[i]-diM[i]) / sum
			} else {
				dx[i] = 0
			}
		}
	}
	if n >= period*2-1 {
		sumDX := 0.0
		for i := period - 1; i < period*2-1 && i < n; i++ {
			if finite(dx[i]) {
				sumDX += dx[i]
			}
		}
		adx[period*2-2] = sumDX / float64(period)
		for i := period*2 - 1; i < n; i++ {
			v := 0.0
			if finite(dx[i]) {
				v = dx[i]
			}
			adx[i] = (adx[i-1]*float64(period-1) + v) / float64(period)
		}
	}
	return adx, diP, diM
}

func kWilliamsR(highs, lows, closes []float64, period int) []float64 {
	out := nanSlice(len(closes))
	for i := period - 1; i < len(closes); i++ {
		hi, lo := -math.MaxFloat64, math.MaxFloat64
		for j := 0; j < period; j++ {
			hi = math.Max(hi, highs[i-j])
			lo = math.Min(lo, lows[i-j])
		}
		if hi > lo {
			out[i] = ((hi - closes[i]) / (hi - lo)) * -100
		}
	}
	return out
}

func kStochRSI(closes []float64, rsiPeriod, stochPeriod, kSmooth, dSmooth int) ([]float64, []float64) {
	rsi := kRSI(closes, rsiPeriod)
	n := len(closes)
	stoch, k, d := nanSlice(n), nanSlice(n), nanSlice(n)
	for i := stochPeriod - 1; i < n; i++ {
		minR, maxR := math.MaxFloat64, -math.MaxFloat64
		ok := true
		for j := 0; j < stochPeriod; j++ {
			v := rsi[i-j]
			if !finite(v) {
				ok = false
				break
			}
			minR, maxR = math.Min(minR, v), math.Max(maxR, v)
		}
		if ok {
			if maxR != minR {
				stoch[i] = (rsi[i] - minR) / (maxR - minR) * 100
			} else {
				stoch[i] = 0
			}
		}
	}
	for i := range stoch {
		if !finite(stoch[i]) {
			continue
		}
		sum, cnt := 0.0, 0
		for j := 0; j < kSmooth && i-j >= 0; j++ {
			if finite(stoch[i-j]) {
				sum += stoch[i-j]
				cnt++
			}
		}
		if cnt == kSmooth {
			k[i] = sum / float64(cnt)
		}
	}
	for i := range k {
		if !finite(k[i]) {
			continue
		}
		sum, cnt := 0.0, 0
		for j := 0; j < dSmooth && i-j >= 0; j++ {
			if finite(k[i-j]) {
				sum += k[i-j]
				cnt++
			}
		}
		if cnt == dSmooth {
			d[i] = sum / float64(cnt)
		}
	}
	return k, d
}

func kCMF(highs, lows, closes, vols []float64, period int) []float64 {
	out := nanSlice(len(closes))
	for i := period - 1; i < len(closes); i++ {
		sumMFV, sumVol := 0.0, 0.0
		for j := 0; j < period; j++ {
			idx := i - j
			r := highs[idx] - lows[idx]
			mfv := 0.0
			if r > 0 {
				mfv = ((closes[idx] - lows[idx]) - (highs[idx] - closes[idx])) / r * vols[idx]
			}
			sumMFV += mfv
			sumVol += vols[idx]
		}
		if sumVol > 0 {
			out[i] = sumMFV / sumVol
		}
	}
	return out
}

func kAroon(highs, lows []float64, period int) ([]float64, []float64) {
	up, down := nanSlice(len(highs)), nanSlice(len(highs))
	for i := period - 1; i < len(highs); i++ {
		highIdx, lowIdx := 0, 0
		for j := 1; j < period; j++ {
			if highs[i-j] > highs[i-highIdx] {
				highIdx = j
			}
			if lows[i-j] < lows[i-lowIdx] {
				lowIdx = j
			}
		}
		up[i] = float64(period-1-highIdx) / float64(period-1) * 100
		down[i] = float64(period-1-lowIdx) / float64(period-1) * 100
	}
	return up, down
}

func kCMO(closes []float64, period int) []float64 {
	out := nanSlice(len(closes))
	for i := period; i < len(closes); i++ {
		up, down := 0.0, 0.0
		for j := 0; j < period; j++ {
			diff := closes[i-j] - closes[i-j-1]
			if diff > 0 {
				up += diff
			} else {
				down -= diff
			}
		}
		if up+down > 0 {
			out[i] = (up - down) / (up + down) * 100
		} else {
			out[i] = 0
		}
	}
	return out
}

func kForceIndex(closes, vols []float64, period int) []float64 {
	raw := nanSlice(len(closes))
	if len(closes) == 0 {
		return raw
	}
	raw[0] = 0
	for i := 1; i < len(closes); i++ {
		raw[i] = (closes[i] - closes[i-1]) * vols[i]
	}
	return kEMA(raw, period)
}

func kPivot(highs, lows, closes []float64) ([]float64, []float64, []float64) {
	n := len(closes)
	pp, s1, r1 := nanSlice(n), nanSlice(n), nanSlice(n)
	for i := 1; i < n; i++ {
		p := (highs[i-1] + lows[i-1] + closes[i-1]) / 3
		pp[i] = p
		r1[i] = 2*p - lows[i-1]
		s1[i] = 2*p - highs[i-1]
	}
	return pp, s1, r1
}

func kDEMA(closes []float64, period int) []float64 {
	e1 := kEMA(closes, period)
	e1Fill := make([]float64, len(e1))
	for i, v := range e1 {
		if finite(v) {
			e1Fill[i] = v
		}
	}
	e2 := kEMA(e1Fill, period)
	out := nanSlice(len(closes))
	for i := range closes {
		if finite(e1[i]) && finite(e2[i]) {
			out[i] = 2*e1[i] - e2[i]
		}
	}
	return out
}

func kZigZagDirections(highs, lows, closes []float64, threshold float64) []int {
	n := len(closes)
	dirs := make([]int, n)
	if n < 3 {
		return dirs
	}
	type point struct {
		idx    int
		price  float64
		isHigh bool
	}
	points := []point{{idx: 0, price: highs[0], isHigh: true}}
	lastHigh := point{idx: 0, price: highs[0], isHigh: true}
	lastLow := point{idx: 0, price: lows[0], isHigh: false}
	lookingHigh := true
	for i := 1; i < n; i++ {
		if lookingHigh {
			if highs[i] >= lastHigh.price {
				lastHigh = point{idx: i, price: highs[i], isHigh: true}
				points[len(points)-1] = lastHigh
			} else if lastHigh.price-lows[i] >= lastHigh.price*threshold/100 {
				points = append(points, lastHigh)
				lastLow = point{idx: i, price: lows[i], isHigh: false}
				lookingHigh = false
			}
		} else {
			if lows[i] <= lastLow.price {
				lastLow = point{idx: i, price: lows[i], isHigh: false}
				points[len(points)-1] = lastLow
			} else if highs[i]-lastLow.price >= lastLow.price*threshold/100 {
				points = append(points, lastLow)
				lastHigh = point{idx: i, price: highs[i], isHigh: true}
				lookingHigh = true
			}
		}
	}
	for _, p := range points {
		if p.idx >= 0 && p.idx < n {
			if p.isHigh {
				dirs[p.idx] = 1
			} else {
				dirs[p.idx] = -1
			}
		}
	}
	return dirs
}

func kSATSDirection(highs, lows, closes, vols []float64) []float64 {
	n := len(closes)
	rawATR := kATR(highs, lows, closes, 14)
	atrBase := kSMA(rawATR, 100)
	dir := nanSlice(n)
	prevLower, prevUpper, prevDir := math.NaN(), math.NaN(), 0.0
	prevActive, prevPassive := math.NaN(), math.NaN()
	trendStart := 0
	for i := 0; i < n; i++ {
		if !finite(rawATR[i]) || !finite(atrBase[i]) {
			continue
		}
		atrVal := rawATR[i]
		volRatio := 1.0
		if atrBase[i] != 0 {
			volRatio = atrVal / atrBase[i]
		}
		er := 0.0
		if i >= 20 {
			change := math.Abs(closes[i] - closes[i-20])
			volatility := 0.0
			for j := 0; j < 20; j++ {
				volatility += math.Abs(closes[i-j] - closes[i-j-1])
			}
			if volatility != 0 {
				er = change / volatility
			}
		}
		effATR := atrVal * (0.5 + 0.5*er)
		tqiER := clamp(er, 0, 1)
		tqiVol := clamp((volRatio-0.6)/(1.8-0.6), 0, 1)
		if i >= 20 && vols[i] > 0 {
			mean := 0.0
			for j := 0; j < 20; j++ {
				mean += vols[i-j]
			}
			mean /= 20
			varSq := 0.0
			for j := 0; j < 20; j++ {
				d := vols[i-j] - mean
				varSq += d * d
			}
			std := math.Sqrt(varSq / 20)
			volZ := 0.0
			if std != 0 {
				volZ = (vols[i] - mean) / std
			}
			tqiVol = clamp((volZ+1)/3, 0, 1)
		}
		tqiStruct := 0.0
		if i >= 20 {
			hi, lo := -math.MaxFloat64, math.MaxFloat64
			for j := 0; j < 20; j++ {
				hi = math.Max(hi, highs[i-j])
				lo = math.Min(lo, lows[i-j])
			}
			if hi > lo {
				pos := (closes[i] - lo) / (hi - lo)
				tqiStruct = clamp(math.Abs(pos-0.5)*2, 0, 1)
			}
		}
		tqiMom := 0.0
		if i >= 10 {
			windowChange := closes[i] - closes[i-10]
			aligned := 0
			for j := 0; j < 10; j++ {
				ch := closes[i-j] - closes[i-j-1]
				if (windowChange > 0 && ch > 0) || (windowChange < 0 && ch < 0) {
					aligned++
				}
			}
			tqiMom = float64(aligned) / 10
		}
		tqi := clamp((tqiER*0.35 + tqiVol*0.20 + tqiStruct*0.25 + tqiMom*0.20), 0, 1)
		legacyAdapt := 1 + 0.5*(0.5-er)
		qualityDeviation := math.Pow(1-tqi, 1.5)
		tqiMult := 1 - 0.4 + 0.4*(0.6+0.8*qualityDeviation)
		symMult := 2.0 * legacyAdapt * tqiMult
		activeRaw := symMult * (1 - 0.5*tqi*0.3)
		passiveRaw := symMult * (1 + 0.5*tqi*0.4)
		active := activeRaw
		passive := passiveRaw
		if finite(prevActive) {
			active = prevActive*0.85 + activeRaw*0.15
		}
		if finite(prevPassive) {
			passive = prevPassive*0.85 + passiveRaw*0.15
		}
		prevActive, prevPassive = active, passive
		curPrevDir := prevDir
		if curPrevDir == 0 {
			curPrevDir = 1
		}
		lowerMult, upperMult := active, passive
		if curPrevDir != 1 {
			lowerMult, upperMult = passive, active
		}
		hl2 := (highs[i] + lows[i]) / 2
		lower := hl2 - lowerMult*effATR
		upper := hl2 + upperMult*effATR
		if finite(prevLower) && i > 0 && closes[i-1] > prevLower {
			lower = math.Max(lower, prevLower)
		}
		if finite(prevUpper) && i > 0 && closes[i-1] < prevUpper {
			upper = math.Min(upper, prevUpper)
		}
		flipUp := prevDir == -1 && finite(prevUpper) && closes[i] > prevUpper
		flipDown := prevDir == 1 && finite(prevLower) && closes[i] < prevLower
		prevTQI := 0.5
		trendAge := i - trendStart
		charBase := prevTQI > 0.55 && tqi < 0.25 && trendAge >= 5
		charDown := charBase && curPrevDir == 1 && i > 0 && closes[i] < closes[i-1]
		charUp := charBase && curPrevDir == -1 && i > 0 && closes[i] > closes[i-1]
		nextDir := curPrevDir
		if flipUp || charUp {
			nextDir = 1
		} else if flipDown || charDown {
			nextDir = -1
		}
		if nextDir != curPrevDir {
			trendStart = i
		}
		prevLower, prevUpper, prevDir = lower, upper, nextDir
		dir[i] = nextDir
	}
	return dir
}

func kAlligator(highs, lows, closes []float64) ([]float64, []float64, []float64) {
	n := len(closes)
	mid := make([]float64, n)
	for i := range closes {
		mid[i] = (highs[i] + lows[i]) / 2
	}
	shift := func(raw []float64, offset int) []float64 {
		out := nanSlice(n)
		for i := offset; i < n; i++ {
			if finite(raw[i-offset]) {
				out[i] = raw[i-offset]
			}
		}
		return out
	}
	return shift(kSMA(mid, 13), 8), shift(kSMA(mid, 8), 5), shift(kSMA(mid, 5), 3)
}

func kAO(highs, lows []float64, fast, slow int) []float64 {
	mid := make([]float64, len(highs))
	for i := range highs {
		mid[i] = (highs[i] + lows[i]) / 2
	}
	fastSMA, slowSMA := kSMA(mid, fast), kSMA(mid, slow)
	out := nanSlice(len(highs))
	for i := range highs {
		if finite(fastSMA[i]) && finite(slowSMA[i]) {
			out[i] = fastSMA[i] - slowSMA[i]
		}
	}
	return out
}

func kHullMA(closes []float64, period int) []float64 {
	half, sqrtLen := period/2, int(math.Sqrt(float64(period)))
	wmaHalf, wmaFull := kWMA(closes, half), kWMA(closes, period)
	diff := nanSlice(len(closes))
	for i := range closes {
		if finite(wmaHalf[i]) && finite(wmaFull[i]) {
			diff[i] = 2*wmaHalf[i] - wmaFull[i]
		}
	}
	return kWMA(diff, sqrtLen)
}

func kADLine(highs, lows, closes, vols []float64) []float64 {
	out := make([]float64, len(closes))
	for i := range closes {
		r := highs[i] - lows[i]
		mfm := 0.0
		if r > 0 {
			mfm = ((closes[i] - lows[i]) - (highs[i] - closes[i])) / r
		}
		mfv := mfm * vols[i]
		if i > 0 {
			out[i] = out[i-1] + mfv
		} else {
			out[i] = mfv
		}
	}
	return out
}

func kTRIX(closes []float64, period int) []float64 {
	ema1 := kEMA(closes, period)
	ema2 := kEMA(ema1, period)
	ema3 := kEMA(ema2, period)
	out := nanSlice(len(closes))
	for i := 1; i < len(closes); i++ {
		if finite(ema3[i]) && finite(ema3[i-1]) && ema3[i-1] != 0 {
			out[i] = (ema3[i] - ema3[i-1]) / ema3[i-1] * 10000
		}
	}
	return out
}

func kROC(closes []float64, period int) []float64 {
	out := nanSlice(len(closes))
	for i := period; i < len(closes); i++ {
		if closes[i-period] != 0 {
			out[i] = (closes[i] - closes[i-period]) / closes[i-period] * 100
		}
	}
	return out
}

func kCHOP(highs, lows, closes []float64, period int) []float64 {
	out := nanSlice(len(closes))
	for i := period - 1; i < len(closes); i++ {
		atrSum := 0.0
		for j := 0; j < period; j++ {
			idx := i - j
			tr := highs[idx] - lows[idx]
			if idx > 0 {
				tr = math.Max(tr, math.Max(math.Abs(highs[idx]-closes[idx-1]), math.Abs(lows[idx]-closes[idx-1])))
			}
			atrSum += tr
		}
		hi, lo := -math.MaxFloat64, math.MaxFloat64
		for j := i - period + 1; j <= i; j++ {
			hi = math.Max(hi, highs[j])
			lo = math.Min(lo, lows[j])
		}
		if hi > lo && atrSum > 0 {
			out[i] = 100 * math.Log(atrSum/(hi-lo)) / math.Log(float64(period))
		}
	}
	return out
}

func kElderRay(highs, lows, closes []float64, period int) ([]float64, []float64) {
	ema := kEMA(closes, period)
	bull, bear := nanSlice(len(closes)), nanSlice(len(closes))
	for i := range closes {
		if finite(ema[i]) {
			bull[i] = highs[i] - ema[i]
			bear[i] = lows[i] - ema[i]
		}
	}
	return bull, bear
}

func kChaikinOsc(highs, lows, closes, vols []float64, fast, slow int) []float64 {
	ad := kADLine(highs, lows, closes, vols)
	fastEMA, slowEMA := kEMA(ad, fast), kEMA(ad, slow)
	out := nanSlice(len(closes))
	for i := range closes {
		if finite(fastEMA[i]) && finite(slowEMA[i]) {
			out[i] = fastEMA[i] - slowEMA[i]
		}
	}
	return out
}

func kVWAPBands(highs, lows, closes, vols []float64, period int, mult float64) ([]float64, []float64, []float64) {
	vwap := kVWAP(highs, lows, closes, vols, period)
	upper, lower := nanSlice(len(closes)), nanSlice(len(closes))
	for i := range closes {
		if !finite(vwap[i]) {
			continue
		}
		sumSq, cnt := 0.0, 0.0
		start := i - period + 1
		if start < 0 {
			start = 0
		}
		for j := start; j <= i; j++ {
			tp := (highs[j] + lows[j] + closes[j]) / 3
			weight := vols[j]
			if weight <= 0 {
				weight = 1
			}
			diff := tp - vwap[i]
			sumSq += diff * diff * weight
			cnt += weight
		}
		if cnt > 0 {
			std := math.Sqrt(sumSq / cnt)
			upper[i] = vwap[i] + mult*std
			lower[i] = vwap[i] - mult*std
		}
	}
	return vwap, upper, lower
}

func kMassIndex(highs, lows []float64, emaPeriod, emaPeriod2, sumPeriod int) []float64 {
	ranges := make([]float64, len(highs))
	for i := range highs {
		ranges[i] = highs[i] - lows[i]
	}
	single := kEMA(ranges, emaPeriod)
	double := kEMA(single, emaPeriod)
	ratio := nanSlice(len(highs))
	for i := range highs {
		if finite(single[i]) && finite(double[i]) && double[i] != 0 {
			ratio[i] = single[i] / double[i]
		}
	}
	ratioEMA := kEMA(ratio, emaPeriod2)
	out := nanSlice(len(highs))
	for i := sumPeriod - 1; i < len(highs); i++ {
		sum := 0.0
		ok := true
		for j := 0; j < sumPeriod; j++ {
			if !finite(ratioEMA[i-j]) {
				ok = false
				break
			}
			sum += ratioEMA[i-j]
		}
		if ok {
			out[i] = sum
		}
	}
	return out
}

func kUlcerIndex(closes []float64, period int) []float64 {
	out := nanSlice(len(closes))
	for i := period - 1; i < len(closes); i++ {
		maxClose := -math.MaxFloat64
		for j := 0; j < period; j++ {
			maxClose = math.Max(maxClose, closes[i-j])
		}
		sumSq := 0.0
		for j := 0; j < period; j++ {
			drawdown := (closes[i-j] - maxClose) / maxClose * 100
			sumSq += drawdown * drawdown
		}
		out[i] = math.Sqrt(sumSq / float64(period))
	}
	return out
}

func kCoppock(closes []float64) []float64 {
	rocA, rocB := nanSlice(len(closes)), nanSlice(len(closes))
	for i := 14; i < len(closes); i++ {
		if closes[i-14] != 0 {
			rocA[i] = (closes[i] - closes[i-14]) / closes[i-14] * 100
		}
	}
	for i := 11; i < len(closes); i++ {
		if closes[i-11] != 0 {
			rocB[i] = (closes[i] - closes[i-11]) / closes[i-11] * 100
		}
	}
	sum := nanSlice(len(closes))
	for i := range closes {
		if finite(rocA[i]) && finite(rocB[i]) {
			sum[i] = rocA[i] + rocB[i]
		}
	}
	return kWMA(sum, 10)
}

func kTEMA(closes []float64, period int) []float64 {
	ema1 := kEMA(closes, period)
	ema2 := kEMA(ema1, period)
	ema3 := kEMA(ema2, period)
	out := nanSlice(len(closes))
	for i := range closes {
		if finite(ema1[i]) && finite(ema2[i]) && finite(ema3[i]) {
			out[i] = ema1[i] + (ema1[i] - ema2[i]) + ((ema1[i] - ema2[i]) - (ema2[i] - ema3[i]))
		}
	}
	return out
}

func kSMI(highs, lows, closes []float64) ([]float64, []float64) {
	n := len(closes)
	highest, lowest := nanSlice(n), nanSlice(n)
	for i := 13; i < n; i++ {
		hi, lo := -math.MaxFloat64, math.MaxFloat64
		for j := 0; j < 14; j++ {
			hi = math.Max(hi, highs[i-j])
			lo = math.Min(lo, lows[i-j])
		}
		highest[i], lowest[i] = hi, lo
	}
	raw := nanSlice(n)
	for i := range closes {
		if finite(highest[i]) && finite(lowest[i]) && highest[i] != lowest[i] {
			raw[i] = 200 * ((closes[i] - (highest[i]+lowest[i])/2) / (highest[i] - lowest[i]))
		}
	}
	line := kEMA(raw, 3)
	return line, kEMA(line, 3)
}

func clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
