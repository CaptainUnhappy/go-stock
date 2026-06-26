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
