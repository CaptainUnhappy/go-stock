package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/tidwall/gjson"

	"go-stock/backend/data"
	"go-stock/backend/db"
)

type followedStockPositionInput struct {
	StockCode          string  `json:"stockCode"`
	StockName          string  `json:"stockName,omitempty"`
	CostPrice          float64 `json:"costPrice"`
	Volume             int64   `json:"volume"`
	EntryPrice         float64 `json:"entryPrice,omitempty"`
	TakeProfitPrice    float64 `json:"takeProfitPrice,omitempty"`
	StopLossPrice      float64 `json:"stopLossPrice,omitempty"`
	AlarmChangePercent float64 `json:"alarmChangePercent,omitempty"`
	AlarmPrice         float64 `json:"alarmPrice,omitempty"`
	Sort               int64   `json:"sort,omitempty"`
	Reason             string  `json:"reason,omitempty"`
	Confirm            bool    `json:"confirm,omitempty"`
	ConfirmToken       string  `json:"confirmToken,omitempty"`
}

// GetSetFollowedStockPositionTool returns the only MCP-facing write tool for
// followed-stock position/reminder settings. It is intentionally separated from
// the broader SetTradingPrice tool and requires a preview-confirm cycle.
func GetSetFollowedStockPositionTool() *DataToolWrapper {
	return NewDataToolWrapper(
		"SetFollowedStockPosition",
		"受控设置自选股持仓和提醒。Agent 可以根据持仓信息自行提出止损、止盈、涨跌提醒，但第一次调用只返回预览和确认令牌；必须由用户二次确认后，第二次用 confirm=true 和 confirmToken 才会写入。",
		map[string]*schema.ParameterInfo{
			"stockCode": {
				Type:     "string",
				Desc:     "股票代码，支持 003026、003026.SZ、sz003026、600519.SH 等格式。",
				Required: true,
			},
			"stockName": {
				Type:     "string",
				Desc:     "股票名称，可选；新增自选股且行情源无法识别名称时使用。",
				Required: false,
			},
			"costPrice": {
				Type:     "number",
				Desc:     "持仓成本价，必须大于 0。",
				Required: true,
			},
			"volume": {
				Type:     "integer",
				Desc:     "持仓数量，必须大于等于 0。",
				Required: true,
			},
			"entryPrice": {
				Type:     "number",
				Desc:     "开仓价，0 表示不设置或清空。",
				Required: false,
			},
			"takeProfitPrice": {
				Type:     "number",
				Desc:     "止盈价，0 表示不设置或清空。可由 Agent 根据持仓和风险偏好提出，但需要用户确认。",
				Required: false,
			},
			"stopLossPrice": {
				Type:     "number",
				Desc:     "止损价，0 表示不设置或清空。可由 Agent 根据持仓和风险偏好提出，但需要用户确认。",
				Required: false,
			},
			"alarmChangePercent": {
				Type:     "number",
				Desc:     "涨跌提醒百分比阈值，0 表示不设置或清空。例如 5 表示涨跌幅达到 5% 提醒。",
				Required: false,
			},
			"alarmPrice": {
				Type:     "number",
				Desc:     "股价提醒价，0 表示不设置或清空。",
				Required: false,
			},
			"sort": {
				Type:     "integer",
				Desc:     "自选股排序，可选；0 表示保持当前排序。",
				Required: false,
			},
			"reason": {
				Type:     "string",
				Desc:     "Agent 给出止盈、止损、涨跌提醒建议的简短理由，便于用户二次确认。",
				Required: false,
			},
			"confirm": {
				Type:     "boolean",
				Desc:     "是否确认写入。false 或不传只预览；true 时必须同时传入匹配的 confirmToken。",
				Required: false,
			},
			"confirmToken": {
				Type:     "string",
				Desc:     "第一次预览返回的确认令牌；只有 confirm=true 时需要。",
				Required: false,
			},
		},
		func(args string) (string, error) {
			input := parseFollowedStockPositionInput(args)
			if err := validateFollowedStockPositionInput(input); err != nil {
				return "", err
			}

			input.StockCode = normalizeFollowedStockPositionCode(input.StockCode)
			token := followedStockPositionConfirmToken(input)
			warnings := followedStockPositionWarnings(input)
			if !input.Confirm {
				return formatFollowedStockPositionPreview(input, token, warnings), nil
			}
			if strings.TrimSpace(input.ConfirmToken) != token {
				return "", fmt.Errorf("confirmToken 不匹配；请先预览并使用同一组参数返回的确认令牌")
			}

			if err := applyFollowedStockPosition(input); err != nil {
				return "", err
			}
			updated := data.NewStockDataApi().GetFollowedStockByStockCode(input.StockCode)
			return formatFollowedStockPositionResult(updated, warnings), nil
		},
	)
}

func parseFollowedStockPositionInput(args string) followedStockPositionInput {
	root := gjson.Parse(args)
	return followedStockPositionInput{
		StockCode:          strings.TrimSpace(root.Get("stockCode").String()),
		StockName:          strings.TrimSpace(root.Get("stockName").String()),
		CostPrice:          root.Get("costPrice").Float(),
		Volume:             root.Get("volume").Int(),
		EntryPrice:         root.Get("entryPrice").Float(),
		TakeProfitPrice:    root.Get("takeProfitPrice").Float(),
		StopLossPrice:      root.Get("stopLossPrice").Float(),
		AlarmChangePercent: root.Get("alarmChangePercent").Float(),
		AlarmPrice:         root.Get("alarmPrice").Float(),
		Sort:               root.Get("sort").Int(),
		Reason:             strings.TrimSpace(root.Get("reason").String()),
		Confirm:            root.Get("confirm").Bool(),
		ConfirmToken:       strings.TrimSpace(root.Get("confirmToken").String()),
	}
}

func validateFollowedStockPositionInput(input followedStockPositionInput) error {
	if strings.TrimSpace(input.StockCode) == "" {
		return fmt.Errorf("stockCode 不能为空")
	}
	if input.CostPrice <= 0 {
		return fmt.Errorf("costPrice 必须大于 0")
	}
	if input.Volume < 0 {
		return fmt.Errorf("volume 不能小于 0")
	}
	if input.EntryPrice < 0 || input.TakeProfitPrice < 0 || input.StopLossPrice < 0 || input.AlarmPrice < 0 {
		return fmt.Errorf("价格类字段不能小于 0")
	}
	if input.AlarmChangePercent < 0 {
		return fmt.Errorf("alarmChangePercent 不能小于 0")
	}
	if input.Sort < 0 {
		return fmt.Errorf("sort 不能小于 0")
	}
	return nil
}

func normalizeFollowedStockPositionCode(stockCode string) string {
	return data.NormalizeFollowedStockCode(stockCode)
}

func followedStockPositionConfirmToken(input followedStockPositionInput) string {
	input.Confirm = false
	input.ConfirmToken = ""
	data, _ := json.Marshal(input)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:16]
}

func followedStockPositionWarnings(input followedStockPositionInput) []string {
	var warnings []string
	if input.TakeProfitPrice > 0 && input.TakeProfitPrice <= input.CostPrice {
		warnings = append(warnings, "止盈价不高于成本价，请确认这是否符合你的计划。")
	}
	if input.StopLossPrice > 0 && input.StopLossPrice >= input.CostPrice {
		warnings = append(warnings, "止损价不低于成本价，请确认这是否符合你的计划。")
	}
	if input.AlarmChangePercent == 0 && input.AlarmPrice == 0 {
		warnings = append(warnings, "未设置涨跌提醒或股价提醒。")
	}
	if input.TakeProfitPrice == 0 || input.StopLossPrice == 0 {
		warnings = append(warnings, "止盈价或止损价未完整设置。")
	}
	return warnings
}

func applyFollowedStockPosition(input followedStockPositionInput) error {
	if err := ensureFollowedStockPosition(input); err != nil {
		return err
	}

	updates := map[string]any{
		"cost_price":           input.CostPrice,
		"volume":               input.Volume,
		"entry_price":          input.EntryPrice,
		"take_profit_price":    input.TakeProfitPrice,
		"stop_loss_price":      input.StopLossPrice,
		"alarm_change_percent": input.AlarmChangePercent,
		"alarm_price":          input.AlarmPrice,
	}
	if input.StockName != "" {
		updates["name"] = input.StockName
	}

	result := db.Dao.Model(&data.FollowedStock{}).
		Where("stock_code = ?", strings.ToLower(input.StockCode)).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := db.Dao.Model(&data.FollowedStock{}).
			Where("stock_code = ?", strings.ToLower(input.StockCode)).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("设置失败：未找到自选股 %s", input.StockCode)
		}
	}
	if input.Sort > 0 {
		data.NewStockDataApi().SetStockSort(input.Sort, input.StockCode)
	}
	return nil
}

func ensureFollowedStockPosition(input followedStockPositionInput) error {
	var count int64
	if err := db.Dao.Model(&data.FollowedStock{}).
		Where("stock_code = ?", strings.ToLower(input.StockCode)).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	name := input.StockName
	if name == "" {
		name = input.StockCode
	}
	return db.Dao.Model(&data.FollowedStock{}).Create(&data.FollowedStock{
		StockCode:          strings.ToLower(input.StockCode),
		Name:               name,
		CostPrice:          input.CostPrice,
		Volume:             input.Volume,
		Time:               time.Now(),
		Sort:               data.FollowedStockDefaultSort,
		AlarmChangePercent: input.AlarmChangePercent,
		AlarmPrice:         input.AlarmPrice,
		EntryPrice:         input.EntryPrice,
		TakeProfitPrice:    input.TakeProfitPrice,
		StopLossPrice:      input.StopLossPrice,
	}).Error
}

func formatFollowedStockPositionPreview(input followedStockPositionInput, token string, warnings []string) string {
	var b strings.Builder
	b.WriteString("# 自选股持仓提醒设置预览\n\n")
	b.WriteString("本次调用未写入数据库。必须向用户复述下表内容，等待明确二次确认后，再用同一组参数加 `confirm=true` 和 `confirmToken` 调用。\n\n")
	b.WriteString(formatFollowedStockPositionTable(input, token))
	if input.Reason != "" {
		b.WriteString("\n## Agent 判断依据\n\n")
		b.WriteString(input.Reason)
		b.WriteString("\n")
	}
	if len(warnings) > 0 {
		b.WriteString("\n## 需要确认的点\n\n")
		for _, warning := range warnings {
			b.WriteString("- ")
			b.WriteString(warning)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n## 确认令牌\n\n")
	b.WriteString("`")
	b.WriteString(token)
	b.WriteString("`\n")
	return b.String()
}

func formatFollowedStockPositionResult(stock data.FollowedStock, warnings []string) string {
	var b strings.Builder
	b.WriteString("# 自选股持仓提醒已写入\n\n")
	b.WriteString("| 项目 | 值 |\n| --- | --- |\n")
	b.WriteString(fmt.Sprintf("| 股票代码 | %s |\n", stock.StockCode))
	b.WriteString(fmt.Sprintf("| 股票名称 | %s |\n", stock.Name))
	b.WriteString(fmt.Sprintf("| 成本价 | %.3f |\n", stock.CostPrice))
	b.WriteString(fmt.Sprintf("| 持仓数量 | %d |\n", stock.Volume))
	b.WriteString(fmt.Sprintf("| 开仓价 | %.3f |\n", stock.EntryPrice))
	b.WriteString(fmt.Sprintf("| 止盈价 | %.3f |\n", stock.TakeProfitPrice))
	b.WriteString(fmt.Sprintf("| 止损价 | %.3f |\n", stock.StopLossPrice))
	b.WriteString(fmt.Sprintf("| 涨跌提醒(%%) | %.3f |\n", stock.AlarmChangePercent))
	b.WriteString(fmt.Sprintf("| 股价提醒 | %.3f |\n", stock.AlarmPrice))
	b.WriteString(fmt.Sprintf("| 排序 | %d |\n", stock.Sort))
	if len(warnings) > 0 {
		b.WriteString("\n## 已确认但仍需留意\n\n")
		for _, warning := range warnings {
			b.WriteString("- ")
			b.WriteString(warning)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func formatFollowedStockPositionTable(input followedStockPositionInput, token string) string {
	var b strings.Builder
	b.WriteString("| 项目 | 值 |\n| --- | --- |\n")
	b.WriteString(fmt.Sprintf("| 股票代码 | %s |\n", input.StockCode))
	if input.StockName != "" {
		b.WriteString(fmt.Sprintf("| 股票名称 | %s |\n", input.StockName))
	}
	b.WriteString(fmt.Sprintf("| 成本价 | %.3f |\n", input.CostPrice))
	b.WriteString(fmt.Sprintf("| 持仓数量 | %d |\n", input.Volume))
	b.WriteString(fmt.Sprintf("| 开仓价 | %.3f |\n", input.EntryPrice))
	b.WriteString(fmt.Sprintf("| 止盈价 | %.3f |\n", input.TakeProfitPrice))
	b.WriteString(fmt.Sprintf("| 止损价 | %.3f |\n", input.StopLossPrice))
	b.WriteString(fmt.Sprintf("| 涨跌提醒(%%) | %.3f |\n", input.AlarmChangePercent))
	b.WriteString(fmt.Sprintf("| 股价提醒 | %.3f |\n", input.AlarmPrice))
	if input.Sort > 0 {
		b.WriteString(fmt.Sprintf("| 排序 | %d |\n", input.Sort))
	}
	b.WriteString(fmt.Sprintf("| confirmToken | `%s` |\n", token))
	return b.String()
}

type followedStockReadable struct {
	StockCode          string  `md:"股票代码"`
	Name               string  `md:"股票名称"`
	CostPrice          float64 `md:"成本价格"`
	Volume             int64   `md:"持仓数量"`
	EntryPrice         float64 `md:"开仓价"`
	TakeProfitPrice    float64 `md:"止盈价"`
	StopLossPrice      float64 `md:"止损价"`
	AlarmChangePercent float64 `md:"涨跌提醒(%)"`
	AlarmPrice         float64 `md:"股价提醒"`
	Sort               int64   `md:"排序"`
}

func followedStockReadableRow(s data.FollowedStock) followedStockReadable {
	return followedStockReadable{
		StockCode:          s.StockCode,
		Name:               s.Name,
		CostPrice:          s.CostPrice,
		Volume:             s.Volume,
		EntryPrice:         s.EntryPrice,
		TakeProfitPrice:    s.TakeProfitPrice,
		StopLossPrice:      s.StopLossPrice,
		AlarmChangePercent: s.AlarmChangePercent,
		AlarmPrice:         s.AlarmPrice,
		Sort:               s.Sort,
	}
}
