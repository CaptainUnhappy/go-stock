package data

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

func testTHSIndexClient(server *httptest.Server, key string, now time.Time) *thsIndexClient {
	return &thsIndexClient{
		httpClient: resty.New().SetTimeout(2 * time.Second),
		apiBaseURL: server.URL,
		webBaseURL: server.URL,
		apiKey:     key,
		now:        func() time.Time { return now },
	}
}

func webJSONP(rows string) string {
	return `quotebridge_v6_line_bk_883418_01_2025({"data":"` + rows + `"})`
}

func TestTHSIndexRESTRequestAndMapping(t *testing.T) {
	barDate := time.Date(2025, 1, 2, 0, 0, 0, 0, shanghaiLocation)
	asOf := time.Date(2025, 1, 2, 13, 0, 0, 0, shanghaiLocation)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/a-share-index/prices/historical" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("X-api-key"); got != "secret-key" {
			t.Fatalf("X-api-key = %q", got)
		}
		query := r.URL.Query()
		if query.Get("thscode") != "883418.TI" || query.Get("interval") != "1d" {
			t.Fatalf("unexpected query: %v", query)
		}
		start, _ := time.ParseInLocation("2006-01-02", "2025-01-01", shanghaiLocation)
		end, _ := time.ParseInLocation("2006-01-02", "2025-01-03", shanghaiLocation)
		if query.Get("start") != strconv.FormatInt(start.UnixMilli(), 10) || query.Get("end") != strconv.FormatInt(end.UnixMilli(), 10) {
			t.Fatalf("unexpected time query: %v", query)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success","data":{"timestamp":%d,"adjust":null,"item":[{"date_ms":%d,"open_price":2003.133,"high_price":2045.128,"low_price":1990.83,"close_price":2044.112,"volume":808261170,"turnover":4934662600}]}}`, asOf.UnixMilli(), barDate.UnixMilli())
	}))
	defer server.Close()

	client := testTHSIndexClient(server, "secret-key", asOf)
	result, err := client.fetchIndexHistory(context.Background(), "883418.ti", "2025-01-01", "2025-01-03")
	if err != nil {
		t.Fatal(err)
	}
	if result.Code != "883418.TI" || result.Source != "ths-finance-api" || len(result.Bars) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	bar := result.Bars[0]
	if bar.Date != "2025-01-02" || bar.Open != 2003.133 || bar.Amount != 4934662600 || !bar.Partial {
		t.Fatalf("unexpected bar: %+v", bar)
	}
	if result.AsOf != asOf.Format(time.RFC3339) {
		t.Fatalf("asOf = %s", result.AsOf)
	}
}

func TestTHSIndexPublicWebCrossYearFilterDeduplicateAndWarnings(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 30, 0, 0, shanghaiLocation)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/2025.js"):
			fmt.Fprint(w, webJSONP("20241231,1,2,0.5,1.5,10,20;20250102,2,3,1,2.5,100,200;bad,row"))
		case strings.HasSuffix(r.URL.Path, "/2026.js"):
			fmt.Fprint(w, webJSONP("20250102,9,10,8,9.5,900,2000;20260102,3,4,2,3.5,300,400;20260103,4,5,3,4.5,400,500"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	result, err := testTHSIndexClient(server, "", now).fetchIndexHistory(context.Background(), "883418.TI", "2025-01-01", "2026-01-02")
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != "ths-public-web" || len(result.Bars) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Bars[0].Date != "2025-01-02" || result.Bars[0].Open != 9 {
		t.Fatalf("duplicate date did not use the last row: %+v", result.Bars[0])
	}
	if result.Bars[1].Date != "2026-01-02" || !result.Bars[1].Partial {
		t.Fatalf("current bar partial mismatch: %+v", result.Bars[1])
	}
	if len(result.Warnings) < 2 || !strings.Contains(strings.Join(result.Warnings, " "), "未配置") || !strings.Contains(strings.Join(result.Warnings, " "), "字段不足") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestTHSIndexFallbackMatrix(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		wantFallback bool
		wantError    string
	}{
		{name: "invalid key", status: 200, body: `{"code":2001,"message":"invalid key"}`, wantError: "业务码 2001"},
		{name: "permission denied", status: 200, body: `{"code":2003,"message":"permission denied"}`, wantError: "业务码 2003"},
		{name: "http forbidden", status: 403, body: `{}`, wantError: "HTTP 403"},
		{name: "rate limited", status: 429, body: `{}`, wantFallback: true},
		{name: "server error", status: 503, body: `{}`, wantFallback: true},
		{name: "broken json", status: 200, body: `{`, wantFallback: true},
		{name: "empty data", status: 200, body: `{"code":0,"data":{"item":[]}}`, wantFallback: true},
		{name: "other business error", status: 200, body: `{"code":1003,"message":"bad range"}`, wantError: "业务码 1003"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var webCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/v6/line/") {
					webCalls.Add(1)
					fmt.Fprint(w, webJSONP("20250102,1,2,0.5,1.5,10,20"))
					return
				}
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()

			result, err := testTHSIndexClient(server, "configured-key", time.Date(2025, 1, 3, 16, 0, 0, 0, shanghaiLocation)).fetchIndexHistory(context.Background(), "883418.TI", "2025-01-01", "2025-01-03")
			if tt.wantFallback {
				if err != nil || result == nil || result.Source != "ths-public-web" || webCalls.Load() != 1 {
					t.Fatalf("fallback result=%+v err=%v webCalls=%d", result, err, webCalls.Load())
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("err = %v, want %q", err, tt.wantError)
			}
			if webCalls.Load() != 0 {
				t.Fatalf("auth/input failure must not fallback, webCalls=%d", webCalls.Load())
			}
		})
	}
}

func TestTHSIndexFallbackReportsWebFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/v6/line/") {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := testTHSIndexClient(server, "configured-key", time.Now()).fetchIndexHistory(context.Background(), "883418.TI", "2025-01-01", "2025-01-03")
	if err == nil || !strings.Contains(err.Error(), "网页兼容源也失败") {
		t.Fatalf("err = %v", err)
	}
}

func TestTHSIndexTimeoutAndNetworkErrorsFallback(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/v6/line/") {
				fmt.Fprint(w, webJSONP("20250102,1,2,0.5,1.5,10,20"))
				return
			}
			time.Sleep(100 * time.Millisecond)
			fmt.Fprint(w, `{"code":0,"data":{"item":[]}}`)
		}))
		defer server.Close()
		client := testTHSIndexClient(server, "key", time.Now())
		client.httpClient.SetTimeout(20 * time.Millisecond)
		result, err := client.fetchIndexHistory(context.Background(), "883418.TI", "2025-01-01", "2025-01-03")
		if err != nil || result.Source != "ths-public-web" {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})

	t.Run("network", func(t *testing.T) {
		apiServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		apiURL := apiServer.URL
		apiServer.Close()
		webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, webJSONP("20250102,1,2,0.5,1.5,10,20"))
		}))
		defer webServer.Close()
		client := testTHSIndexClient(webServer, "key", time.Now())
		client.apiBaseURL = apiURL
		result, err := client.fetchIndexHistory(context.Background(), "883418.TI", "2025-01-01", "2025-01-03")
		if err != nil || result.Source != "ths-public-web" {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
}

func TestNormalizeIndexCodeAndRange(t *testing.T) {
	for input, want := range map[string]string{
		"883418.ti":  "883418.TI",
		"000001.sh":  "000001.SH",
		"399001.SZ":  "399001.SZ",
		"930599.csi": "930599.CSI",
	} {
		got, err := NormalizeIndexCode(input)
		if err != nil || got != want {
			t.Fatalf("NormalizeIndexCode(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	if got := NormalizeFollowedStockCode("883418"); got != "bj883418" {
		t.Fatalf("bare 883418 changed semantics: %s", got)
	}
	if got := NormalizeFollowedStockCode("832000"); got != "bj832000" {
		t.Fatalf("bare 832000 changed semantics: %s", got)
	}
	if _, _, err := ParseIndexHistoryRange("2020-01-01", "2030-01-02"); err == nil {
		t.Fatal("expected 10-year range validation error")
	}
}

func TestTHSFinanceAPIKeyEnvPrecedence(t *testing.T) {
	t.Setenv("THS_FINANCE_API_KEY", "env-key")
	config := &SettingConfig{Settings: &Settings{ThsFinanceApiKey: "db-key"}}
	if got := thsFinanceAPIKey(config); got != "env-key" {
		t.Fatalf("key = %q", got)
	}
}
