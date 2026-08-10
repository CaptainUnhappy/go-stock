package main

import "testing"

func TestParseArgsNormalizesCommaStockCodeList(t *testing.T) {
	req, _, err := parseArgs([]string{"tool", "GetStockLatestFinance", "--stock-code", "sz002335,sz002506,sh603690"})
	if err != nil {
		t.Fatalf("parseArgs failed: %v", err)
	}
	if req.CommandPath != "tool GetStockLatestFinance" {
		t.Fatalf("CommandPath = %q, want raw tool command path", req.CommandPath)
	}
	if got := req.Args["stockCode"]; got != "sz002335,sz002506,sh603690" {
		t.Fatalf("stockCode arg = %#v, want comma string", got)
	}
}

func TestParseArgsNormalizesStockCodeAliases(t *testing.T) {
	for _, flag := range []string{"--stock-code", "--stock_code", "--stockcode", "--stockCode"} {
		req, _, err := parseArgs([]string{"tool", "GetStockInfo", flag, "sh600237"})
		if err != nil {
			t.Fatalf("parseArgs(%s) failed: %v", flag, err)
		}
		if got := req.Args["stockCode"]; got != "sh600237" {
			t.Fatalf("%s stockCode arg = %#v, want sh600237", flag, got)
		}
	}
}

func TestParseArgsPreservesBareStockCodeAsString(t *testing.T) {
	req, _, err := parseArgs([]string{"tool", "GetStockInfo", "--stock-code", "600237"})
	if err != nil {
		t.Fatalf("parseArgs failed: %v", err)
	}
	if got := req.Args["stockCode"]; got != "600237" {
		t.Fatalf("stockCode arg = %#v, want string 600237", got)
	}
}

func TestParseArgsNormalizesAdjustFlagAliases(t *testing.T) {
	for _, flag := range []string{"--adjust", "--adjust-flag", "--adjust_flag", "--adjustFlag"} {
		req, _, err := parseArgs([]string{"kline", "show", "--stock-code", "002335", flag, "hfq"})
		if err != nil {
			t.Fatalf("parseArgs(%s) failed: %v", flag, err)
		}
		if got := req.Args["adjustFlag"]; got != "hfq" {
			t.Fatalf("%s adjustFlag arg = %#v, want hfq", flag, got)
		}
	}
}

func TestParseArgsNormalizesCalendarAndNoticeAliases(t *testing.T) {
	req, _, err := parseArgs([]string{
		"market", "hot", "calendar",
		"--year-month", "2026-07",
		"--stock-list", "600237",
		"--start-date", "2026-07-01",
		"--end-date", "2026-07-03",
		"--data-type", "gdp",
	})
	if err != nil {
		t.Fatalf("parseArgs failed: %v", err)
	}
	for key, want := range map[string]any{
		"yearMonth":  "2026-07",
		"stock_list": "600237",
		"startDate":  "2026-07-01",
		"endDate":    "2026-07-03",
		"dataType":   "gdp",
	} {
		if got := req.Args[key]; got != want {
			t.Fatalf("%s arg = %#v, want %#v", key, got, want)
		}
	}
}

func TestParseArgsAppendsBareStockCodeAfterOption(t *testing.T) {
	req, _, err := parseArgs([]string{"tool", "GetStockLatestFinance", "--stockCode", "sh600237", "sz002335"})
	if err != nil {
		t.Fatalf("parseArgs failed: %v", err)
	}
	if req.CommandPath != "tool GetStockLatestFinance" {
		t.Fatalf("CommandPath = %q, want raw tool command path", req.CommandPath)
	}
	if got := req.Args["stockCode"]; got != "sh600237,sz002335" {
		t.Fatalf("stockCode arg = %#v, want comma-joined list", got)
	}
}

func TestParseArgsAppendsBareStockCodeWithSplitCommas(t *testing.T) {
	req, _, err := parseArgs([]string{"tool", "GetStockInfo", "--stockCode", "sz300308,", "sz300502"})
	if err != nil {
		t.Fatalf("parseArgs failed: %v", err)
	}
	if got := req.Args["stockCode"]; got != "sz300308,sz300502" {
		t.Fatalf("stockCode arg = %#v, want comma-joined list", got)
	}
}

func TestExtractStockCodesFromText(t *testing.T) {
	got := extractStockCodesFromText("代码 sz002335 / 002506.SZ / sh603690 / 600237 / sz002335")
	want := []string{"sz002335", "002506.SZ", "sh603690", "600237"}
	if len(got) != len(want) {
		t.Fatalf("extractStockCodesFromText len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("extractStockCodesFromText[%d] = %q, want %q; full=%#v", i, got[i], want[i], got)
		}
	}
}

func TestApplyStdinStockCodeTextReplacesDash(t *testing.T) {
	req, _, err := parseArgs([]string{"tool", "GetStockInfo", "--stock-code", "-"})
	if err != nil {
		t.Fatalf("parseArgs failed: %v", err)
	}
	if err := applyStdinStockCodeText(&req, "sz002335\nsz002506\n"); err != nil {
		t.Fatalf("applyStdinStockCodeText failed: %v", err)
	}
	if got := req.Args["stockCode"]; got != "sz002335,sz002506" {
		t.Fatalf("stockCode arg = %#v, want joined stdin codes", got)
	}
}

func TestApplyStdinStockCodeTextSetsBothStockKeysForStdinFlag(t *testing.T) {
	req, _, err := parseArgs([]string{"tool", "GetStockInfo", "--stdin"})
	if err != nil {
		t.Fatalf("parseArgs failed: %v", err)
	}
	if err := applyStdinStockCodeText(&req, "| 代码 |\n| sh600237 |\n| sz002335 |\n"); err != nil {
		t.Fatalf("applyStdinStockCodeText failed: %v", err)
	}
	for _, key := range []string{"stockCode", "stockCodes"} {
		if got := req.Args[key]; got != "sh600237,sz002335" {
			t.Fatalf("%s arg = %#v, want joined stdin codes", key, got)
		}
	}
	if _, ok := req.Args["stdin"]; ok {
		t.Fatalf("stdin flag should be removed after applying stdin stock codes")
	}
}

func TestParseArgsConfirmAcceptsExplicitBool(t *testing.T) {
	req, _, err := parseArgs([]string{"portfolio", "position", "set", "--confirm", "true", "--confirm-token", "abc"})
	if err != nil {
		t.Fatalf("parseArgs failed: %v", err)
	}
	if req.CommandPath != "portfolio position set" {
		t.Fatalf("CommandPath = %q, want portfolio position set", req.CommandPath)
	}
	if !req.Confirm {
		t.Fatal("Confirm = false, want true")
	}
	if req.ConfirmToken != "abc" {
		t.Fatalf("ConfirmToken = %q, want abc", req.ConfirmToken)
	}
}

func TestParseArgsAllowsGlobalOptionsBeforeCommand(t *testing.T) {
	req, _, err := parseArgs([]string{"--json", "market", "major-index"})
	if err != nil {
		t.Fatalf("parseArgs failed: %v", err)
	}
	if req.CommandPath != "market major-index" {
		t.Fatalf("CommandPath = %q, want market major-index", req.CommandPath)
	}
	if req.Format != "json" {
		t.Fatalf("Format = %q, want json", req.Format)
	}
}

func TestParseValuePreservesLeadingZeroDigits(t *testing.T) {
	got := parseValue("002335")
	if got != "002335" {
		t.Fatalf("parseValue(002335) = %#v, want string 002335", got)
	}
}

func TestParseValueStillParsesPlainNumbers(t *testing.T) {
	got := parseValue("2335")
	if got != int64(2335) {
		t.Fatalf("parseValue(2335) = %#v, want int64 2335", got)
	}
}

func TestParseArgsIndexHistory(t *testing.T) {
	req, _, err := parseArgs([]string{"--json", "index", "history", "--code", "883418.TI", "--start", "2025-01-01", "--end", "2026-08-10"})
	if err != nil {
		t.Fatal(err)
	}
	if req.CommandPath != "index history" || req.Format != "json" {
		t.Fatalf("request = %+v", req)
	}
	if req.Args["code"] != "883418.TI" || req.Args["start"] != "2025-01-01" || req.Args["end"] != "2026-08-10" {
		t.Fatalf("args = %#v", req.Args)
	}
}
