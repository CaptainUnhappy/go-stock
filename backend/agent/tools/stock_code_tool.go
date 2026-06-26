package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"go-stock/backend/data"
	"go-stock/backend/logger"
)

// @Author spark
// @Date 2025/8/4 18:25
// @Desc
//-----------------------------------------------------------------------------------

func GetQueryStockCodeInfoTool() tool.InvokableTool {
	return &QueryStockCodeInfo{}
}

type QueryStockCodeInfo struct {
}

var stockBasicFileCache struct {
	once  sync.Once
	items []data.StockBasic
}

func (q QueryStockCodeInfo) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "QueryStockCodeInfo",
		Desc: "查询股票/指数信息(股票/指数名称,股票/指数代码,股票/指数拼音,股票/指数拼音首字母,股票/指数交易所等",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"searchWord": {
				Type:     "string",
				Desc:     "股票搜索关键词",
				Required: true,
			},
		}),
	}, nil
}

func (q QueryStockCodeInfo) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	logger.SugaredLogger.Infof("QueryStockCodeInfo called with args: %s", argumentsInJSON)
	parms := map[string]any{}
	err := json.Unmarshal([]byte(argumentsInJSON), &parms)
	if err != nil {
		logger.SugaredLogger.Errorf("QueryStockCodeInfo unmarshal error: %v", err)
		return "", err
	}
	searchWord, ok := parms["searchWord"].(string)
	searchWord = strings.TrimSpace(searchWord)
	if !ok || searchWord == "" {
		logger.SugaredLogger.Errorf("QueryStockCodeInfo searchWord not found in args")
		return "未找到股票信息", nil
	}
	logger.SugaredLogger.Infof("QueryStockCodeInfo searching for: %s", searchWord)
	stockList := queryStockCodeInfo(searchWord)
	marshal, err := json.Marshal(stockList)
	if err != nil {
		logger.SugaredLogger.Errorf("QueryStockCodeInfo marshal error: %v", err)
		return "", err
	}
	logger.SugaredLogger.Infof("QueryStockCodeInfo result length: %d", len(marshal))
	return string(marshal), nil
}

func queryStockCodeInfo(searchWord string) []data.StockBasic {
	api := data.NewStockDataApi()
	seen := map[string]bool{}
	var result []data.StockBasic

	add := func(items []data.StockBasic) {
		for _, item := range items {
			key := strings.TrimSpace(item.TsCode)
			if key == "" {
				key = strings.TrimSpace(item.Symbol) + ":" + strings.TrimSpace(item.Name)
			}
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			result = append(result, item)
		}
	}

	for _, word := range stockSearchVariants(searchWord) {
		add(api.GetStockList(word))
	}
	if len(result) > 0 {
		return result
	}

	add(matchStockBasicFile(searchWord))
	if len(result) > 0 {
		return result
	}

	if code := realtimeStockCodeVariant(searchWord); code != "" {
		if rows, err := api.GetStockCodeRealTimeData(code); err == nil && rows != nil {
			for _, s := range *rows {
				if s.Code == "" && s.Name == "" {
					continue
				}
				add([]data.StockBasic{{
					TsCode:   realtimeCodeToTushare(s.Code),
					Symbol:   data.RemoveAllNonDigitChar(s.Code),
					Name:     s.Name,
					Fullname: s.Name,
					Market:   strings.ToUpper(data.RemoveAllDigitChar(realtimeCodeToTushare(s.Code))),
				}})
			}
		}
	}
	return result
}

func stockSearchVariants(searchWord string) []string {
	clean := strings.TrimSpace(searchWord)
	if clean == "" {
		return nil
	}
	added := map[string]bool{}
	var variants []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" || added[v] {
			return
		}
		added[v] = true
		variants = append(variants, v)
	}

	add(clean)
	upper := strings.ToUpper(clean)
	lower := strings.ToLower(clean)
	add(upper)
	add(lower)

	digits := data.RemoveAllNonDigitChar(clean)
	if len(digits) == 6 {
		add(digits)
		if strings.HasPrefix(digits, "6") {
			add(digits + ".SH")
			add("sh" + digits)
		} else if strings.HasPrefix(digits, "0") || strings.HasPrefix(digits, "3") {
			add(digits + ".SZ")
			add("sz" + digits)
		} else if strings.HasPrefix(digits, "4") || strings.HasPrefix(digits, "8") {
			add(digits + ".BJ")
			add("bj" + digits)
		}
	}
	if strings.Contains(upper, ".") {
		parts := strings.Split(upper, ".")
		if len(parts) == 2 {
			add(parts[0])
			add(strings.ToLower(parts[1]) + parts[0])
		}
	}
	return variants
}

func realtimeStockCodeVariant(searchWord string) string {
	digits := data.RemoveAllNonDigitChar(searchWord)
	if len(digits) != 6 {
		return ""
	}
	prefix := strings.ToLower(data.RemoveAllDigitChar(searchWord))
	if strings.Contains(prefix, "sh") || strings.HasPrefix(digits, "6") {
		return "sh" + digits
	}
	if strings.Contains(prefix, "bj") || strings.HasPrefix(digits, "4") || strings.HasPrefix(digits, "8") {
		return "bj" + digits
	}
	return "sz" + digits
}

func realtimeCodeToTushare(code string) string {
	digits := data.RemoveAllNonDigitChar(code)
	prefix := strings.ToUpper(data.RemoveAllDigitChar(code))
	if digits == "" {
		return code
	}
	switch {
	case strings.Contains(prefix, "SH"):
		return digits + ".SH"
	case strings.Contains(prefix, "SZ"):
		return digits + ".SZ"
	case strings.Contains(prefix, "BJ"):
		return digits + ".BJ"
	default:
		return code
	}
}

func matchStockBasicFile(searchWord string) []data.StockBasic {
	items := loadStockBasicFile()
	if len(items) == 0 {
		return nil
	}
	var matches []data.StockBasic
	for _, item := range items {
		if stockBasicMatches(item, searchWord) {
			matches = append(matches, item)
			if len(matches) >= 20 {
				break
			}
		}
	}
	return matches
}

func stockBasicMatches(item data.StockBasic, searchWord string) bool {
	word := strings.ToLower(strings.TrimSpace(searchWord))
	digits := data.RemoveAllNonDigitChar(searchWord)
	fields := []string{item.TsCode, item.Symbol, item.Name, item.Fullname, item.Cnspell, item.Market, item.Exchange}
	for _, field := range fields {
		v := strings.ToLower(strings.TrimSpace(field))
		if v == "" {
			continue
		}
		if strings.Contains(v, word) || (digits != "" && strings.Contains(v, strings.ToLower(digits))) {
			return true
		}
	}
	return false
}

func loadStockBasicFile() []data.StockBasic {
	stockBasicFileCache.once.Do(func() {
		for _, path := range stockBasicFileCandidates() {
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var resp data.TushareStockBasicResponse
			if err := json.Unmarshal(raw, &resp); err != nil {
				continue
			}
			stockBasicFileCache.items = tushareStockBasics(resp)
			if len(stockBasicFileCache.items) > 0 {
				return
			}
		}
	})
	return stockBasicFileCache.items
}

func stockBasicFileCandidates() []string {
	_, sourceFile, _, ok := runtime.Caller(0)
	var candidates []string
	if ok {
		candidates = append(candidates, filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "data", "stock_basic.json")))
	}
	candidates = append(candidates,
		filepath.Clean("backend/data/stock_basic.json"),
		filepath.Clean("build/stock_basic.json"),
		filepath.Clean("stock_basic.json"),
	)
	return candidates
}

func tushareStockBasics(resp data.TushareStockBasicResponse) []data.StockBasic {
	fieldIndex := map[string]int{}
	for i, field := range resp.Data.Fields {
		fieldIndex[field] = i
	}
	get := func(item []any, field string) string {
		idx, ok := fieldIndex[field]
		if !ok || idx >= len(item) || item[idx] == nil {
			return ""
		}
		return fmt.Sprint(item[idx])
	}
	out := make([]data.StockBasic, 0, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		out = append(out, data.StockBasic{
			TsCode:     get(item, "ts_code"),
			Symbol:     get(item, "symbol"),
			Name:       get(item, "name"),
			Area:       get(item, "area"),
			Industry:   get(item, "industry"),
			Fullname:   get(item, "fullname"),
			Ename:      get(item, "enname"),
			Cnspell:    get(item, "cnspell"),
			Market:     get(item, "market"),
			Exchange:   get(item, "exchange"),
			CurrType:   get(item, "curr_type"),
			ListStatus: get(item, "list_status"),
			ListDate:   get(item, "list_date"),
			DelistDate: get(item, "delist_date"),
			IsHs:       get(item, "is_hs"),
			ActName:    get(item, "act_name"),
			ActEntType: get(item, "act_ent_type"),
		})
	}
	return out
}
