package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"go-stock/backend/appdata"
	stockcli "go-stock/backend/cli"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go.uber.org/zap"
)

func main() {
	quietLogger()
	req, dbPath, err := parseArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	if dbPath == "" {
		dbPath = dbPathFromEnv()
	}
	if dbPath == "" {
		log.Fatal("resolve go-stock database path failed")
	}
	db.Init(dbPath)
	data.InitAnalyzeSentiment()

	runner, err := stockcli.NewRunner()
	if err != nil {
		log.Fatal(err)
	}
	out, err := runner.RunText(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
}

func quietLogger() {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("GO_STOCK_CLI_VERBOSE")), "true") {
		return
	}
	logger.Logger = zap.NewNop()
	logger.SugaredLogger = logger.Logger.Sugar()
}

func dbPathFromEnv() string {
	if path := strings.TrimSpace(os.Getenv("GO_STOCK_DB")); path != "" {
		return path
	}
	path, err := appdata.DefaultDBPath()
	if err != nil {
		return ""
	}
	return path
}

func parseArgs(argv []string) (stockcli.Request, string, error) {
	req := stockcli.Request{Args: map[string]any{}}
	var commandParts []string
	var dbPath string

	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch arg {
		case "--json":
			req.Format = "json"
		case "--format":
			i++
			if i >= len(argv) {
				return req, dbPath, fmt.Errorf("--format requires a value")
			}
			req.Format = argv[i]
		case "--db":
			i++
			if i >= len(argv) {
				return req, dbPath, fmt.Errorf("--db requires a value")
			}
			dbPath = argv[i]
		case "--args-json":
			i++
			if i >= len(argv) {
				return req, dbPath, fmt.Errorf("--args-json requires a JSON object")
			}
			if err := json.Unmarshal([]byte(argv[i]), &req.Args); err != nil {
				return req, dbPath, fmt.Errorf("parse --args-json: %w", err)
			}
		case "--arg":
			i++
			if i >= len(argv) {
				return req, dbPath, fmt.Errorf("--arg requires key=value")
			}
			if err := setKeyValue(req.Args, argv[i]); err != nil {
				return req, dbPath, err
			}
		case "--confirm":
			req.Confirm = true
		case "--confirm-token":
			i++
			if i >= len(argv) {
				return req, dbPath, fmt.Errorf("--confirm-token requires a value")
			}
			req.ConfirmToken = argv[i]
		default:
			if strings.HasPrefix(arg, "--") {
				key := strings.TrimPrefix(arg, "--")
				if strings.Contains(key, "=") {
					if err := setKeyValue(req.Args, key); err != nil {
						return req, dbPath, err
					}
					continue
				}
				if i+1 >= len(argv) || strings.HasPrefix(argv[i+1], "--") {
					req.Args[normalizeArgKey(key)] = true
					continue
				}
				i++
				req.Args[normalizeArgKey(key)] = parseValue(argv[i])
				continue
			}
			commandParts = append(commandParts, arg)
		}
	}

	if len(commandParts) == 0 {
		req.CommandPath = "help"
	} else {
		req.CommandPath = strings.Join(commandParts, " ")
	}
	return req, dbPath, nil
}

func setKeyValue(args map[string]any, raw string) error {
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return fmt.Errorf("argument %q must be key=value", raw)
	}
	args[normalizeArgKey(parts[0])] = parseValue(parts[1])
	return nil
}

func normalizeArgKey(key string) string {
	key = strings.TrimSpace(key)
	replacements := map[string]string{
		"stock-code":           "stockCode",
		"stock_name":           "stockName",
		"stock-name":           "stockName",
		"fund-code":            "fundCode",
		"group-id":             "groupId",
		"new-name":             "newName",
		"top-n":                "topN",
		"page-index":           "pageIndex",
		"page-size":            "pageSize",
		"cost-price":           "costPrice",
		"entry-price":          "entryPrice",
		"take-profit-price":    "takeProfitPrice",
		"stop-loss-price":      "stopLossPrice",
		"alarm-change-percent": "alarmChangePercent",
		"alarm-price":          "alarmPrice",
		"k-line-type":          "kLineType",
		"ma-periods":           "maPeriods",
	}
	if v, ok := replacements[key]; ok {
		return v
	}
	return key
}

func parseValue(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if hasLeadingZeroDigits(value) {
		return value
	}
	if strings.EqualFold(value, "true") {
		return true
	}
	if strings.EqualFold(value, "false") {
		return false
	}
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return value
}

func hasLeadingZeroDigits(value string) bool {
	if len(value) <= 1 || value[0] != '0' {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
