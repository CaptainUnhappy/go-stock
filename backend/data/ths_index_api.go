package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	thsFinanceAPIBaseURL = "https://fuyao.aicubes.cn"
	thsPublicWebBaseURL  = "https://d.10jqka.com.cn"
)

var shanghaiLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*60*60)
	}
	return loc
}()

// IndexPriceBar is the stable JSON contract exposed by `index history`.
type IndexPriceBar struct {
	Date    string  `json:"date"`
	Open    float64 `json:"open"`
	High    float64 `json:"high"`
	Low     float64 `json:"low"`
	Close   float64 `json:"close"`
	Volume  float64 `json:"volume"`
	Amount  float64 `json:"amount"`
	Partial bool    `json:"partial"`
}

// IndexHistoryResult is shared by the CLI and the K-line fallback chain.
// Adjust intentionally remains nil because indexes have no adjustment semantics.
type IndexHistoryResult struct {
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	Interval string          `json:"interval"`
	Start    string          `json:"start"`
	End      string          `json:"end"`
	Adjust   *string         `json:"adjust"`
	Source   string          `json:"source"`
	AsOf     string          `json:"asOf"`
	Warnings []string        `json:"warnings"`
	Bars     []IndexPriceBar `json:"bars"`
}

type thsIndexClient struct {
	httpClient *resty.Client
	apiBaseURL string
	webBaseURL string
	apiKey     string
	now        func() time.Time
}

type thsSourceError struct {
	message      string
	fallbackable bool
}

func (e *thsSourceError) Error() string { return e.message }

func newTHSIndexClient() *thsIndexClient {
	config := GetSettingConfig()
	timeout := 30 * time.Second
	if config != nil && config.CrawlTimeOut > 0 {
		timeout = time.Duration(config.CrawlTimeOut) * time.Second
	}
	return &thsIndexClient{
		httpClient: CreateHTTPClientWithTimeout(timeout),
		apiBaseURL: thsFinanceAPIBaseURL,
		webBaseURL: thsPublicWebBaseURL,
		apiKey:     thsFinanceAPIKey(config),
		now:        time.Now,
	}
}

func thsFinanceAPIKey(config *SettingConfig) string {
	if value := strings.TrimSpace(os.Getenv("THS_FINANCE_API_KEY")); value != "" {
		return value
	}
	if config == nil || config.Settings == nil {
		return ""
	}
	return strings.TrimSpace(config.ThsFinanceApiKey)
}

// NormalizeIndexCode preserves the public suffix form used by index APIs.
func NormalizeIndexCode(code string) (string, error) {
	upper := strings.ToUpper(strings.TrimSpace(code))
	parts := strings.Split(upper, ".")
	if len(parts) != 2 || len(parts[0]) != 6 {
		return "", fmt.Errorf("指数代码格式无效 %q；请使用 000001.SH、399001.SZ、930599.CSI 或 883418.TI", code)
	}
	for _, r := range parts[0] {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("指数代码格式无效 %q", code)
		}
	}
	switch parts[1] {
	case "SH", "SZ", "CSI", "TI":
		return upper, nil
	default:
		return "", fmt.Errorf("暂不支持指数后缀 .%s；仅支持 .SH、.SZ、.CSI、.TI", parts[1])
	}
}

func ParseIndexHistoryRange(startText, endText string) (time.Time, time.Time, error) {
	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startText), shanghaiLocation)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("start 必须为 YYYY-MM-DD")
	}
	end, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endText), shanghaiLocation)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("end 必须为 YYYY-MM-DD")
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("end 不能早于 start")
	}
	if end.After(start.AddDate(10, 0, 0)) {
		return time.Time{}, time.Time{}, fmt.Errorf("指数历史查询跨度不能超过 10 年")
	}
	return start, end, nil
}

// FetchIndexHistory fetches an inclusive daily index range.
func FetchIndexHistory(ctx context.Context, code, startText, endText string) (*IndexHistoryResult, error) {
	return newTHSIndexClient().fetchIndexHistory(ctx, code, startText, endText)
}

func (c *thsIndexClient) fetchIndexHistory(ctx context.Context, code, startText, endText string) (*IndexHistoryResult, error) {
	normalized, err := NormalizeIndexCode(code)
	if err != nil {
		return nil, err
	}
	start, end, err := ParseIndexHistoryRange(startText, endText)
	if err != nil {
		return nil, err
	}
	if IsTHSIndexCode(normalized) {
		return c.fetchTHSIndexHistory(ctx, normalized, start, end)
	}
	return c.fetchExistingIndexHistory(normalized, start, end)
}

func (c *thsIndexClient) fetchExistingIndexHistory(code string, start, end time.Time) (*IndexHistoryResult, error) {
	calendarDays := int(end.Sub(start).Hours()/24) + 1
	limit := calendarDays*5/7 + 40
	if limit < 60 {
		limit = 60
	}
	// Date-range queries must honor the requested historical end date. The regular
	// fallback chain starts with MAC, whose limit API always returns the latest bars
	// and ignores end, so use EastMoney's before-date endpoint directly here.
	fetched := fetchFromEastMoney(code, "", "101", limit, end.Format("20060102"), "none")
	if fetched != nil {
		fetched.Source = "eastmoney"
	}
	if fetched == nil || fetched.Data == nil || len(*fetched.Data) == 0 {
		message := "未获取到指数历史 K 线数据"
		if fetched != nil && strings.TrimSpace(fetched.Error) != "" {
			message = fetched.Error
		}
		return nil, errors.New(message)
	}
	warnings := append([]string{}, fetched.Warnings...)
	bars := make([]IndexPriceBar, 0, len(*fetched.Data))
	for i, item := range *fetched.Data {
		bar, err := kLineDataToIndexBar(item, c.now())
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("跳过第 %d 行：%v", i+1, err))
			continue
		}
		date, _ := time.ParseInLocation("2006-01-02", bar.Date, shanghaiLocation)
		if date.Before(start) || date.After(end) {
			continue
		}
		bars = append(bars, bar)
	}
	bars = normalizeIndexBars(bars)
	if len(bars) == 0 {
		return nil, errors.New("指数数据源返回了响应，但指定日期范围内没有有效 K 线")
	}
	return &IndexHistoryResult{
		Code: code, Interval: "1d", Start: start.Format("2006-01-02"), End: end.Format("2006-01-02"),
		Source: fetched.Source, AsOf: c.now().In(shanghaiLocation).Format(time.RFC3339), Warnings: warnings, Bars: bars,
	}, nil
}

func (c *thsIndexClient) fetchTHSIndexHistory(ctx context.Context, code string, start, end time.Time) (*IndexHistoryResult, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		result, err := c.fetchPublicWeb(ctx, code, start, end)
		if result != nil {
			result.Warnings = append([]string{"未配置同花顺金融数据 API Key，已使用未文档化网页兼容源。"}, result.Warnings...)
		}
		return result, err
	}
	result, err := c.fetchFinanceREST(ctx, code, start, end)
	if err == nil {
		return result, nil
	}
	var sourceErr *thsSourceError
	if !errors.As(err, &sourceErr) || !sourceErr.fallbackable {
		return nil, err
	}
	webResult, webErr := c.fetchPublicWeb(ctx, code, start, end)
	if webErr != nil {
		return nil, fmt.Errorf("同花顺金融数据 REST 失败（%v），网页兼容源也失败：%w", err, webErr)
	}
	webResult.Warnings = append([]string{"同花顺金融数据 REST 暂不可用，已降级到未文档化网页兼容源：" + err.Error()}, webResult.Warnings...)
	return webResult, nil
}

type thsFinanceEnvelope struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Data      struct {
		Timestamp int64           `json:"timestamp"`
		Items     []thsRESTBarRaw `json:"item"`
	} `json:"data"`
}

type thsRESTBarRaw struct {
	DateMS int64           `json:"date_ms"`
	Open   json.RawMessage `json:"open_price"`
	High   json.RawMessage `json:"high_price"`
	Low    json.RawMessage `json:"low_price"`
	Close  json.RawMessage `json:"close_price"`
	Volume json.RawMessage `json:"volume"`
	Amount json.RawMessage `json:"turnover"`
}

func (c *thsIndexClient) fetchFinanceREST(ctx context.Context, code string, start, end time.Time) (*IndexHistoryResult, error) {
	response, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("X-api-key", c.apiKey).
		SetQueryParams(map[string]string{
			"thscode":  code,
			"interval": "1d",
			"start":    strconv.FormatInt(start.UnixMilli(), 10),
			"end":      strconv.FormatInt(end.UnixMilli(), 10),
		}).Get(strings.TrimRight(c.apiBaseURL, "/") + "/api/a-share-index/prices/historical")
	if err != nil {
		return nil, &thsSourceError{message: "REST 网络或超时错误：" + err.Error(), fallbackable: true}
	}
	status := response.StatusCode()
	if status == 401 || status == 403 {
		return nil, &thsSourceError{message: fmt.Sprintf("同花顺金融数据鉴权或权限失败（HTTP %d）", status)}
	}
	if status == 429 || status >= 500 {
		return nil, &thsSourceError{message: fmt.Sprintf("REST 暂时不可用（HTTP %d）", status), fallbackable: true}
	}
	if status < 200 || status >= 300 {
		return nil, &thsSourceError{message: fmt.Sprintf("REST 请求失败（HTTP %d）", status)}
	}
	var envelope thsFinanceEnvelope
	if err := json.Unmarshal(response.Body(), &envelope); err != nil {
		return nil, &thsSourceError{message: "REST 响应 JSON 损坏", fallbackable: true}
	}
	if envelope.Code == 2001 || envelope.Code == 2003 {
		return nil, &thsSourceError{message: fmt.Sprintf("同花顺金融数据鉴权或权限失败（业务码 %d）：%s", envelope.Code, strings.TrimSpace(envelope.Message))}
	}
	if envelope.Code != 0 {
		return nil, &thsSourceError{message: fmt.Sprintf("同花顺金融数据请求失败（业务码 %d）：%s", envelope.Code, strings.TrimSpace(envelope.Message))}
	}
	warnings := []string{}
	bars := make([]IndexPriceBar, 0, len(envelope.Data.Items))
	for i, item := range envelope.Data.Items {
		bar, err := restBarToIndexBar(item, c.now())
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("REST 第 %d 行无效，已跳过：%v", i+1, err))
			continue
		}
		date, _ := time.ParseInLocation("2006-01-02", bar.Date, shanghaiLocation)
		if !date.Before(start) && !date.After(end) {
			bars = append(bars, bar)
		}
	}
	bars = normalizeIndexBars(bars)
	if len(bars) == 0 {
		return nil, &thsSourceError{message: "REST 返回空数据或没有有效 K 线", fallbackable: true}
	}
	asOf := c.now().In(shanghaiLocation)
	if envelope.Data.Timestamp > 0 {
		asOf = time.UnixMilli(envelope.Data.Timestamp).In(shanghaiLocation)
	}
	return &IndexHistoryResult{
		Code: code, Interval: "1d", Start: start.Format("2006-01-02"), End: end.Format("2006-01-02"),
		Source: "ths-finance-api", AsOf: asOf.Format(time.RFC3339), Warnings: warnings, Bars: bars,
	}, nil
}

func restBarToIndexBar(item thsRESTBarRaw, now time.Time) (IndexPriceBar, error) {
	if item.DateMS <= 0 {
		return IndexPriceBar{}, errors.New("date_ms 缺失")
	}
	values := make([]float64, 6)
	raw := []json.RawMessage{item.Open, item.High, item.Low, item.Close, item.Volume, item.Amount}
	for i := range raw {
		value, err := rawJSONFloat(raw[i])
		if err != nil {
			return IndexPriceBar{}, err
		}
		values[i] = value
	}
	date := time.UnixMilli(item.DateMS).In(shanghaiLocation).Format("2006-01-02")
	return IndexPriceBar{
		Date: date, Open: values[0], High: values[1], Low: values[2], Close: values[3], Volume: values[4], Amount: values[5],
		Partial: isPartialIndexBar(date, now),
	}, nil
}

func rawJSONFloat(raw json.RawMessage) (float64, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" || text == `""` || text == `"-"` {
		return 0, errors.New("数值字段缺失")
	}
	text = strings.Trim(text, `"`)
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("非法数值 %q", text)
	}
	return value, nil
}

type thsWebEnvelope struct {
	Data string `json:"data"`
}

func (c *thsIndexClient) fetchPublicWeb(ctx context.Context, code string, start, end time.Time) (*IndexHistoryResult, error) {
	digits := strings.TrimSuffix(code, ".TI")
	warnings := []string{}
	bars := []IndexPriceBar{}
	for year := start.Year(); year <= end.Year(); year++ {
		url := fmt.Sprintf("%s/v6/line/bk_%s/01/%d.js", strings.TrimRight(c.webBaseURL, "/"), digits, year)
		response, err := c.httpClient.R().SetContext(ctx).SetHeader("Referer", "https://q.10jqka.com.cn/").Get(url)
		if err != nil {
			return nil, fmt.Errorf("网页兼容源 %d 年请求失败：%w", year, err)
		}
		if response.StatusCode() < 200 || response.StatusCode() >= 300 {
			warnings = append(warnings, fmt.Sprintf("网页兼容源 %d 年返回 HTTP %d，已跳过", year, response.StatusCode()))
			continue
		}
		payload, err := unwrapTHSJSONP(response.Body())
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("网页兼容源 %d 年响应损坏，已跳过", year))
			continue
		}
		var envelope thsWebEnvelope
		if err := json.Unmarshal(payload, &envelope); err != nil {
			warnings = append(warnings, fmt.Sprintf("网页兼容源 %d 年 JSON 无法解析，已跳过", year))
			continue
		}
		for rowIndex, row := range strings.Split(envelope.Data, ";") {
			bar, err := parseTHSWebRow(row, c.now())
			if err != nil {
				if strings.TrimSpace(row) != "" {
					warnings = append(warnings, fmt.Sprintf("网页兼容源 %d 年第 %d 行无效，已跳过：%v", year, rowIndex+1, err))
				}
				continue
			}
			date, _ := time.ParseInLocation("2006-01-02", bar.Date, shanghaiLocation)
			if !date.Before(start) && !date.After(end) {
				bars = append(bars, bar)
			}
		}
	}
	bars = normalizeIndexBars(bars)
	if len(bars) == 0 {
		return nil, errors.New("网页兼容源没有返回指定日期范围内的有效 K 线")
	}
	return &IndexHistoryResult{
		Code: code, Interval: "1d", Start: start.Format("2006-01-02"), End: end.Format("2006-01-02"),
		Source: "ths-public-web", AsOf: c.now().In(shanghaiLocation).Format(time.RFC3339), Warnings: warnings, Bars: bars,
	}, nil
}

func unwrapTHSJSONP(body []byte) ([]byte, error) {
	text := strings.TrimSpace(string(body))
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start < 0 || end < start {
		return nil, errors.New("不是有效 JSONP")
	}
	payload := []byte(text[start : end+1])
	if !json.Valid(payload) {
		return nil, errors.New("JSONP 内层不是有效 JSON")
	}
	return payload, nil
}

func parseTHSWebRow(row string, now time.Time) (IndexPriceBar, error) {
	parts := strings.Split(strings.TrimSpace(row), ",")
	if len(parts) < 7 {
		return IndexPriceBar{}, errors.New("字段不足")
	}
	date, err := time.ParseInLocation("20060102", strings.TrimSpace(parts[0]), shanghaiLocation)
	if err != nil {
		return IndexPriceBar{}, errors.New("日期无效")
	}
	values := make([]float64, 6)
	for i := 0; i < 6; i++ {
		value, err := strconv.ParseFloat(strings.TrimSpace(parts[i+1]), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return IndexPriceBar{}, fmt.Errorf("第 %d 个数值无效", i+1)
		}
		values[i] = value
	}
	dateText := date.Format("2006-01-02")
	return IndexPriceBar{
		Date: dateText, Open: values[0], High: values[1], Low: values[2], Close: values[3], Volume: values[4], Amount: values[5],
		Partial: isPartialIndexBar(dateText, now),
	}, nil
}

func isPartialIndexBar(date string, now time.Time) bool {
	localNow := now.In(shanghaiLocation)
	return date == localNow.Format("2006-01-02") && (localNow.Hour() < 15)
}

func normalizeIndexBars(input []IndexPriceBar) []IndexPriceBar {
	byDate := make(map[string]IndexPriceBar, len(input))
	for _, bar := range input {
		byDate[bar.Date] = bar
	}
	out := make([]IndexPriceBar, 0, len(byDate))
	for _, bar := range byDate {
		out = append(out, bar)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

func kLineDataToIndexBar(item KLineData, now time.Time) (IndexPriceBar, error) {
	date, err := normalizeKLineDate(item.Day)
	if err != nil {
		return IndexPriceBar{}, err
	}
	values := make([]float64, 6)
	for i, raw := range []string{item.Open, item.High, item.Low, item.Close, item.Volume, item.Amount} {
		if strings.TrimSpace(raw) == "" && i >= 4 {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return IndexPriceBar{}, fmt.Errorf("数值字段无效")
		}
		values[i] = value
	}
	return IndexPriceBar{Date: date, Open: values[0], High: values[1], Low: values[2], Close: values[3], Volume: values[4], Amount: values[5], Partial: isPartialIndexBar(date, now)}, nil
}

func normalizeKLineDate(value string) (string, error) {
	text := strings.TrimSpace(value)
	for _, layout := range []string{"2006-01-02", "20060102", "2006/01/02"} {
		if date, err := time.ParseInLocation(layout, text, shanghaiLocation); err == nil {
			return date.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("日期 %q 无效", value)
}

func indexBarsToKLineData(bars []IndexPriceBar) []KLineData {
	result := make([]KLineData, 0, len(bars))
	var previousClose float64
	for _, bar := range bars {
		change := 0.0
		changePercent := 0.0
		if previousClose != 0 {
			change = bar.Close - previousClose
			changePercent = change / previousClose * 100
		}
		amplitude := 0.0
		if previousClose != 0 {
			amplitude = (bar.High - bar.Low) / previousClose * 100
		}
		result = append(result, KLineData{
			Day: bar.Date, Open: formatIndexNumber(bar.Open), High: formatIndexNumber(bar.High), Low: formatIndexNumber(bar.Low), Close: formatIndexNumber(bar.Close),
			Volume: formatIndexNumber(bar.Volume), Amount: formatIndexNumber(bar.Amount), ChangeValue: formatIndexNumber(change),
			ChangePercent: formatIndexNumber(changePercent), Amplitude: formatIndexNumber(amplitude), MA: map[string]string{},
		})
		previousClose = bar.Close
	}
	return result
}

func formatIndexNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func fetchTHSIndexKLine(stockCode string, klt string, limit int, endText string) *KLineSourceResult {
	empty := []KLineData{}
	if NormalizeKLineType(klt) != "101" {
		return &KLineSourceResult{Data: &empty, Error: ".TI 指数当前仅支持日线（1d）"}
	}
	if limit <= 0 {
		limit = 120
	}
	end := time.Now().In(shanghaiLocation)
	if strings.TrimSpace(endText) != "" {
		parsed, err := time.ParseInLocation("20060102", strings.TrimSpace(endText), shanghaiLocation)
		if err != nil {
			return &KLineSourceResult{Data: &empty, Error: "K 线结束日期格式无效"}
		}
		end = parsed
	}
	calendarDays := int(math.Ceil(float64(limit)*7.0/5.0)) + 45
	start := end.AddDate(0, 0, -calendarDays)
	result, err := newTHSIndexClient().fetchIndexHistory(context.Background(), stockCode, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return &KLineSourceResult{Data: &empty, Error: err.Error()}
	}
	bars := result.Bars
	if len(bars) > limit {
		bars = bars[len(bars)-limit:]
	}
	data := indexBarsToKLineData(bars)
	return &KLineSourceResult{Data: &data, Source: result.Source, Warnings: result.Warnings}
}
