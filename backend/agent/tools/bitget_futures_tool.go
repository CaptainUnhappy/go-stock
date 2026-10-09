package tools

import (
	"go-stock/backend/data"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// GetBitgetFuturesTools exposes the Bitget US-stock perpetual data through the
// same invokable interface used by the standalone CLI.
func GetBitgetFuturesTools() []tool.BaseTool {
	return []tool.BaseTool{
		NewDataToolWrapper(
			"GetBitgetFuturesMarket",
			"获取 Bitget 美股永续合约行情榜单，包含最新价、24h涨跌幅、成交额和资金费率。",
			map[string]*schema.ParameterInfo{
				"sort":    {Type: "string", Desc: "排序：percent（默认）、amount、funding。"},
				"limit":   {Type: "number", Desc: "返回条数，默认 20，最大 100。"},
				"symbols": {Type: "string", Desc: "可选合约列表，如 aapl,tsla 或 bt:aaplusdt。"},
			},
			func(args string) (string, error) { return data.BitgetFuturesMarketMarkdown(args) },
		),
		NewDataToolWrapper(
			"GetBitgetFuturesKLine",
			"获取 Bitget 美股永续合约 K 线数据。",
			map[string]*schema.ParameterInfo{
				"symbol":   {Type: "string", Desc: "合约标识，如 aapl、AAPLUSDT、bt:aaplusdt。", Required: true},
				"interval": {Type: "string", Desc: "周期：day/week/month 或 1/5/15/30/60/120/240。"},
				"limit":    {Type: "number", Desc: "K 线根数，默认 90，最大 1000。"},
			},
			func(args string) (string, error) { return data.BitgetFuturesKLineMarkdown(args) },
		),
		NewDataToolWrapper(
			"GetBitgetFuturesDerivatives",
			"获取 Bitget 美股永续合约的资金费率、标记价、指数价和未平仓量。",
			map[string]*schema.ParameterInfo{
				"symbol": {Type: "string", Desc: "合约标识，如 aapl、AAPLUSDT、bt:aaplusdt。", Required: true},
			},
			func(args string) (string, error) { return data.BitgetFuturesDerivativesMarkdown(args) },
		),
	}
}
