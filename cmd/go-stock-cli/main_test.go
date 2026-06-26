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
