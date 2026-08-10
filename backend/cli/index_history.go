package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go-stock/backend/data"
	"go-stock/backend/util"
)

type indexHistoryRow struct {
	Date    string `md:"日期"`
	Open    string `md:"开"`
	High    string `md:"高"`
	Low     string `md:"低"`
	Close   string `md:"收"`
	Volume  string `md:"成交量"`
	Amount  string `md:"成交额"`
	Partial string `md:"状态"`
}

var fetchIndexHistory = data.FetchIndexHistory

func runIndexHistory(ctx context.Context, args map[string]any) (HandlerResult, error) {
	name := strings.TrimSpace(optionalString(args, "name", ""))
	code := strings.TrimSpace(optionalString(args, "code", ""))
	if name == "" && code == "" {
		return HandlerResult{}, fmt.Errorf("--code、--name 至少提供一个")
	}

	var byName majorIndexSpec
	var foundByName bool
	if name != "" {
		byName, foundByName = majorIndexSpecByName(name)
		if !foundByName {
			return HandlerResult{}, fmt.Errorf("未知指数名称 %q；请改用 --code 传入完整后缀代码", name)
		}
	}
	if code != "" {
		normalized, err := data.NormalizeIndexCode(code)
		if err != nil {
			return HandlerResult{}, err
		}
		code = normalized
	}
	if foundByName {
		if code != "" && !strings.EqualFold(code, byName.Code) {
			return HandlerResult{}, fmt.Errorf("--code %s 与 --name %s 指向不同指数（%s）", code, name, byName.Code)
		}
		code = byName.Code
		name = byName.Name
	} else if byCode, found := majorIndexSpecByCode(code); found {
		name = byCode.Name
	} else {
		name = code
	}

	interval := strings.ToLower(strings.TrimSpace(optionalString(args, "interval", "1d")))
	if interval != "1d" {
		return HandlerResult{}, fmt.Errorf("--interval 当前仅支持 1d")
	}
	if hasAny(args, "adjust", "adjustFlag", "adjust-flag", "adjust_flag") {
		return HandlerResult{}, fmt.Errorf("指数没有复权语义，index history 不接受复权参数")
	}
	start, err := requiredString(args, "start")
	if err != nil {
		return HandlerResult{}, fmt.Errorf("--start 必填，格式为 YYYY-MM-DD")
	}
	end, err := requiredString(args, "end")
	if err != nil {
		return HandlerResult{}, fmt.Errorf("--end 必填，格式为 YYYY-MM-DD")
	}

	result, err := fetchIndexHistory(ctx, code, start, end)
	if err != nil {
		return HandlerResult{}, err
	}
	result.Name = name
	rows := make([]indexHistoryRow, 0, len(result.Bars))
	for _, bar := range result.Bars {
		state := ""
		if bar.Partial {
			state = "盘中"
		}
		rows = append(rows, indexHistoryRow{
			Date: bar.Date, Open: cliIndexNumber(bar.Open), High: cliIndexNumber(bar.High), Low: cliIndexNumber(bar.Low), Close: cliIndexNumber(bar.Close),
			Volume: cliIndexNumber(bar.Volume), Amount: cliIndexNumber(bar.Amount), Partial: state,
		})
	}
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# %s（%s）指数历史 K 线\n\n", name, code))
	output.WriteString(fmt.Sprintf("- 数据源：%s\n- 时间范围：%s 至 %s（含首尾）\n- 周期：1d\n- 数据时间：%s\n", result.Source, result.Start, result.End, result.AsOf))
	if len(result.Warnings) > 0 {
		output.WriteString("\n## 提示\n\n")
		for _, warning := range result.Warnings {
			output.WriteString("- " + warning + "\n")
		}
	}
	output.WriteString("\n")
	output.WriteString(util.MarkdownTableWithTitle(fmt.Sprintf("日 K（共 %d 条）", len(rows)), rows))
	return HandlerResult{Output: strings.TrimSpace(output.String()), Data: result}, nil
}

func cliIndexNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
