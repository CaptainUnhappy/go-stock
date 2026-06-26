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
