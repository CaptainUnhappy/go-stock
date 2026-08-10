package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"go-stock/backend/data"
	"go-stock/backend/db"
)

func ensureFollowedStockPositionTestDB(t *testing.T) {
	t.Helper()
	ensureDataToolsTestDB(t)
	if err := db.Dao.AutoMigrate(&data.FollowedStock{}); err != nil {
		t.Fatalf("migrate followed_stock failed: %v", err)
	}
}

func cleanupFollowedStockPosition(t *testing.T, code string) {
	t.Helper()
	if err := db.Dao.Unscoped().Where("stock_code = ?", strings.ToLower(code)).Delete(&data.FollowedStock{}).Error; err != nil {
		t.Fatalf("cleanup followed stock failed: %v", err)
	}
}

func TestSetFollowedStockPositionPreviewDoesNotWrite(t *testing.T) {
	ensureFollowedStockPositionTestDB(t)
	cleanupFollowedStockPosition(t, "sz003026")

	tool := GetSetFollowedStockPositionTool()
	out, err := tool.InvokableRun(context.Background(), `{
		"stockCode":"003026.SZ",
		"stockName":"中晶科技",
		"costPrice":10.5,
		"volume":200,
		"takeProfitPrice":12,
		"stopLossPrice":9.8,
		"alarmChangePercent":5,
		"reason":"按成本价上下约 10% 和 7% 设置提醒"
	}`)
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if !strings.Contains(out, "未写入数据库") || !strings.Contains(out, "confirmToken") {
		t.Fatalf("preview output missing confirmation text: %s", out)
	}

	var count int64
	if err := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", "sz003026").Count(&count).Error; err != nil {
		t.Fatalf("count followed stock failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("preview wrote %d rows, want 0", count)
	}
}

func TestSetFollowedStockPositionRejectsWrongToken(t *testing.T) {
	ensureFollowedStockPositionTestDB(t)
	cleanupFollowedStockPosition(t, "sz003026")

	tool := GetSetFollowedStockPositionTool()
	_, err := tool.InvokableRun(context.Background(), `{
		"stockCode":"003026.SZ",
		"stockName":"中晶科技",
		"costPrice":10.5,
		"volume":200,
		"takeProfitPrice":12,
		"stopLossPrice":9.8,
		"alarmChangePercent":5,
		"confirm":true,
		"confirmToken":"wrong"
	}`)
	if err == nil {
		t.Fatal("expected confirmToken mismatch error")
	}

	var count int64
	if err := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", "sz003026").Count(&count).Error; err != nil {
		t.Fatalf("count followed stock failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("wrong token wrote %d rows, want 0", count)
	}
}

func TestSetFollowedStockPositionConfirmWrites(t *testing.T) {
	ensureFollowedStockPositionTestDB(t)
	cleanupFollowedStockPosition(t, "sz003026")

	input := followedStockPositionInput{
		StockCode:          normalizeFollowedStockPositionCode("003026.SZ"),
		StockName:          "中晶科技",
		CostPrice:          10.5,
		Volume:             200,
		EntryPrice:         10.5,
		TakeProfitPrice:    12,
		StopLossPrice:      9.8,
		AlarmChangePercent: 5,
		AlarmPrice:         11.5,
		Sort:               7,
		Reason:             "按成本价上下约 14% 和 7% 设置提醒",
	}
	token := followedStockPositionConfirmToken(input)
	args := fmt.Sprintf(`{
		"stockCode":"003026.SZ",
		"stockName":"中晶科技",
		"costPrice":10.5,
		"volume":200,
		"entryPrice":10.5,
		"takeProfitPrice":12,
		"stopLossPrice":9.8,
		"alarmChangePercent":5,
		"alarmPrice":11.5,
		"sort":7,
		"reason":"按成本价上下约 14%% 和 7%% 设置提醒",
		"confirm":true,
		"confirmToken":"%s"
	}`, token)

	tool := GetSetFollowedStockPositionTool()
	out, err := tool.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("confirm failed: %v", err)
	}
	if !strings.Contains(out, "已写入") {
		t.Fatalf("confirm output missing success text: %s", out)
	}

	var stock data.FollowedStock
	if err := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", "sz003026").First(&stock).Error; err != nil {
		t.Fatalf("read followed stock failed: %v", err)
	}
	if stock.Name != "中晶科技" ||
		stock.CostPrice != 10.5 ||
		stock.Volume != 200 ||
		stock.EntryPrice != 10.5 ||
		stock.TakeProfitPrice != 12 ||
		stock.StopLossPrice != 9.8 ||
		stock.AlarmChangePercent != 5 ||
		stock.AlarmPrice != 11.5 ||
		stock.Sort != 7 {
		t.Fatalf("unexpected followed stock: %#v", stock)
	}
}

func TestSetFollowedStockPositionDefaultsSortToWatchList(t *testing.T) {
	ensureFollowedStockPositionTestDB(t)
	cleanupFollowedStockPosition(t, "sh688525")

	input := followedStockPositionInput{
		StockCode:          normalizeFollowedStockPositionCode("688525.SH"),
		StockName:          "佰维存储",
		CostPrice:          100.5,
		Volume:             100,
		EntryPrice:         100.5,
		TakeProfitPrice:    110,
		StopLossPrice:      95,
		AlarmChangePercent: 5,
		AlarmPrice:         99,
		Reason:             "未传 sort 的关注标的默认放到观察尾部",
	}
	token := followedStockPositionConfirmToken(input)
	args := fmt.Sprintf(`{
		"stockCode":"688525.SH",
		"stockName":"佰维存储",
		"costPrice":100.5,
		"volume":100,
		"entryPrice":100.5,
		"takeProfitPrice":110,
		"stopLossPrice":95,
		"alarmChangePercent":5,
		"alarmPrice":99,
		"reason":"未传 sort 的关注标的默认放到观察尾部",
		"confirm":true,
		"confirmToken":"%s"
	}`, token)

	tool := GetSetFollowedStockPositionTool()
	if _, err := tool.InvokableRun(context.Background(), args); err != nil {
		t.Fatalf("confirm failed: %v", err)
	}

	var stock data.FollowedStock
	if err := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", "sh688525").First(&stock).Error; err != nil {
		t.Fatalf("read followed stock failed: %v", err)
	}
	if stock.Sort != data.FollowedStockDefaultSort {
		t.Fatalf("Sort = %d, want %d", stock.Sort, data.FollowedStockDefaultSort)
	}
}

func TestSetFollowedStockPositionRejectsTIIndex(t *testing.T) {
	tool := GetSetFollowedStockPositionTool()
	_, err := tool.InvokableRun(context.Background(), `{"stockCode":"883418.TI","costPrice":1,"volume":1}`)
	if err == nil || !strings.Contains(err.Error(), "不接受 .TI") {
		t.Fatalf("err = %v", err)
	}
}

func TestGetFollowedStocksReturnsMarkdownTable(t *testing.T) {
	ensureFollowedStockPositionTestDB(t)
	cleanupFollowedStockPosition(t, "sh603690")

	if err := db.Dao.Create(&data.FollowedStock{
		StockCode:          "sh603690",
		Name:               "至纯科技",
		CostPrice:          31.11,
		Volume:             100,
		EntryPrice:         31.11,
		TakeProfitPrice:    34.22,
		StopLossPrice:      29.88,
		AlarmChangePercent: 3.5,
		AlarmPrice:         32.2,
		Sort:               5,
	}).Error; err != nil {
		t.Fatalf("create followed stock failed: %v", err)
	}

	var tool *DataToolWrapper
	for _, candidate := range GetAllDataTools() {
		info, err := candidate.Info(context.Background())
		if err != nil {
			t.Fatalf("tool info failed: %v", err)
		}
		if info.Name == "GetFollowedStocks" {
			var ok bool
			tool, ok = candidate.(*DataToolWrapper)
			if !ok {
				t.Fatalf("GetFollowedStocks has type %T, want *DataToolWrapper", candidate)
			}
			break
		}
	}
	if tool == nil {
		t.Fatal("GetFollowedStocks tool not found")
	}

	out, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("GetFollowedStocks failed: %v", err)
	}
	if !strings.Contains(out, "至纯科技") ||
		!strings.Contains(out, "31.11") ||
		strings.Contains(out, "切片/数组元素必须是结构体") {
		t.Fatalf("unexpected GetFollowedStocks output: %s", out)
	}
}
