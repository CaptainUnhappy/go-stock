package cli

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

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
		"fund ranking",
		"kline show",
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
