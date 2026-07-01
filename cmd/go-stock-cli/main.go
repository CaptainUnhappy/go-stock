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
	var seenOption bool
	var lastValueKey string

	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch arg {
		case "--json":
			req.Format = "json"
			lastValueKey = ""
		case "--format":
			i++
			if i >= len(argv) {
				return req, dbPath, fmt.Errorf("--format requires a value")
			}
			req.Format = argv[i]
			lastValueKey = ""
		case "--db":
			i++
			if i >= len(argv) {
				return req, dbPath, fmt.Errorf("--db requires a value")
			}
			dbPath = argv[i]
			lastValueKey = ""
		case "--args-json":
			i++
			if i >= len(argv) {
				return req, dbPath, fmt.Errorf("--args-json requires a JSON object")
			}
			if err := json.Unmarshal([]byte(argv[i]), &req.Args); err != nil {
				return req, dbPath, fmt.Errorf("parse --args-json: %w", err)
			}
			seenOption = true
			lastValueKey = ""
		case "--arg":
			i++
			if i >= len(argv) {
				return req, dbPath, fmt.Errorf("--arg requires key=value")
			}
			if err := setKeyValue(req.Args, argv[i]); err != nil {
				return req, dbPath, err
			}
			seenOption = true
			lastValueKey = ""
		case "--confirm":
			req.Confirm = true
			seenOption = true
			lastValueKey = ""
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "--") {
				if v, ok := parseBoolLiteral(argv[i+1]); ok {
					req.Confirm = v
					i++
				}
			}
		case "--confirm-token":
			i++
			if i >= len(argv) {
				return req, dbPath, fmt.Errorf("--confirm-token requires a value")
			}
			req.ConfirmToken = argv[i]
			seenOption = true
			lastValueKey = ""
		default:
			if strings.HasPrefix(arg, "--") {
				key := strings.TrimPrefix(arg, "--")
				if strings.Contains(key, "=") {
					if err := setKeyValue(req.Args, key); err != nil {
						return req, dbPath, err
					}
					seenOption = true
					lastValueKey = ""
					continue
				}
				normalizedKey := normalizeArgKey(key)
				if i+1 >= len(argv) || strings.HasPrefix(argv[i+1], "--") {
					req.Args[normalizedKey] = true
					seenOption = true
					lastValueKey = ""
					continue
				}
				i++
				req.Args[normalizedKey] = parseValue(argv[i])
				seenOption = true
				lastValueKey = normalizedKey
				continue
			}
			if seenOption {
				if appendBareArgValue(req.Args, lastValueKey, arg) {
					continue
				}
				return req, dbPath, fmt.Errorf("unexpected positional argument %q after options; quote comma-separated values or use --args-json", arg)
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
	keyLower := strings.ToLower(key)
	replacements := map[string]string{
		"stock-code":           "stockCode",
		"stock_code":           "stockCode",
		"stockcode":            "stockCode",
		"stock-codes":          "stockCodes",
		"stock_codes":          "stockCodes",
		"stockcodes":           "stockCodes",
		"stock_name":           "stockName",
		"stock-name":           "stockName",
		"stockname":            "stockName",
		"fund-code":            "fundCode",
		"fund_code":            "fundCode",
		"fundcode":             "fundCode",
		"group-id":             "groupId",
		"group_id":             "groupId",
		"groupid":              "groupId",
		"new-name":             "newName",
		"new_name":             "newName",
		"newname":              "newName",
		"top-n":                "topN",
		"top_n":                "topN",
		"page-index":           "pageIndex",
		"page_index":           "pageIndex",
		"page-size":            "pageSize",
		"page_size":            "pageSize",
		"cost-price":           "costPrice",
		"cost_price":           "costPrice",
		"entry-price":          "entryPrice",
		"entry_price":          "entryPrice",
		"take-profit-price":    "takeProfitPrice",
		"take_profit_price":    "takeProfitPrice",
		"stop-loss-price":      "stopLossPrice",
		"stop_loss_price":      "stopLossPrice",
		"alarm-change-percent": "alarmChangePercent",
		"alarm_change_percent": "alarmChangePercent",
		"alarm-price":          "alarmPrice",
		"alarm_price":          "alarmPrice",
		"k-line-type":          "kLineType",
		"k_line_type":          "kLineType",
		"ma-periods":           "maPeriods",
		"ma_periods":           "maPeriods",
		"adjust":               "adjustFlag",
		"adjust-flag":          "adjustFlag",
		"adjust_flag":          "adjustFlag",
		"adjustflag":           "adjustFlag",
	}
	if v, ok := replacements[key]; ok {
		return v
	}
	if v, ok := replacements[keyLower]; ok {
		return v
	}
	return key
}

func parseBoolLiteral(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

func appendBareArgValue(args map[string]any, key string, value string) bool {
	if key != "stockCode" && key != "stockCodes" {
		return false
	}
	value = cleanBareStockCodeToken(value)
	if !looksLikeSecurityCode(value) {
		return false
	}
	current := cleanBareStockCodeToken(fmt.Sprint(args[key]))
	if current == "" {
		args[key] = value
	} else {
		args[key] = current + "," + value
	}
	return true
}

func cleanBareStockCodeToken(value string) string {
	return strings.Trim(strings.TrimSpace(value), ",，;；")
}

func looksLikeSecurityCode(value string) bool {
	if value == "" || strings.HasPrefix(value, "-") {
		return false
	}
	hasDigit := false
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '.' || r == '_':
		default:
			return false
		}
	}
	return hasDigit
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
