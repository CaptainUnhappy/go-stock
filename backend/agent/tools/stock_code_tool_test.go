package tools

import "testing"

func TestQueryStockCodeInfoFallsBackToBundledStockBasic(t *testing.T) {
	ensureDataToolsTestDB(t)

	for _, keyword := range []string{"中晶科技", "003026"} {
		matches := queryStockCodeInfo(keyword)
		found := false
		for _, item := range matches {
			if item.TsCode == "003026.SZ" && item.Name == "中晶科技" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("queryStockCodeInfo(%q) did not find 003026.SZ 中晶科技: %#v", keyword, matches)
		}
	}
}
