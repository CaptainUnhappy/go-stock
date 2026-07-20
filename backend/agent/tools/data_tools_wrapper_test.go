package tools

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/eino/schema"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/models"
)

var dataToolsTestDBOnce sync.Once

func ensureDataToolsTestDB(t *testing.T) {
	t.Helper()
	dataToolsTestDBOnce.Do(func() {
		if db.Dao == nil {
			db.Init("file:data_tools_test?mode=memory&cache=shared")
		}
	})
}

func TestGetAllDataTools(t *testing.T) {
	ensureDataToolsTestDB(t)

	tools := GetAllDataTools()
	t.Logf("Total tools count: %d", len(tools))

	toolNames := make(map[string]int)
	for i, tool := range tools {
		info, err := tool.Info(nil)
		if err != nil {
			t.Errorf("Tool %d: failed to get info: %v", i, err)
			continue
		}
		t.Logf("Tool %d: %s - %s", i+1, info.Name, info.Desc)

		if count, exists := toolNames[info.Name]; exists {
			t.Errorf("Duplicate tool name found: %s (count: %d)", info.Name, count+1)
		}
		toolNames[info.Name]++
	}

	t.Log("\n=== Tool List ===")
	for name, count := range toolNames {
		if count > 1 {
			t.Errorf("Duplicate tool: %s (count: %d)", name, count)
		}
	}
}

func TestMoneyFlowToolsExposeFrontendRankAndBKFundFlow(t *testing.T) {
	ensureDataToolsTestDB(t)

	tools := GetAllDataTools()
	infos := make(map[string]*schema.ToolInfo, len(tools))
	for _, source := range tools {
		info, err := source.Info(nil)
		if err != nil {
			t.Fatalf("read tool info: %v", err)
		}
		infos[info.Name] = info
	}

	for _, name := range []string{
		"GetMoneyRankSina",
		"GetAllBKCodes",
		"GetBKFundFlowTopList",
		"GetBKFundFlowTopListByDate",
		"GetBKFundFlowList",
		"GetBKFundFlowListByDate",
		"GetIndustryRank",
	} {
		if _, ok := infos[name]; !ok {
			t.Fatalf("missing tool %s", name)
		}
	}

	schemaData, err := infos["GetMoneyRankSina"].ParamsOneOf.ToJSONSchema()
	if err != nil {
		t.Fatalf("convert GetMoneyRankSina schema: %v", err)
	}
	raw, err := json.Marshal(schemaData)
	if err != nil {
		t.Fatalf("marshal GetMoneyRankSina schema: %v", err)
	}
	var payload struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal GetMoneyRankSina schema: %v", err)
	}
	if got := payload.Properties["sort"].Enum; len(got) != 9 {
		t.Fatalf("sort enum has %d values, want 9: %#v", len(got), got)
	}
}

func TestSearchToolsExposeWithDefaultEastMoneyIdentifier(t *testing.T) {
	ensureDataToolsTestDB(t)

	tools := GetAllDataTools()
	infos := make(map[string]*schema.ToolInfo, len(tools))
	for _, source := range tools {
		info, err := source.Info(nil)
		if err != nil {
			t.Fatalf("read tool info: %v", err)
		}
		infos[info.Name] = info
	}

	for _, name := range []string{"SearchStockByIndicators", "SearchBk", "SearchETF"} {
		if _, ok := infos[name]; !ok {
			t.Fatalf("missing EastMoney search tool %s", name)
		}
	}
}

func TestRealtimeAndClockToolsExposeCleanInterfaces(t *testing.T) {
	ensureDataToolsTestDB(t)

	tools := GetAllDataTools()
	infos := make(map[string]*schema.ToolInfo, len(tools))
	var currentTimeTool *DataToolWrapper
	for _, source := range tools {
		info, err := source.Info(nil)
		if err != nil {
			t.Fatalf("read tool info: %v", err)
		}
		infos[info.Name] = info
		if wrapper, ok := source.(*DataToolWrapper); ok && info.Name == "GetCurrentTime" {
			currentTimeTool = wrapper
		}
	}

	for _, name := range []string{"GetStockOrderBook", "GetGlobalMarketStatus"} {
		if _, ok := infos[name]; !ok {
			t.Fatalf("missing realtime/clock tool %s", name)
		}
	}
	if currentTimeTool == nil {
		t.Fatal("missing GetCurrentTime wrapper")
	}
	out, err := currentTimeTool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("GetCurrentTime failed: %v", err)
	}
	if strings.Contains(out, "指数") || strings.Contains(out, "|") {
		t.Fatalf("GetCurrentTime should only return clock text, got: %s", out)
	}
}

func TestStockInfoChangeUsesPreviousClose(t *testing.T) {
	price, preClose, change, pChange := stockInfoChange(data.StockInfo{
		Price:    "12.00",
		PreClose: "10.00",
		PrePrice: 0,
	})
	if price != 12 || preClose != 10 || change != 2 || pChange != 20 {
		t.Fatalf("stockInfoChange = price %.2f preClose %.2f change %.2f pct %.2f, want 12/10/2/20", price, preClose, change, pChange)
	}
}

func TestIndustryRankToolsExposeFrontendSubTabs(t *testing.T) {
	ensureDataToolsTestDB(t)

	tools := GetAllDataTools()
	infos := make(map[string]*schema.ToolInfo, len(tools))
	for _, source := range tools {
		info, err := source.Info(nil)
		if err != nil {
			t.Fatalf("read tool info: %v", err)
		}
		infos[info.Name] = info
	}

	for _, name := range []string{"GetIndustryRank", "GetIndustryMoneyRank"} {
		if _, ok := infos[name]; !ok {
			t.Fatalf("missing tool %s", name)
		}
	}

	schemaData, err := infos["GetIndustryMoneyRank"].ParamsOneOf.ToJSONSchema()
	if err != nil {
		t.Fatalf("convert GetIndustryMoneyRank schema: %v", err)
	}
	raw, err := json.Marshal(schemaData)
	if err != nil {
		t.Fatalf("marshal GetIndustryMoneyRank schema: %v", err)
	}
	var payload struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal GetIndustryMoneyRank schema: %v", err)
	}
	if got := payload.Properties["fenlei"].Enum; len(got) != 4 {
		t.Fatalf("fenlei enum has %d values, want 4: %#v", len(got), got)
	}
	if got := payload.Properties["sort"].Enum; len(got) != 3 {
		t.Fatalf("sort enum has %d values, want 3: %#v", len(got), got)
	}
}

func TestFormatStockHistoryMoneyDataAddsReadableSummary(t *testing.T) {
	rows := []models.StockMoneyDataHis{
		{Date: "2026-06-20", F62: "-10000", F184: "-1.20", F2: "10.00", F3: "-1.00"},
		{Date: "2026-06-23", F62: "20000", F184: "2.00", F2: "10.20", F3: "2.00"},
		{Date: "2026-06-24", F62: "30000", F184: "3.00", F2: "10.30", F3: "0.98"},
		{Date: "2026-06-25", F62: "40000", F184: "4.00", F2: "10.40", F3: "0.97"},
	}
	out := formatStockHistoryMoneyData("sz300308", rows, 2)
	for _, want := range []string{
		"历史资金流向摘要",
		"近3日主力净额合计：9.00万",
		"主力连续净流入天数：3",
		"最近 2 条",
		"主力净占比",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestWithAgentMeta_AgentMetaFromCtx(t *testing.T) {
	// 空 context 取不到
	if _, ok := AgentMetaFromCtx(context.Background()); ok {
		t.Fatalf("AgentMetaFromCtx should return ok=false for empty context")
	}

	// 注入后能完整取出
	meta := AgentMeta{
		ModelName:    "glm-5.2",
		SystemPrompt: "你是股票分析大师",
		UserPrompt:   "分析 600519",
	}
	ctx := WithAgentMeta(context.Background(), meta)
	got, ok := AgentMetaFromCtx(ctx)
	if !ok {
		t.Fatalf("AgentMetaFromCtx should return ok=true after WithAgentMeta")
	}
	if got != meta {
		t.Fatalf("AgentMeta round-trip mismatch: got %+v, want %+v", got, meta)
	}
}

func TestInjectRecommendMeta_Single(t *testing.T) {
	// AI 自填 modelName，应被实际值覆盖；系统/用户提示词应被填充
	args := `{"modelName":"ai-fake-name","stockCode":"600519.SH","stockName":"贵州茅台","rating":"买入"}`
	meta := AgentMeta{
		ModelName:    "glm-5.2",
		SystemPrompt: "你是顶级股票投资大师",
		UserPrompt:   "请分析贵州茅台",
	}

	out := injectRecommendMeta("CreateAiRecommendStocks", args, meta)
	if out == "" {
		t.Fatalf("injectRecommendMeta returned empty for valid single args")
	}

	var rec models.AiRecommendStocks
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("unmarshal injected result failed: %v", err)
	}
	if rec.ModelName != meta.ModelName {
		t.Errorf("ModelName not overridden: got %q, want %q", rec.ModelName, meta.ModelName)
	}
	if rec.SystemPrompt != meta.SystemPrompt {
		t.Errorf("SystemPrompt not filled: got %q, want %q", rec.SystemPrompt, meta.SystemPrompt)
	}
	if rec.UserPrompt != meta.UserPrompt {
		t.Errorf("UserPrompt not filled: got %q, want %q", rec.UserPrompt, meta.UserPrompt)
	}
	// 原有字段应保留
	if rec.StockCode != "600519.SH" {
		t.Errorf("StockCode should be preserved: got %q", rec.StockCode)
	}
	if rec.Rating != "买入" {
		t.Errorf("Rating should be preserved: got %q", rec.Rating)
	}
}

func TestInjectRecommendMeta_Batch(t *testing.T) {
	args := `{"stocks":[{"modelName":"fake-1","stockCode":"600519.SH"},{"modelName":"fake-2","stockCode":"000001.SZ"}]}`
	meta := AgentMeta{
		ModelName:    "deepseek-chat",
		SystemPrompt: "系统提示",
		UserPrompt:   "用户提问",
	}

	out := injectRecommendMeta("BatchCreateAiRecommendStocks", args, meta)
	if out == "" {
		t.Fatalf("injectRecommendMeta returned empty for valid batch args")
	}

	// 校验外层仍是 {"stocks":[...]} 结构
	if !isValidJSON(out) {
		t.Fatalf("injected output is not valid JSON: %s", out)
	}

	var wrapper struct {
		Stocks []*models.AiRecommendStocks `json:"stocks"`
	}
	if err := json.Unmarshal([]byte(out), &wrapper); err != nil {
		t.Fatalf("unmarshal wrapper failed: %v", err)
	}
	if len(wrapper.Stocks) != 2 {
		t.Fatalf("expected 2 stocks, got %d", len(wrapper.Stocks))
	}
	for i, rec := range wrapper.Stocks {
		if rec.ModelName != meta.ModelName {
			t.Errorf("stock[%d] ModelName not overridden: got %q, want %q", i, rec.ModelName, meta.ModelName)
		}
		if rec.SystemPrompt != meta.SystemPrompt {
			t.Errorf("stock[%d] SystemPrompt not filled: got %q, want %q", i, rec.SystemPrompt, meta.SystemPrompt)
		}
		if rec.UserPrompt != meta.UserPrompt {
			t.Errorf("stock[%d] UserPrompt not filled: got %q, want %q", i, rec.UserPrompt, meta.UserPrompt)
		}
	}
	// 原有字段保留
	if wrapper.Stocks[0].StockCode != "600519.SH" {
		t.Errorf("stock[0] StockCode should be preserved: got %q", wrapper.Stocks[0].StockCode)
	}
}

func TestInjectRecommendMeta_InvalidJSON(t *testing.T) {
	// 非法 JSON 应返回空字符串（调用方保留原 args）
	out := injectRecommendMeta("CreateAiRecommendStocks", "{not valid json", AgentMeta{ModelName: "x"})
	if out != "" {
		t.Errorf("injectRecommendMeta should return empty for invalid JSON, got %q", out)
	}
}

func isValidJSON(s string) bool {
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}
