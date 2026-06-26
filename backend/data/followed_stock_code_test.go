package data

import (
	"testing"

	"go-stock/backend/db"
)

func ensureFollowedStockDataTestDB(t *testing.T) {
	t.Helper()
	db.Init("file:followed_stock_data_test?mode=memory&cache=shared")
	if err := db.Dao.AutoMigrate(&FollowedStock{}); err != nil {
		t.Fatalf("migrate followed_stock failed: %v", err)
	}
}

func TestNormalizeFollowedStockCode(t *testing.T) {
	tests := map[string]string{
		"003026.SZ": "sz003026",
		"600237.SH": "sh600237",
		"832000.BJ": "bj832000",
		"00700.HK":  "hk00700",
		"003026":    "sz003026",
		"600237":    "sh600237",
		"832000":    "bj832000",
		"sh600237":  "sh600237",
		"GB_AAPL":   "gb_aapl",
	}

	for input, want := range tests {
		if got := NormalizeFollowedStockCode(input); got != want {
			t.Fatalf("NormalizeFollowedStockCode(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSetCostPriceAndVolumeNormalizesStockCode(t *testing.T) {
	ensureFollowedStockDataTestDB(t)
	db.Dao.Unscoped().Where("stock_code = ?", "sh600237").Delete(&FollowedStock{})

	if err := db.Dao.Create(&FollowedStock{StockCode: "sh600237", Name: "铜峰电子"}).Error; err != nil {
		t.Fatalf("create followed stock failed: %v", err)
	}

	got := NewStockDataApi().SetCostPriceAndVolume(12.56, 300, "600237.SH")
	if got != "设置成功" {
		t.Fatalf("SetCostPriceAndVolume returned %q, want 设置成功", got)
	}

	var stock FollowedStock
	if err := db.Dao.Where("stock_code = ?", "sh600237").First(&stock).Error; err != nil {
		t.Fatalf("read followed stock failed: %v", err)
	}
	if stock.CostPrice != 12.56 || stock.Volume != 300 {
		t.Fatalf("unexpected cost/volume: cost=%v volume=%v", stock.CostPrice, stock.Volume)
	}
}

func TestSetCostPriceAndVolumeReportsMissingStock(t *testing.T) {
	ensureFollowedStockDataTestDB(t)
	db.Dao.Unscoped().Where("stock_code = ?", "sh600238").Delete(&FollowedStock{})

	got := NewStockDataApi().SetCostPriceAndVolume(12.56, 300, "600238.SH")
	if got != "股票未关注" {
		t.Fatalf("SetCostPriceAndVolume returned %q, want 股票未关注", got)
	}
}

func TestGetFollowedStockByStockCodeNormalizesStockCode(t *testing.T) {
	ensureFollowedStockDataTestDB(t)
	db.Dao.Unscoped().Where("stock_code = ?", "sh600237").Delete(&FollowedStock{})

	if err := db.Dao.Create(&FollowedStock{StockCode: "sh600237", Name: "铜峰电子", CostPrice: 12.56, Volume: 300}).Error; err != nil {
		t.Fatalf("create followed stock failed: %v", err)
	}

	stock := NewStockDataApi().GetFollowedStockByStockCode("600237.SH")
	if stock.StockCode != "sh600237" || stock.CostPrice != 12.56 || stock.Volume != 300 {
		t.Fatalf("unexpected followed stock: %#v", stock)
	}
}
