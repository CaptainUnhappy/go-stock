package cli

import (
	"fmt"
	"sort"
	"strings"
)

type TreeNode struct {
	Label    string
	Path     string
	Example  string
	Children []TreeNode
}

func CommandTree() TreeNode {
	return TreeNode{
		Label: "go-stock",
		Children: []TreeNode{
			portfolioTree(),
			marketTree(),
			indexTree(),
			klineTree(),
			fundTree(),
			calendarTree(),
			researchTree(),
			rawToolTree(),
		},
	}
}

func RenderHelp() string {
	var b strings.Builder
	b.WriteString("# go-stock CLI 功能树\n\n")
	b.WriteString("CLI 路径和 GUI 菜单对齐；原对外 MCP 工具已归档到 `tool <原工具名>` 兼容层。\n\n")
	tree := CommandTree()
	for i, child := range tree.Children {
		if i > 0 {
			b.WriteString("\n")
		}
		renderNode(&b, child, "", i == len(tree.Children)-1, true)
	}
	b.WriteString("\n## 常用示例\n\n")
	b.WriteString("```powershell\n")
	b.WriteString(".\\go-stock-cli.exe help\n")
	b.WriteString(".\\go-stock-cli.exe market news\n")
	b.WriteString(".\\go-stock-cli.exe market major-index\n")
	b.WriteString(".\\go-stock-cli.exe market major-index --name 上证指数\n")
	b.WriteString(".\\go-stock-cli.exe index history --code 883418.TI --start 2025-01-01 --end 2026-08-10\n")
	b.WriteString(".\\go-stock-cli.exe --json index history --name 微盘股 --start 2025-01-01 --end 2026-08-10\n")
	b.WriteString(".\\go-stock-cli.exe market industry-rank concept-money --sort netamount --limit 20\n")
	b.WriteString(".\\go-stock-cli.exe market money-flow stock --sort r0_net --limit 20\n")
	b.WriteString(".\\go-stock-cli.exe kline show --stock-code 002335 --k-line-type day --adjust qfq --limit 120\n")
	b.WriteString(".\\go-stock-cli.exe kline signals --stock-code 002335 --k-line-type day --adjust qfq --limit 250\n")
	b.WriteString(".\\go-stock-cli.exe portfolio list\n")
	b.WriteString(".\\go-stock-cli.exe portfolio group rename --group-id 1 --new-name 短线观察\n")
	b.WriteString(".\\go-stock-cli.exe portfolio position set --stock-code 600237 --cost-price 12.56 --volume 300\n")
	b.WriteString(".\\go-stock-cli.exe fund ranking --page-size 20\n")
	b.WriteString(".\\go-stock-cli.exe calendar is-trading-day --date 2026-07-03\n")
	b.WriteString(".\\go-stock-cli.exe calendar next-trading-day --date 2026-07-03\n")
	b.WriteString(".\\go-stock-cli.exe tool list\n")
	b.WriteString(".\\go-stock-cli.exe tool info --name GetStockOrderBook\n")
	b.WriteString(".\\go-stock-cli.exe tool GetStockOrderBook --stock-code 600237\n")
	b.WriteString("```\n")
	return b.String()
}

func CommandPathsFromTree() []string {
	var paths []string
	var walk func(TreeNode)
	walk = func(n TreeNode) {
		if strings.TrimSpace(n.Path) != "" {
			paths = append(paths, NormalizeCommandPath(n.Path))
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(CommandTree())
	sort.Strings(paths)
	return paths
}

func renderNode(b *strings.Builder, n TreeNode, prefix string, last bool, root bool) {
	connector := ""
	nextPrefix := prefix
	if !root {
		if last {
			connector = "└─ "
			nextPrefix = prefix + "   "
		} else {
			connector = "├─ "
			nextPrefix = prefix + "│  "
		}
	}

	label := n.Label
	if n.Path != "" {
		label = fmt.Sprintf("%s  `%s`", label, NormalizeCommandPath(n.Path))
	} else if n.Example != "" {
		label = fmt.Sprintf("%s  `%s`", label, n.Example)
	}
	b.WriteString(prefix)
	b.WriteString(connector)
	b.WriteString(label)
	b.WriteString("\n")
	for i, child := range n.Children {
		renderNode(b, child, nextPrefix, i == len(n.Children)-1, false)
	}
}

func portfolioTree() TreeNode {
	return TreeNode{Label: "股票自选", Children: []TreeNode{
		{Label: "自选列表", Children: []TreeNode{
			{Label: "查看全部", Path: "portfolio list"},
			{Label: "查看分组内股票"},
			{Label: "搜索股票", Path: "portfolio search"},
			{Label: "关注股票", Path: "portfolio add"},
			{Label: "取消关注股票", Path: "portfolio remove"},
		}},
		{Label: "持仓/提醒设置", Children: []TreeNode{
			{Label: "查看持仓设置", Path: "portfolio position get"},
			{Label: "设置股票成本/数量/提醒/价位线", Path: "portfolio position set"},
			{Label: "股票成本"},
			{Label: "股票数量"},
			{Label: "涨跌提醒"},
			{Label: "股价提醒"},
			{Label: "开仓价"},
			{Label: "止盈价"},
			{Label: "止损价"},
			{Label: "股票排序"},
		}},
		{Label: "分组管理", Children: []TreeNode{
			{Label: "查看分组", Path: "portfolio group list"},
			{Label: "添加分组", Path: "portfolio group add"},
			{Label: "重命名分组", Path: "portfolio group rename"},
			{Label: "设置分组", Path: "portfolio group assign"},
			{Label: "移出分组", Path: "portfolio group remove"},
		}},
		{Label: "个股快捷查看", Children: []TreeNode{
			{Label: "分时", Path: "portfolio view minute"},
			{Label: "日K", Path: "portfolio view daily-k"},
			{Label: "多周期K线", Path: "portfolio view multi-k"},
			{Label: "资金", Path: "portfolio view money"},
			{Label: "详情", Path: "portfolio view detail"},
			{Label: "公告", Path: "portfolio view notice"},
			{Label: "研报", Path: "portfolio view report"},
		}},
	}}
}

func marketTree() TreeNode {
	return TreeNode{Label: "市场行情", Children: []TreeNode{
		{Label: "市场快讯", Path: "market news", Children: []TreeNode{
			{Label: "主要股指"},
			{Label: "涨跌家数比"},
			{Label: "涨跌停家数比"},
			{Label: "当日异动次数最多的概念"},
			{Label: "查看热词"},
			{Label: "按天涨跌/涨跌停分析"},
			{Label: "历史异动分析"},
			{Label: "异动排行"},
			{Label: "利好/利空排行"},
			{Label: "快讯列表"},
		}},
		{Label: "全球股指", Path: "market global-index"},
		{Label: "重大指数", Path: "market major-index"},
		{Label: "行业排名", Children: []TreeNode{
			{Label: "行业涨幅排名", Path: "market industry-rank gain"},
			{Label: "行业资金排名", Path: "market industry-rank money"},
			{Label: "证监会行业资金排名", Path: "market industry-rank csrc-money"},
			{Label: "概念板块资金排名", Path: "market industry-rank concept-money"},
		}},
		{Label: "个股资金流向", Path: "market money-flow stock"},
		{Label: "板块资金流向", Children: []TreeNode{
			{Label: "板块列表", Path: "market money-flow bk list"},
			{Label: "最新资金排名", Path: "market money-flow bk latest"},
			{Label: "指定日期资金排名", Path: "market money-flow bk date"},
			{Label: "板块资金趋势", Path: "market money-flow bk trend"},
		}},
		{Label: "概念资金流向", Children: []TreeNode{
			{Label: "概念列表", Path: "market money-flow concept list"},
			{Label: "最新资金排名", Path: "market money-flow concept latest"},
			{Label: "指定日期资金排名", Path: "market money-flow concept date"},
			{Label: "概念资金趋势", Path: "market money-flow concept trend"},
		}},
		{Label: "龙虎榜", Path: "market billboard"},
		{Label: "个股研报", Path: "market stock-report"},
		{Label: "公司公告", Path: "market announcement"},
		{Label: "行业研究", Path: "market industry-research"},
		{Label: "当前热门", Children: []TreeNode{
			{Label: "全球", Path: "market hot global"},
			{Label: "沪深", Path: "market hot cn"},
			{Label: "港股", Path: "market hot hk"},
			{Label: "美股", Path: "market hot us"},
			{Label: "热门话题", Path: "market hot topic"},
			{Label: "重大事件时间轴", Path: "market hot timeline"},
			{Label: "财经日历", Path: "market hot calendar"},
		}},
	}}
}

func klineTree() TreeNode {
	return TreeNode{Label: "K线分析", Children: []TreeNode{
		{Label: "标的选择", Children: []TreeNode{
			{Label: "搜索股票/指数", Path: "kline search"},
			{Label: "最近查看", Path: "kline recent"},
		}},
		{Label: "K线展示", Path: "kline show", Children: []TreeNode{
			{Label: "周期：1分/5分/15分/30分/60分/日K/周K/月K/季K/年K"},
			{Label: "复权：qfq前复权/hfq后复权/none不复权"},
			{Label: "指标：趋势/波动/动量/量价/强度"},
			{Label: "价位线：开仓价/止损价/止盈价"},
		}},
		{Label: "指标信号汇总", Path: "kline signals", Children: []TreeNode{
			{Label: "看多/看空/震荡/中性统计"},
			{Label: "指标标签：MA/MACD/RSI/KDJ/BOLL 等"},
		}},
	}}
}

func indexTree() TreeNode {
	return TreeNode{Label: "指数数据", Children: []TreeNode{
		{Label: "指定日期区间日 K 线（.SH/.SZ/.CSI/.TI）", Path: "index history", Children: []TreeNode{
			{Label: "日期范围包含首尾，最大 10 年"},
			{Label: "周期：仅 1d；指数不复权"},
		}},
	}}
}

func fundTree() TreeNode {
	return TreeNode{Label: "基金", Children: []TreeNode{
		{Label: "基金自选", Path: "fund follow"},
		{Label: "基金排行", Path: "fund ranking"},
		{Label: "基金搜索", Path: "fund search"},
		{Label: "基金详情", Path: "fund info"},
		{Label: "基金K线", Path: "fund kline"},
		{Label: "基金净值", Path: "fund nav"},
		{Label: "基金持仓", Path: "fund holdings"},
	}}
}

func calendarTree() TreeNode {
	return TreeNode{Label: "交易日历", Children: []TreeNode{
		{Label: "当前时间", Path: "calendar now"},
		{Label: "是否交易日", Path: "calendar is-trading-day"},
		{Label: "下一交易日", Path: "calendar next-trading-day"},
		{Label: "节假日查询", Path: "calendar holiday"},
		{Label: "年度节假日", Path: "calendar holiday-year"},
		{Label: "批量节假日", Path: "calendar holiday-batch"},
	}}
}

func researchTree() TreeNode {
	return TreeNode{Label: "研究中心", Children: []TreeNode{
		{Label: "AI分析报告（禁用）", Path: "research ai-report"},
		{Label: "股票推荐记录（禁用）", Path: "research recommend"},
		{Label: "异动监控", Path: "research changes"},
		{Label: "涨停梯队", Path: "research uplimit"},
		{Label: "提示词模板（禁用）", Path: "research prompt-template"},
		{Label: "提示词广场（禁用）", Path: "research prompt-plaza"},
		{Label: "问答广场（禁用）", Path: "research qa-plaza"},
		{Label: "形态选股", Path: "research pattern-screen"},
		{Label: "指标选股", Path: "research indicator-screen"},
		{Label: "定时任务（禁用）", Path: "research cron-task"},
		{Label: "交易日志（禁用）", Path: "research trade-log"},
	}}
}

func rawToolTree() TreeNode {
	return TreeNode{Label: "归档工具兼容层", Children: []TreeNode{
		{Label: "查看全部已迁移工具", Path: "tool list"},
		{Label: "查看工具作用和参数", Path: "tool info"},
		{Label: "按原工具名调用", Path: "tool get-stock-order-book"},
	}}
}

func majorIndexNodes() []TreeNode {
	catalog := majorIndexCatalog()
	nodes := make([]TreeNode, 0, len(catalog))
	for _, spec := range catalog {
		nodes = append(nodes, TreeNode{Label: spec.Name, Example: "market major-index --name " + spec.Name})
	}
	return nodes
}
