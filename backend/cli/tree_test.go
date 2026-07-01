package cli

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"

	"go-stock/backend/data"
	"go-stock/backend/db"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var testDBOnce sync.Once

func ensureTestDB(t *testing.T) {
	t.Helper()
	testDBOnce.Do(func() {
		if db.Dao == nil {
			db.Init("file:go_stock_cli_test?mode=memory&cache=shared")
		}
	})
}

func TestCommandTreeHasNoDuplicatePaths(t *testing.T) {
	seen := map[string]bool{}
	for _, path := range CommandPathsFromTree() {
		if seen[path] {
			t.Fatalf("duplicate command path %q", path)
		}
		seen[path] = true
	}
}

func TestCommandTreeExcludesRemovedGUIItems(t *testing.T) {
	help := RenderHelp()
	for _, forbidden := range []string{"名站优选", "MCP服务"} {
		if strings.Contains(help, forbidden) {
			t.Fatalf("help contains removed GUI item %q", forbidden)
		}
	}
}

func TestCommandTreeCoversConfirmedMarketItems(t *testing.T) {
	help := RenderHelp()
	for _, required := range []string{
		"主要股指",
		"涨跌家数比",
		"涨跌停家数比",
		"当日异动次数最多的概念",
		"查看热词",
		"按天涨跌/涨跌停分析",
		"历史异动分析",
		"异动排行",
		"利好/利空排行",
		"快讯列表",
		"富时中国三倍做多",
		"VIX恐慌指数",
		"基金排行",
	} {
		if !strings.Contains(help, required) {
			t.Fatalf("help missing %q", required)
		}
	}
}

func TestRunnerRegistersExecutableCommands(t *testing.T) {
	ensureTestDB(t)
	runner, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	paths := map[string]bool{}
	for _, cmd := range runner.Commands() {
		paths[cmd.Path] = true
	}
	for _, required := range []string{
		"market news",
		"market major-index",
		"market money-flow stock",
		"market money-flow bk latest",
		"market money-flow concept latest",
		"portfolio list",
		"portfolio position set",
		"portfolio group rename",
		"fund ranking",
		"kline show",
		"kline signals",
		"tool list",
		"tool info",
	} {
		if !paths[required] {
			t.Fatalf("runner missing executable command %q", required)
		}
	}
}

func TestRunnerRegistersAllAllowedRawMCPTools(t *testing.T) {
	ensureTestDB(t)
	runner, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	paths := map[string]bool{}
	for _, cmd := range runner.Commands() {
		paths[cmd.Path] = true
	}

	for name := range buildToolEntries() {
		path := rawToolCommandPath(name)
		if isBlockedRawTool(name) {
			if paths[path] {
				t.Fatalf("blocked raw MCP tool %q should not be directly registered at %q", name, path)
			}
			continue
		}
		if !paths[path] {
			t.Fatalf("raw MCP tool %q was not migrated to CLI path %q", name, path)
		}
	}
}

func TestRawToolInfoAndKebabAlias(t *testing.T) {
	ensureTestDB(t)
	runner, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	info, err := runner.Run(context.Background(), Request{
		CommandPath: "tool info",
		Args: map[string]any{
			"name": "get-stock-order-book",
		},
	})
	if err != nil {
		t.Fatalf("tool info failed: %v", err)
	}
	if !strings.Contains(info.Output, "GetStockOrderBook") {
		t.Fatalf("tool info output missing canonical tool name: %s", info.Output)
	}
	if !strings.Contains(info.Output, "JSON Schema") {
		t.Fatalf("tool info output missing schema: %s", info.Output)
	}

	cmd, ok := runner.resolveRawToolCommand("tool get-stock-order-book")
	if !ok {
		t.Fatalf("kebab raw tool command did not resolve")
	}
	if cmd.Path != "tool GetStockOrderBook" {
		t.Fatalf("resolved path = %q, want canonical raw tool path", cmd.Path)
	}
}

func TestStockCodesFromCLIArgsSplitsCommaAndArray(t *testing.T) {
	codes := stockCodesFromCLIArgs(map[string]any{
		"stockCode":  "sz002335, sz002506 sh603690",
		"stockCodes": []any{"sh603690"},
	})
	want := []string{"sz002335", "sz002506", "sh603690", "sh603690"}
	if len(codes) != len(want) {
		t.Fatalf("codes = %#v, want %#v", codes, want)
	}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("codes = %#v, want %#v", codes, want)
		}
	}
}

type fakeInvokableTool struct {
	calls []string
}

func (f *fakeInvokableTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "GetStockLatestFinance"}, nil
}

type staticInvokableTool struct {
	output string
	calls  []string
}

func (f *staticInvokableTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "Static"}, nil
}

func (f *staticInvokableTool) InvokableRun(_ context.Context, args string, _ ...einotool.Option) (string, error) {
	f.calls = append(f.calls, args)
	return f.output, nil
}

func (f *fakeInvokableTool) InvokableRun(_ context.Context, args string, _ ...einotool.Option) (string, error) {
	f.calls = append(f.calls, args)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return "", err
	}
	return parsed["stockCode"].(string), nil
}

func TestToolRunnerSplitsSingleStockArchiveToolCalls(t *testing.T) {
	fake := &fakeInvokableTool{}
	r := &toolRunner{
		tools: map[string]einotool.InvokableTool{
			"GetStockLatestFinance": fake,
		},
	}
	out, err := r.call("GetStockLatestFinance", nil)(context.Background(), map[string]any{
		"stockCode": "sz002335,sz002506,sh603690",
	})
	if err != nil {
		t.Fatalf("split call failed: %v", err)
	}
	for _, code := range []string{"sz002335", "sz002506", "sh603690"} {
		if !strings.Contains(out, "## "+code) {
			t.Fatalf("output missing section for %s: %s", code, out)
		}
	}
	if len(fake.calls) != 3 {
		t.Fatalf("fake tool calls = %d, want 3", len(fake.calls))
	}
}

func TestRawToolRequiresStockCodeBeforeInvoke(t *testing.T) {
	fake := &staticInvokableTool{output: "should not run"}
	r := &toolRunner{
		tools: map[string]einotool.InvokableTool{
			"GetStockInfo": fake,
		},
	}
	_, err := r.call("GetStockInfo", nil)(context.Background(), map[string]any{
		"stock-code-wrong": "sh600237",
	})
	if err == nil {
		t.Fatal("expected missing stockCode error")
	}
	if !strings.Contains(err.Error(), "requires stockCode") {
		t.Fatalf("error = %v, want stockCode guidance", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("tool was invoked %d times, want 0", len(fake.calls))
	}
}

func TestOrderBookFallsBackToStockInfo(t *testing.T) {
	orderBook := &staticInvokableTool{output: "未找到盘口数据"}
	stockInfo := &staticInvokableTool{output: "买一: 12.58\n卖一: 12.59"}
	r := &toolRunner{
		tools: map[string]einotool.InvokableTool{
			"GetStockOrderBook": orderBook,
			"GetStockInfo":      stockInfo,
		},
	}
	out, err := r.call("GetStockOrderBook", nil)(context.Background(), map[string]any{
		"stock-code": "sh600237",
	})
	if err != nil {
		t.Fatalf("order book call failed: %v", err)
	}
	if !strings.Contains(out, "盘口兜底") || !strings.Contains(out, "买一") {
		t.Fatalf("fallback output missing expected content: %s", out)
	}
	if len(orderBook.calls) != 1 || len(stockInfo.calls) != 1 {
		t.Fatalf("calls = orderBook:%d stockInfo:%d, want 1/1", len(orderBook.calls), len(stockInfo.calls))
	}
}

func TestSingleStockArchiveToolClassification(t *testing.T) {
	for _, name := range []string{"GetStockLatestFinance", "GetStockConceptInfo"} {
		if !isSingleStockArchiveTool(name) {
			t.Fatalf("%s should be classified as single-stock archive tool", name)
		}
	}
	if isSingleStockArchiveTool("GetEastMoneyKLineWithMA") {
		t.Fatal("GetEastMoneyKLineWithMA supports multi-code output and should not be split by CLI wrapper")
	}
}

func TestRawToolInfoShowsCLIStockCodeAlias(t *testing.T) {
	ensureTestDB(t)
	runner, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	info, err := runner.Run(context.Background(), Request{
		CommandPath: "tool info",
		Args: map[string]any{
			"name": "GetStockInfo",
		},
	})
	if err != nil {
		t.Fatalf("tool info failed: %v", err)
	}
	for _, want := range []string{"CLI 参数名", "--stock-code", "sz300308,sz300502"} {
		if !strings.Contains(info.Output, want) {
			t.Fatalf("tool info missing %q: %s", want, info.Output)
		}
	}
}

func TestKLineSummaryCalculatesMovingAverageAndVolume(t *testing.T) {
	list := make([]data.KLineData, 0, 25)
	for i := 1; i <= 25; i++ {
		list = append(list, data.KLineData{
			Day:    "2026-06-" + strconv.Itoa(i),
			Close:  strconv.Itoa(i),
			Volume: "100",
		})
	}
	list[len(list)-1].Volume = "150"
	ma5, ok := averageClose(list, 5)
	if !ok || ma5 != 23 {
		t.Fatalf("MA5 = %v/%v, want 23/true", ma5, ok)
	}
	ret5, ok := kLineReturnPercent(list, 5)
	if !ok || ret5 <= 0 {
		t.Fatalf("ret5 = %v/%v, want positive", ret5, ok)
	}
	vol, ok := kLineVolumeSummary(list)
	if !ok || !strings.Contains(vol, "放量") {
		t.Fatalf("volume summary = %q/%v, want 放量", vol, ok)
	}
}

func TestKLineSignalsCalculateSummary(t *testing.T) {
	list := make([]data.KLineData, 0, 220)
	for i := 1; i <= 220; i++ {
		closeValue := 10.0 + float64(i)*0.1
		list = append(list, data.KLineData{
			Day:    "2026-06-" + strconv.Itoa(i),
			Open:   strconv.FormatFloat(closeValue-0.05, 'f', 2, 64),
			Close:  strconv.FormatFloat(closeValue, 'f', 2, 64),
			High:   strconv.FormatFloat(closeValue+0.15, 'f', 2, 64),
			Low:    strconv.FormatFloat(closeValue-0.15, 'f', 2, 64),
			Volume: strconv.Itoa(1000 + i),
		})
	}
	bars := parseKLineBars(list)
	signals := evaluateKLineSignals(bars)
	if len(signals) < 30 {
		t.Fatalf("signals = %d, want at least 30", len(signals))
	}
	var bullish int
	for _, signal := range signals {
		if signal.Signal == "bullish" {
			bullish++
		}
	}
	if bullish == 0 {
		t.Fatalf("expected at least one bullish signal: %#v", signals)
	}
}

func TestKLineAdjustFlagDefaultsAndValidation(t *testing.T) {
	got, err := kLineAdjustFlag(map[string]any{}, "day")
	if err != nil {
		t.Fatalf("kLineAdjustFlag day failed: %v", err)
	}
	if got != "qfq" {
		t.Fatalf("day default adjust = %q, want qfq", got)
	}

	got, err = kLineAdjustFlag(map[string]any{"adjustFlag": "hfq"}, "5")
	if err != nil {
		t.Fatalf("kLineAdjustFlag minute failed: %v", err)
	}
	if got != "" {
		t.Fatalf("minute adjust = %q, want empty", got)
	}

	got, err = kLineAdjustFlag(map[string]any{"adjustFlag": "bad"}, "day")
	if err == nil {
		t.Fatalf("kLineAdjustFlag bad err = nil, got %q", got)
	}
}
