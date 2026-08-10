package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"go-stock/backend/agent/tools"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/models"
	"go-stock/backend/util"
)

func commandDefinitions() []Command {
	r := newToolRunner()
	commands := []Command{
		read("portfolio list", "查看自选股", r.call("GetFollowedStocks", nil)),
		read("portfolio search", "搜索股票", r.call("QueryStockCodeInfo", mapArgs("keyword", "searchWord"))),
		write("portfolio add", "关注股票", runPortfolioAdd),
		write("portfolio remove", "取消关注股票", runPortfolioRemove),
		read("portfolio position get", "查看持仓提醒设置", r.call("GetFollowedStocks", nil)),
		write("portfolio position set", "设置持仓提醒", runPortfolioPositionSet(r)),
		read("portfolio group list", "查看分组", runPortfolioGroupList),
		write("portfolio group add", "添加分组", runPortfolioGroupAdd),
		write("portfolio group rename", "重命名分组", runPortfolioGroupRename),
		write("portfolio group assign", "股票加入分组", runPortfolioGroupAssign),
		write("portfolio group remove", "股票移出分组", runPortfolioGroupRemove),
		read("portfolio view minute", "查看分时", r.call("GetStockMinuteData", mapArgs("code", "stockCode"))),
		read("portfolio view daily-k", "查看日K", r.call("GetEastMoneyKLineWithMA", withDefaults(mapArgs("code", "stockCode"), map[string]any{"kLineType": "day", "limit": 60}))),
		read("portfolio view multi-k", "查看多周期K线", r.call("GetEastMoneyKLineWithMA", withDefaults(mapArgs("code", "stockCode"), map[string]any{"kLineType": "day", "limit": 60}))),
		read("portfolio view money", "查看个股资金", r.call("GetStockMoneyData", nil)),
		read("portfolio view detail", "查看个股详情", r.call("GetStockInfo", mapArgs("code", "stockCode"))),
		read("portfolio view notice", "查看个股公告", r.call("GetStockNotice", mapArgs("code", "stockCodes", "stockCode", "stockCodes"))),
		read("portfolio view report", "查看个股研报", r.call("GetStockResearchReport", mapArgs("code", "stockCode"))),

		read("market news", "市场快讯", runMarketNews),
		read("market global-index", "全球股指", r.call("GlobalStockIndexesReadable", nil)),
		read("market major-index", "重大指数", runMajorIndex),
		readStructured("index history", "指数历史 K 线", runIndexHistory),
		read("market industry-rank gain", "行业涨幅排名", r.call("GetIndustryRank", withDefaults(nil, map[string]any{"sort": "0", "limit": 20}))),
		read("market industry-rank money", "行业资金排名", r.call("GetIndustryMoneyRank", withDefaults(nil, map[string]any{"fenlei": "0", "sort": "netamount", "limit": 20}))),
		read("market industry-rank csrc-money", "证监会行业资金排名", r.call("GetIndustryMoneyRank", withDefaults(nil, map[string]any{"fenlei": "2", "sort": "netamount", "limit": 20}))),
		read("market industry-rank concept-money", "概念板块资金排名", r.call("GetIndustryMoneyRank", withDefaults(nil, map[string]any{"fenlei": "1", "sort": "netamount", "limit": 20}))),
		read("market money-flow stock", "个股资金流向", r.call("GetMoneyRankSina", nil)),
		read("market money-flow bk list", "板块列表", r.call("GetAllBKCodes", nil)),
		read("market money-flow bk latest", "板块最新资金排名", r.call("GetBKFundFlowTopList", nil)),
		read("market money-flow bk date", "板块指定日期资金排名", r.call("GetBKFundFlowTopListByDate", nil)),
		read("market money-flow bk trend", "板块资金趋势", r.call("GetBKFundFlowList", nil)),
		read("market money-flow concept list", "概念列表", runConceptList),
		read("market money-flow concept latest", "概念最新资金排名", runConceptLatest),
		read("market money-flow concept date", "概念指定日期资金排名", runConceptDate),
		read("market money-flow concept trend", "概念资金趋势", runConceptTrend),
		read("market billboard", "龙虎榜", r.call("GetLongTigerList", nil)),
		read("market stock-report", "个股研报", r.call("GetStockResearchReport", nil)),
		read("market announcement", "公司公告", runMarketAnnouncement(r)),
		read("market industry-research", "行业研究", infoCommand("行业研究的东方财富 AI 生成工具当前不作为 CLI 数据能力开放。请改用 `tool GetIndustryValuation` 查看行业估值，`tool GetSecuritiesCompanyOpinion` 查看券商观点，或用 `tool QueryStockNewsTool`/公告研报类工具做只读研究。")),
		read("market hot global", "当前热门-全球", r.call("GetHotStockList", withDefaults(nil, map[string]any{"marketType": "10", "size": 20}))),
		read("market hot cn", "当前热门-沪深", r.call("GetHotStockList", withDefaults(nil, map[string]any{"marketType": "12", "size": 20}))),
		read("market hot hk", "当前热门-港股", r.call("GetHotStockList", withDefaults(nil, map[string]any{"marketType": "13", "size": 20}))),
		read("market hot us", "当前热门-美股", r.call("GetHotStockList", withDefaults(nil, map[string]any{"marketType": "11", "size": 20}))),
		read("market hot topic", "当前热门-热门话题", runHotTopic),
		read("market hot timeline", "当前热门-重大事件时间轴", r.call("GetHotEventList", nil)),
		read("market hot calendar", "当前热门-财经日历", runInvestCalendar(r)),

		read("kline search", "K线标的搜索", r.call("QueryStockCodeInfo", mapArgs("keyword", "searchWord"))),
		read("kline recent", "K线最近查看", runKlineRecent),
		read("kline show", "K线展示", runKlineShow(r)),
		read("kline signals", "K线指标信号汇总", runKlineSignals(r)),

		read("fund follow", "基金自选", runFundFollow),
		read("fund ranking", "基金排行", runFundRanking),
		read("fund search", "基金搜索", r.call("SearchFund", mapArgs("keyword", "keyword"))),
		read("fund info", "基金详情", r.call("GetFundInfo", mapArgs("code", "fundCode"))),
		read("fund kline", "基金K线", r.call("GetFundKLine", mapArgs("code", "fundCode"))),
		read("fund nav", "基金净值", r.call("GetFundHistoryNetValue", mapArgs("code", "fundCode"))),
		read("fund holdings", "基金持仓", r.call("GetFundTop10Holdings", mapArgs("code", "fundCode"))),

		read("calendar now", "当前时间", r.call("GetCurrentTime", nil)),
		read("calendar is-trading-day", "判断是否交易日", r.call("IsTradingDay", nil)),
		read("calendar next-trading-day", "查询下一交易日", r.call("GetNextTradingDay", mapArgs("date", "startDate"))),
		read("calendar holiday", "查询节假日", r.call("GetHolidayInfo", nil)),
		read("calendar holiday-year", "查询年度节假日", r.call("GetHolidayYear", nil)),
		read("calendar holiday-batch", "批量查询节假日", r.call("GetHolidayBatch", nil)),

		read("research ai-report", "AI分析报告", infoCommand("AI分析报告工具已按项目策略禁用；CLI 不读取历史 AI 分析记录。")),
		read("research recommend", "股票推荐记录", infoCommand("AI推荐/股票推荐记录工具已按项目策略禁用；CLI 不读取或写入 AI 推荐记录。")),
		read("research changes", "异动监控", r.call("GetStockChanges", nil)),
		read("research uplimit", "涨停梯队", runResearchUplimit(r)),
		read("research prompt-template", "提示词模板", infoCommand("提示词模板已按项目策略禁用；CLI 不读取、创建或修改提示词模板。")),
		read("research prompt-plaza", "提示词广场", infoCommand("提示词广场已按项目策略禁用；CLI 不访问外部提示词广场服务。")),
		read("research qa-plaza", "问答广场", infoCommand("问答广场已按项目策略禁用；CLI 不访问外部问答广场服务。")),
		read("research pattern-screen", "形态选股", r.call("FilterStocks", nil)),
		read("research indicator-screen", "指标选股", r.call("SearchStockByIndicators", mapArgs("query", "query"))),
		read("research cron-task", "定时任务", infoCommand("定时任务已按项目策略禁用；CLI 不查看、创建、执行、暂停或删除本地定时任务。")),
		read("research trade-log", "交易日志", infoCommand("交易日志已按项目策略禁用；CLI 不读取、创建、更新或删除交易日志。")),
	}
	commands = append(commands,
		read("tool list", "查看归档工具清单", r.runRawToolList),
		read("tool info", "查看归档工具参数", r.runRawToolInfo),
	)
	commands = append(commands, r.rawToolCommands()...)
	return commands
}

type rawToolEntry struct {
	Name string
	Desc string
	Info *schema.ToolInfo
	Tool einotool.InvokableTool
}

type toolRunner struct {
	tools   map[string]einotool.InvokableTool
	entries map[string]rawToolEntry
	aliases map[string]string
}

func newToolRunner() *toolRunner {
	entries := buildToolEntries()
	r := &toolRunner{
		tools:   map[string]einotool.InvokableTool{},
		entries: entries,
		aliases: map[string]string{},
	}
	for name, entry := range entries {
		r.tools[name] = entry.Tool
		for _, alias := range rawToolAliases(name) {
			r.aliases[normalizeRawToolKey(alias)] = name
		}
	}
	return r
}

func buildToolRegistry() map[string]einotool.InvokableTool {
	registry := map[string]einotool.InvokableTool{}
	for name, entry := range buildToolEntries() {
		registry[name] = entry.Tool
	}
	return registry
}

func buildToolEntries() map[string]rawToolEntry {
	source := []einotool.BaseTool{
		tools.GetQueryStockCodeInfoTool(),
		tools.GetQueryStockNewsTool(),
		tools.GetQueryBKDictTool(),
	}
	source = append(source, tools.GetHolidayTools()...)
	source = append(source, tools.GetAllDataTools()...)
	source = append(source, tools.GetSetFollowedStockPositionTool())

	entries := map[string]rawToolEntry{}
	for _, candidate := range source {
		info, err := candidate.Info(context.Background())
		if err != nil || info == nil || strings.TrimSpace(info.Name) == "" {
			continue
		}
		invokable, ok := candidate.(einotool.InvokableTool)
		if !ok {
			continue
		}
		if _, exists := entries[info.Name]; !exists {
			entries[info.Name] = rawToolEntry{
				Name: info.Name,
				Desc: info.Desc,
				Info: info,
				Tool: invokable,
			}
		}
	}
	return entries
}

func (r *toolRunner) call(toolName string, mapper func(map[string]any) map[string]any) Handler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		tool, ok := r.tools[toolName]
		if !ok {
			return "", fmt.Errorf("underlying tool %q is not registered", toolName)
		}
		args = cloneArgs(args)
		normalizeCommonArgs(args)
		if mapper != nil {
			args = mapper(args)
		}
		if err := validateRawToolArgs(toolName, args); err != nil {
			return "", err
		}
		if isSingleStockArchiveTool(toolName) {
			codes := stockCodesFromCLIArgs(args)
			if len(codes) > 1 {
				return r.callPerStockTool(ctx, toolName, tool, args, codes)
			}
		}
		out, err := invokeToolSafely(ctx, toolName, tool, args)
		if err != nil {
			return "", err
		}
		if toolName == "GetStockOrderBook" && shouldFallbackOrderBook(out) {
			if fallback, ok := r.tools["GetStockInfo"]; ok {
				fallbackOut, fallbackErr := invokeToolSafely(ctx, "GetStockInfo", fallback, args)
				if fallbackErr == nil && strings.TrimSpace(fallbackOut) != "" && !isEmptyStockInfoOutput(fallbackOut) {
					return strings.TrimSpace(out) + "\n\n## 盘口兜底\n\n`GetStockOrderBook` 未返回盘口数据，已自动改用 `GetStockInfo`。`GetStockInfo` 同样包含买一至买五、卖一至卖五；收盘后请按最近快照理解，不代表仍可成交。\n\n" + strings.TrimSpace(fallbackOut), nil
				}
			}
		}
		return out, nil
	}
}

func (r *toolRunner) callPerStockTool(ctx context.Context, toolName string, tool einotool.InvokableTool, args map[string]any, codes []string) (string, error) {
	seen := map[string]bool{}
	var sections []string
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		nextArgs := cloneArgs(args)
		nextArgs["stockCode"] = code
		delete(nextArgs, "stockCodes")
		out, err := invokeToolSafely(ctx, toolName, tool, nextArgs)
		if err != nil {
			sections = append(sections, fmt.Sprintf("## %s\n\n%s 调用失败：%v", code, toolName, err))
			continue
		}
		sections = append(sections, fmt.Sprintf("## %s\n\n%s", code, strings.TrimSpace(out)))
	}
	if len(sections) == 0 {
		return "", fmt.Errorf("no valid stock codes")
	}
	return strings.Join(sections, "\n\n"), nil
}

func invokeToolSafely(ctx context.Context, toolName string, tool einotool.InvokableTool, args map[string]any) (out string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%s panic: %v", toolName, recovered)
		}
	}()
	payload, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	return tool.InvokableRun(ctx, string(payload))
}

func stockCodesFromCLIArgs(args map[string]any) []string {
	var codes []string
	appendCode := func(raw any) {
		for _, part := range strings.FieldsFunc(fmt.Sprint(raw), func(r rune) bool {
			return r == ',' || r == '，' || r == ';' || r == '；' || r == ' ' || r == '\t' || r == '\r' || r == '\n'
		}) {
			code := strings.TrimSpace(part)
			if code != "" {
				codes = append(codes, code)
			}
		}
	}
	if v, ok := args["stockCode"]; ok {
		appendCode(v)
	}
	if v, ok := args["stockCodes"]; ok {
		switch list := v.(type) {
		case []string:
			for _, code := range list {
				appendCode(code)
			}
		case []any:
			for _, code := range list {
				appendCode(code)
			}
		default:
			appendCode(list)
		}
	}
	return codes
}

func isSingleStockArchiveTool(name string) bool {
	switch name {
	case "GetStockInfo",
		"GetStockConceptInfo",
		"GetStockLatestFinance",
		"GetStockQtrMainFinance",
		"GetStockOrgPredict",
		"GetStockPredictSummary",
		"GetStockValuationPercentile",
		"GetStockMarginTrading",
		"GetStockBlockTrade",
		"GetStockHolderTrend",
		"GetStockBillboard",
		"GetStockOperationDeptTrade":
		return true
	default:
		return false
	}
}

func validateRawToolArgs(toolName string, args map[string]any) error {
	switch toolName {
	case "GetStockInfo", "GetStockOrderBook":
		if _, err := requiredString(args, "stockCode", "stockCodes"); err != nil {
			return fmt.Errorf("%s requires stockCode; use --stockCode 600237, --stock-code 600237, or --args-json '{\"stockCode\":\"600237\"}'", toolName)
		}
	}
	return nil
}

func shouldFallbackOrderBook(out string) bool {
	normalized := strings.ReplaceAll(strings.TrimSpace(out), " ", "")
	for _, marker := range []string{
		"未找到盘口数据",
		"未获取到盘口数据",
		"暂无盘口数据",
		"无盘口数据",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func isEmptyStockInfoOutput(out string) bool {
	normalized := strings.ReplaceAll(strings.TrimSpace(out), " ", "")
	for _, marker := range []string{
		"未找到股票信息",
		"未获取到股票信息",
		"暂无股票信息",
		"无股票信息",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func (r *toolRunner) rawToolCommands() []Command {
	names := make([]string, 0, len(r.entries))
	for name := range r.entries {
		if isBlockedRawTool(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	commands := make([]Command, 0, len(names))
	for _, name := range names {
		toolName := name
		path := rawToolCommandPath(toolName)
		cmd := Command{
			Path:        path,
			Title:       "归档工具：" + toolName,
			Description: r.entries[toolName].Desc,
			ReadOnly:    !isGuardedRawWriteTool(toolName),
			Handler:     r.call(toolName, nil),
		}
		commands = append(commands, cmd)
	}
	return commands
}

func (r *toolRunner) runRawToolList(_ context.Context, args map[string]any) (string, error) {
	includeBlocked := optionalBool(args, "includeBlocked", false)
	names := make([]string, 0, len(r.entries))
	for name := range r.entries {
		if !includeBlocked && isBlockedRawTool(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	type row struct {
		Tool        string `md:"归档工具"`
		CommandPath string `md:"CLI 调用路径"`
		Status      string `md:"状态"`
		Description string `md:"作用"`
	}
	rows := make([]row, 0, len(names))
	for _, name := range names {
		status := "只读"
		if isGuardedRawWriteTool(name) {
			status = "受控写入"
		}
		if isBlockedRawTool(name) {
			status = "禁用：" + blockedRawToolReason(name)
		}
		rows = append(rows, row{
			Tool:        name,
			CommandPath: "tool " + name,
			Status:      status,
			Description: oneLine(r.entries[name].Desc),
		})
	}
	if len(rows) == 0 {
		return "暂无归档工具", nil
	}
	return util.MarkdownTableWithTitle("归档工具 CLI 兼容清单", rows), nil
}

func (r *toolRunner) runRawToolInfo(_ context.Context, args map[string]any) (string, error) {
	name, err := requiredString(args, "name", "tool", "toolName")
	if err != nil {
		return "", err
	}
	entry, ok := r.findRawTool(name)
	if !ok {
		return "", fmt.Errorf("unknown raw tool %q; run `tool list` first", name)
	}

	var b strings.Builder
	b.WriteString("# 归档工具：")
	b.WriteString(entry.Name)
	b.WriteString("\n\n")
	if isBlockedRawTool(entry.Name) {
		b.WriteString("状态：禁用。原因：")
		b.WriteString(blockedRawToolReason(entry.Name))
		b.WriteString("\n\n")
	} else if isGuardedRawWriteTool(entry.Name) {
		b.WriteString("状态：受控写入。必须遵守预览/确认令牌流程。\n\n")
	} else {
		b.WriteString("状态：只读。\n\n")
	}
	b.WriteString("CLI 调用路径：`tool ")
	b.WriteString(entry.Name)
	b.WriteString("`\n\n")
	if strings.TrimSpace(entry.Desc) != "" {
		b.WriteString("## 作用\n\n")
		b.WriteString(entry.Desc)
		b.WriteString("\n\n")
	}
	b.WriteString("## 参数 JSON Schema\n\n")
	b.WriteString("```json\n")
	b.WriteString(rawToolSchemaJSON(entry.Info))
	b.WriteString("\n```\n")
	if aliases := rawToolCLIParamAliases(entry.Name); aliases != "" {
		b.WriteString("\n## CLI 参数名\n\n")
		b.WriteString(aliases)
		b.WriteString("\n")
	}
	if hints := rawToolCLIHints(entry.Name); hints != "" {
		b.WriteString("\n## CLI 调用提示\n\n")
		b.WriteString(hints)
		b.WriteString("\n")
	}
	return b.String(), nil
}

func (r *toolRunner) findRawTool(name string) (rawToolEntry, bool) {
	key := normalizeRawToolKey(name)
	canonical, ok := r.aliases[key]
	if !ok {
		return rawToolEntry{}, false
	}
	entry, ok := r.entries[canonical]
	return entry, ok
}

func rawToolCommandPath(name string) string {
	return "tool " + strings.ToLower(strings.TrimSpace(name))
}

func rawToolAliases(name string) []string {
	return []string{
		name,
		strings.ToLower(name),
		toKebabCase(name),
	}
}

func normalizeRawToolKey(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func toKebabCase(name string) string {
	var b strings.Builder
	var prevLowerOrDigit bool
	for i, r := range strings.TrimSpace(name) {
		isUpper := r >= 'A' && r <= 'Z'
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if i > 0 && isUpper && prevLowerOrDigit {
			b.WriteByte('-')
		}
		if isUpper {
			r += 'a' - 'A'
		}
		if isLower || isUpper || isDigit {
			b.WriteRune(r)
			prevLowerOrDigit = isLower || isDigit
			continue
		}
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
		prevLowerOrDigit = false
	}
	return strings.Trim(b.String(), "-")
}

func rawToolSchemaJSON(info *schema.ToolInfo) string {
	if info == nil || info.ParamsOneOf == nil {
		return `{"type":"object","properties":{},"required":[]}`
	}
	jsonSchema, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil || jsonSchema == nil {
		return `{"type":"object","properties":{},"required":[]}`
	}
	data, err := json.MarshalIndent(jsonSchema, "", "  ")
	if err != nil {
		return `{"type":"object","properties":{},"required":[]}`
	}
	return string(data)
}

func rawToolCLIHints(name string) string {
	var hints []string
	switch name {
	case "GetStockInfo":
		hints = append(hints,
			"- 推荐盯盘优先调用：`tool GetStockInfo --stock-code sz002335`；该工具包含实时行情和五档盘口概览。",
			"- PowerShell 多股示例：`.\\go-stock-cli.exe tool GetStockInfo --stock-code \"sz300308,sz300502\"`。",
			"- 文件股票池示例：`Get-Content .\\watchlist.txt | .\\go-stock-cli.exe tool GetStockInfo --stock-code -`。",
			"- `stockCode` 可用 `--stockCode`、`--stock-code`、`--stock_code` 或 `--args-json '{\"stockCode\":\"sz002335\"}'` 传入。",
		)
	case "GetStockOrderBook":
		hints = append(hints,
			"- 该工具专查五档盘口；若数据源返回空，CLI 会自动尝试用 `GetStockInfo` 兜底。",
			"- 盯盘优先使用 `tool GetStockInfo`；需要单独盘口字段时再调用本工具。",
			"- PowerShell 多股示例：`.\\go-stock-cli.exe tool GetStockOrderBook --stock-code \"sz300308,sz300502\"`。",
			"- 文件股票池示例：`Get-Content .\\watchlist.txt | .\\go-stock-cli.exe tool GetStockOrderBook --stock-code -`。",
			"- `stockCode` 可用 `--stockCode`、`--stock-code`、`--stock_code` 或 `--args-json '{\"stockCode\":\"sz002335\"}'` 传入。",
		)
	case "GetStockLatestFinance":
		hints = append(hints,
			"- CLI 已对多股票输入做逐只拆分，避免底层单股接口把多代码合并或 panic。",
			"- PowerShell 中多股票建议写成 `--stockCode='sz002335,sz002506'`，或使用 `--args-json`。",
		)
	case "GetEastMoneyKLine", "GetEastMoneyKLineWithMA":
		hints = append(hints,
			"- 复权参数可用 `--adjust qfq`、`--adjust hfq`、`--adjust none`，也兼容原字段 `--adjust-flag`。",
		)
	}
	if hasStockCodeLikeInput(name) {
		hints = append(hints, "- PowerShell 中逗号分隔参数建议加引号，例如 `--stockCode='sh600237,sz002335'`；从管道读取股票代码时使用 `--stock-code -` 或 `--stdin`。")
	}
	if len(hints) == 0 {
		return ""
	}
	return strings.Join(hints, "\n")
}

func rawToolCLIParamAliases(name string) string {
	if !hasStockCodeLikeInput(name) && !hasAdjustFlagInput(name) {
		return ""
	}
	var b strings.Builder
	b.WriteString("| JSON Schema 字段 | 推荐 CLI 参数 | 兼容 CLI 参数 |\n")
	b.WriteString("| --- | --- | --- |\n")
	if hasStockCodeLikeInput(name) {
		b.WriteString("| `stockCode` | `--stock-code` | `--stockCode`, `--stock_code`, `--stockcode`, `--stock-code -`, `--stdin`, `--args-json '{\"stockCode\":\"...\"}'` |\n")
	}
	if hasAdjustFlagInput(name) {
		b.WriteString("| `adjustFlag` | `--adjust` | `--adjust-flag`, `--adjustFlag`, `--adjust_flag`, `--args-json '{\"adjustFlag\":\"qfq\"}'` |\n")
	}
	return b.String()
}

func hasStockCodeLikeInput(name string) bool {
	switch name {
	case "GetStockInfo", "GetStockOrderBook", "GetStockLatestFinance", "GetStockConceptInfo", "GetEastMoneyKLine", "GetEastMoneyKLineWithMA", "GetStockKLine":
		return true
	default:
		return false
	}
}

func hasAdjustFlagInput(name string) bool {
	switch name {
	case "GetEastMoneyKLine", "GetEastMoneyKLineWithMA", "GetStockKLine":
		return true
	default:
		return false
	}
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func isGuardedRawWriteTool(name string) bool {
	return false
}

func isBlockedRawTool(name string) bool {
	_, ok := blockedRawToolReasons[name]
	return ok
}

func blockedRawToolReason(name string) string {
	if reason, ok := blockedRawToolReasons[name]; ok {
		return reason
	}
	return ""
}

var blockedRawToolReasons = map[string]string{
	"SetTradingPrice":              "写入价格提醒，已由 `portfolio position set` 的二次确认流程替代。",
	"SendDingDingMessage":          "外部通知副作用，CLI 默认不开放。",
	"SendToDingDing":               "外部通知副作用，CLI 默认不开放。",
	"CreateAiRecommendStocks":      "AI 推荐写入副作用，CLI 默认不开放。",
	"BatchCreateAiRecommendStocks": "AI 推荐批量写入副作用，CLI 默认不开放。",
	"AiRecommendStocks":            "AI 推荐记录工具已按项目策略禁用，不提供 raw tool 直连。",
	"GetAIAnalysisHistory":         "AI 分析历史工具已按项目策略禁用，不提供 raw tool 直连。",
	"GetAIAnalysisDetail":          "AI 分析详情工具已按项目策略禁用，不提供 raw tool 直连。",
	"GetAIAnalysisContent":         "AI 分析正文工具已按项目策略禁用，不提供 raw tool 直连。",
	"SetFollowedStockPosition":     "受控持仓写入已迁移到 `portfolio position set`，不提供 raw tool 直连。",
}

func read(path, title string, handler Handler) Command {
	return Command{Path: path, Title: title, ReadOnly: true, Handler: handler}
}

func readStructured(path, title string, handler StructuredHandler) Command {
	return Command{Path: path, Title: title, ReadOnly: true, StructuredHandler: handler}
}

func write(path, title string, handler Handler) Command {
	return Command{Path: path, Title: title, ReadOnly: false, Handler: handler}
}

func infoCommand(msg string) Handler {
	return func(context.Context, map[string]any) (string, error) {
		return msg, nil
	}
}

func mapArgs(aliases ...string) func(map[string]any) map[string]any {
	return func(args map[string]any) map[string]any {
		mapped := cloneArgs(args)
		for i := 0; i+1 < len(aliases); i += 2 {
			from, to := aliases[i], aliases[i+1]
			if v, ok := mapped[from]; ok {
				mapped[to] = v
			}
		}
		normalizeCommonArgs(mapped)
		return mapped
	}
}

func withDefaults(mapper func(map[string]any) map[string]any, defaults map[string]any) func(map[string]any) map[string]any {
	return func(args map[string]any) map[string]any {
		mapped := cloneArgs(args)
		if mapper != nil {
			mapped = mapper(mapped)
		}
		for k, v := range defaults {
			if _, ok := mapped[k]; !ok {
				mapped[k] = v
			}
		}
		normalizeCommonArgs(mapped)
		return mapped
	}
}

func normalizeCommonArgs(args map[string]any) {
	aliases := map[string]string{
		"code":                 "stockCode",
		"stock-code":           "stockCode",
		"stock_code":           "stockCode",
		"stockcode":            "stockCode",
		"stock-codes":          "stockCodes",
		"stock_codes":          "stockCodes",
		"stockcodes":           "stockCodes",
		"stock-list":           "stock_list",
		"stocklist":            "stock_list",
		"stock-name":           "stockName",
		"stock_name":           "stockName",
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
		"year-month":           "yearMonth",
		"year_month":           "yearMonth",
		"yearmonth":            "yearMonth",
		"start-date":           "startDate",
		"start_date":           "startDate",
		"startdate":            "startDate",
		"end-date":             "endDate",
		"end_date":             "endDate",
		"enddate":              "endDate",
		"trade-date":           "tradeDate",
		"trade_date":           "tradeDate",
		"tradedate":            "tradeDate",
		"data-type":            "dataType",
		"data_type":            "dataType",
		"datatype":             "dataType",
		"mutual-type":          "mutualType",
		"mutual_type":          "mutualType",
		"mutualtype":           "mutualType",
		"market-type":          "marketType",
		"market_type":          "marketType",
		"markettype":           "marketType",
		"fund-type":            "fundType",
		"fund_type":            "fundType",
		"fundtype":             "fundType",
		"page-size":            "pageSize",
		"page_size":            "pageSize",
		"page-index":           "pageIndex",
		"page_index":           "pageIndex",
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
	for from, to := range aliases {
		if v, ok := args[from]; ok {
			args[to] = v
		}
	}
	normalizeStockCodeArgs(args)
}

func normalizeStockCodeArgs(args map[string]any) {
	for _, key := range []string{"stockCode", "stockCodes", "stock_list"} {
		if v, ok := args[key]; ok {
			args[key] = normalizeStockCodeArgValue(v)
		}
	}
}

func normalizeStockCodeArgValue(value any) any {
	switch typed := value.(type) {
	case []string:
		normalized := make([]string, 0, len(typed))
		for _, code := range typed {
			normalized = append(normalized, normalizeStockCodeToken(code))
		}
		return normalized
	case []any:
		normalized := make([]any, 0, len(typed))
		for _, code := range typed {
			normalized = append(normalized, normalizeStockCodeToken(fmt.Sprint(code)))
		}
		return normalized
	default:
		raw := fmt.Sprint(value)
		parts := strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == '，' || r == ';' || r == '；' || r == ' ' || r == '\t' || r == '\r' || r == '\n'
		})
		if len(parts) <= 1 {
			return normalizeStockCodeToken(raw)
		}
		normalized := make([]string, 0, len(parts))
		for _, part := range parts {
			if code := normalizeStockCodeToken(part); code != "" {
				normalized = append(normalized, code)
			}
		}
		return strings.Join(normalized, ",")
	}
}

func normalizeStockCodeToken(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	lower := strings.ToLower(code)
	if strings.HasSuffix(lower, ".ti") {
		return strings.ToUpper(code)
	}
	if isBareSixDigitCode(lower) ||
		strings.HasSuffix(lower, ".sz") ||
		strings.HasSuffix(lower, ".sh") ||
		strings.HasSuffix(lower, ".bj") ||
		strings.HasSuffix(lower, ".hk") ||
		strings.HasPrefix(lower, "sh") ||
		strings.HasPrefix(lower, "sz") ||
		strings.HasPrefix(lower, "bj") ||
		strings.HasPrefix(lower, "hk") ||
		strings.HasPrefix(lower, "us") ||
		strings.HasPrefix(lower, "gb_") {
		return data.NormalizeFollowedStockCode(code)
	}
	return code
}

func isBareSixDigitCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func requiredString(args map[string]any, names ...string) (string, error) {
	for _, name := range names {
		if v, ok := args[name]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" {
				return s, nil
			}
		}
	}
	return "", fmt.Errorf("missing required argument: %s", strings.Join(names, " or "))
}

func optionalString(args map[string]any, name, fallback string) string {
	if v, ok := args[name]; ok {
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" {
			return s
		}
	}
	return fallback
}

func optionalInt(args map[string]any, name string, fallback int) int {
	if v, ok := args[name]; ok {
		switch t := v.(type) {
		case int:
			return t
		case int64:
			return int(t)
		case float64:
			return int(t)
		case json.Number:
			n, _ := t.Int64()
			return int(n)
		default:
			if n, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(v))); err == nil {
				return n
			}
		}
	}
	return fallback
}

func optionalBool(args map[string]any, name string, fallback bool) bool {
	if v, ok := args[name]; ok {
		switch t := v.(type) {
		case bool:
			return t
		case string:
			switch strings.ToLower(strings.TrimSpace(t)) {
			case "1", "true", "yes", "y", "on":
				return true
			case "0", "false", "no", "n", "off":
				return false
			}
		default:
			s := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
			switch s {
			case "1", "true", "yes", "y", "on":
				return true
			case "0", "false", "no", "n", "off":
				return false
			}
		}
	}
	return fallback
}

func runPortfolioAdd(_ context.Context, args map[string]any) (string, error) {
	normalizeCommonArgs(args)
	code, err := requiredString(args, "stockCode")
	if err != nil {
		return "", err
	}
	if err := rejectIndexForStockCommand(code); err != nil {
		return "", err
	}
	return data.NewStockDataApi().Follow(code), nil
}

func runPortfolioRemove(_ context.Context, args map[string]any) (string, error) {
	normalizeCommonArgs(args)
	code, err := requiredString(args, "stockCode")
	if err != nil {
		return "", err
	}
	if err := rejectIndexForStockCommand(code); err != nil {
		return "", err
	}
	return data.NewStockDataApi().UnFollow(code), nil
}

func runPortfolioPositionSet(r *toolRunner) Handler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		normalizeCommonArgs(args)
		code, err := requiredString(args, "stockCode")
		if err != nil {
			return "", err
		}
		if err := rejectIndexForStockCommand(code); err != nil {
			return "", err
		}
		return r.call("SetFollowedStockPosition", nil)(ctx, args)
	}
}

func rejectIndexForStockCommand(code string) error {
	if data.IsTHSIndexCode(code) {
		return fmt.Errorf("%s 是同花顺指数代码，股票持仓、自选和分组命令不接受 .TI 指数", strings.ToUpper(strings.TrimSpace(code)))
	}
	return nil
}

func runPortfolioGroupList(context.Context, map[string]any) (string, error) {
	groups := data.NewStockGroupApi(db.Dao).GetGroupList()
	type row struct {
		ID   uint   `md:"ID"`
		Name string `md:"分组名称"`
		Sort int    `md:"排序"`
	}
	rows := make([]row, 0, len(groups))
	for _, g := range groups {
		rows = append(rows, row{ID: g.ID, Name: g.Name, Sort: g.Sort})
	}
	if len(rows) == 0 {
		return "暂无分组", nil
	}
	return util.MarkdownTableWithTitle("股票自选分组", rows), nil
}

func runPortfolioGroupAdd(_ context.Context, args map[string]any) (string, error) {
	name, err := requiredString(args, "name")
	if err != nil {
		return "", err
	}
	sortNo := optionalInt(args, "sort", 1)
	if data.NewStockGroupApi(db.Dao).AddGroup(data.Group{Name: name, Sort: sortNo}) {
		return "添加分组成功", nil
	}
	return "添加分组失败", nil
}

func runPortfolioGroupRename(_ context.Context, args map[string]any) (string, error) {
	normalizeCommonArgs(args)
	groupID := optionalInt(args, "groupId", 0)
	if groupID <= 0 {
		return "", fmt.Errorf("groupId must be greater than 0")
	}
	name, err := requiredString(args, "name", "newName")
	if err != nil {
		return "", err
	}
	if data.NewStockGroupApi(db.Dao).UpdateGroup(groupID, name) {
		return "修改分组成功", nil
	}
	return "修改分组失败", nil
}

func runPortfolioGroupAssign(_ context.Context, args map[string]any) (string, error) {
	normalizeCommonArgs(args)
	code, err := requiredString(args, "stockCode")
	if err != nil {
		return "", err
	}
	if err := rejectIndexForStockCommand(code); err != nil {
		return "", err
	}
	groupID := optionalInt(args, "groupId", 0)
	if groupID <= 0 {
		return "", fmt.Errorf("groupId must be greater than 0")
	}
	if data.NewStockGroupApi(db.Dao).AddStockGroup(groupID, data.NormalizeFollowedStockCode(code)) {
		return "设置分组成功", nil
	}
	return "设置分组失败", nil
}

func runPortfolioGroupRemove(_ context.Context, args map[string]any) (string, error) {
	normalizeCommonArgs(args)
	code, err := requiredString(args, "stockCode")
	if err != nil {
		return "", err
	}
	if err := rejectIndexForStockCommand(code); err != nil {
		return "", err
	}
	groupID := optionalInt(args, "groupId", 0)
	if groupID <= 0 {
		return "", fmt.Errorf("groupId must be greater than 0")
	}
	name := optionalString(args, "stockName", "")
	if data.NewStockGroupApi(db.Dao).RemoveStockGroup(data.NormalizeFollowedStockCode(code), name, groupID) {
		return "移出分组成功", nil
	}
	return "移出分组失败", nil
}

func runMarketNews(ctx context.Context, args map[string]any) (string, error) {
	r := newToolRunner()
	out, err := r.call("GetMarketData", nil)(ctx, args)
	if err != nil {
		return "", err
	}
	return out + "\n\n## 市场快讯子功能\n\n- 主要股指\n- 涨跌家数比\n- 涨跌停家数比\n- 当日异动次数最多的概念\n- 查看热词\n- 按天涨跌/涨跌停分析\n- 历史异动分析\n- 异动排行\n- 利好/利空排行\n- 快讯列表\n", nil
}

func runMajorIndex(ctx context.Context, args map[string]any) (string, error) {
	name := optionalString(args, "name", "")
	code := optionalString(args, "code", "")
	if name == "" && code == "" {
		r := newToolRunner()
		out, err := r.call("GetMarketData", nil)(ctx, args)
		if err != nil {
			return "", err
		}
		names := make([]string, 0, len(majorIndexNodes()))
		for _, node := range majorIndexNodes() {
			if node.Example != "" {
				names = append(names, fmt.Sprintf("- %s：`%s`", node.Label, node.Example))
				continue
			}
			names = append(names, "- "+node.Label)
		}
		return out + "\n\n## 重大指数单独查询\n\n传入 `name` 或 `code` 可查询指定指数 K 线。支持：\n\n" + strings.Join(names, "\n"), nil
	}
	spec, ok := majorIndexSpecByName(name)
	if code != "" {
		if byCode, found := majorIndexSpecByCode(code); found {
			spec = byCode
			ok = true
		} else {
			spec = majorIndexSpec{Name: code, Code: code}
			ok = true
		}
	}
	if !ok || spec.Code == "" {
		return "", fmt.Errorf("unknown major index %q; pass code explicitly", name)
	}
	if spec.LegacyCode != "" {
		return runMajorIndexWithFallback(spec, optionalInt(args, "limit", 60)), nil
	}
	r := newToolRunner()
	return r.call("GetEastMoneyKLineWithMA", withDefaults(nil, map[string]any{
		"stockCode": spec.Code,
		"kLineType": "day",
		"limit":     optionalInt(args, "limit", 60),
	}))(ctx, args)
}

func runMarketAnnouncement(r *toolRunner) Handler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		normalizeCommonArgs(args)
		if stockList := optionalString(args, "stock_list", ""); stockList != "" {
			mapped := cloneArgs(args)
			mapped["stock_list"] = stockList
			return r.call("StockNotice", nil)(ctx, mapped)
		}

		stockCodes := optionalString(args, "stockCodes", optionalString(args, "stockCode", ""))
		if stockCodes != "" {
			mapped := cloneArgs(args)
			mapped["stockCodes"] = stockCodes
			return r.call("GetStockNotice", nil)(ctx, mapped)
		}

		mapped := cloneArgs(args)
		mapped["stock_list"] = ""
		out, err := r.call("StockNotice", nil)(ctx, mapped)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(out) == "" {
			return "暂无公司公告数据。查询单股公告可用 `market announcement --stock-code 600237` 或 `portfolio view notice --stock-code 600237`。", nil
		}
		return out, nil
	}
}

func runInvestCalendar(r *toolRunner) Handler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		normalizeCommonArgs(args)
		out, err := r.call("GetInvestCalendar", nil)(ctx, args)
		if err == nil && strings.TrimSpace(out) != "" && !strings.Contains(out, "暂无投资日历数据") {
			return out, nil
		}

		var b strings.Builder
		if strings.TrimSpace(out) != "" {
			b.WriteString(strings.TrimSpace(out))
			b.WriteString("\n\n")
		}
		if err != nil {
			b.WriteString("九阳公社投资日历源当前异常，已尝试华尔街见闻财经日历兜底。\n\n")
		} else {
			b.WriteString("投资日历源当前无数据，已尝试华尔街见闻财经日历兜底。\n\n")
		}

		fallback, fallbackErr := r.call("GetWallstreetcnCalendar", nil)(ctx, args)
		if fallbackErr == nil && strings.TrimSpace(fallback) != "" {
			b.WriteString("## 全球财经日历兜底\n\n")
			b.WriteString(strings.TrimSpace(fallback))
			return b.String(), nil
		}
		if err != nil {
			b.WriteString("兜底财经日历也暂不可用：")
			b.WriteString(err.Error())
			if fallbackErr != nil {
				b.WriteString("；")
				b.WriteString(fallbackErr.Error())
			}
			return b.String(), nil
		}
		return out, nil
	}
}

func runResearchUplimit(r *toolRunner) Handler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		out, err := r.call("GetUplimitLadder", nil)(ctx, args)
		if err != nil {
			return "", err
		}
		if strings.Contains(out, "涨停总数: 0") || strings.Contains(out, "涨停总数：0") || strings.Contains(out, "暂无") {
			marketOut, marketErr := r.call("GetMarketData", nil)(ctx, args)
			var b strings.Builder
			b.WriteString(strings.TrimSpace(out))
			b.WriteString("\n\n## 数据一致性提示\n\n")
			b.WriteString("涨停梯队接口当前返回为空或 0。盯盘时请交叉验证 `market news` / `tool GetMarketData` 的涨停家数、跌停家数和涨跌分布，不要单独依赖涨停梯队判断市场情绪。")
			if marketErr == nil && strings.TrimSpace(marketOut) != "" {
				b.WriteString("\n\n## 市场总览交叉验证\n\n")
				b.WriteString(marketOut)
			}
			return b.String(), nil
		}
		return out, nil
	}
}

type majorIndexSpec struct {
	Name         string
	Aliases      []string
	Code         string
	LegacyCode   string
	UseLegacyCLI bool
	Note         string
}

func majorIndexCatalog() []majorIndexSpec {
	return []majorIndexSpec{
		{Name: "上证指数", Code: "000001.SH"},
		{Name: "深证指数", Aliases: []string{"深证成指"}, Code: "399001.SZ"},
		{Name: "创业板指", Code: "399006.SZ"},
		{Name: "恒生指数", Code: "100.HSI", LegacyCode: "hkHSI", UseLegacyCLI: true},
		{Name: "道琼斯", Code: "100.DJIA", LegacyCode: "us.DJI", UseLegacyCLI: true},
		{Name: "标普500", Aliases: []string{"标普 500"}, Code: "100.SPX", LegacyCode: "us.INX", UseLegacyCLI: true},
		{Name: "纳斯达克", Code: "100.NDX", LegacyCode: "us.IXIC", UseLegacyCLI: true},
		{Name: "沪深300", Code: "000300.SH"},
		{Name: "上证50", Code: "000016.SH"},
		{Name: "中证A500", Code: "000510.SH"},
		{Name: "中证1000", Code: "000852.SH"},
		{Name: "科创50", Code: "000688.SH"},
		{Name: "科创芯片", Code: "000685.SH"},
		{Name: "证券龙头", Code: "399437.SZ"},
		{Name: "高端装备", Code: "930599.CSI"},
		{Name: "微盘股", Aliases: []string{"微盘股指数"}, Code: "883418.TI"},
		{Name: "中证银行", Code: "399986.SZ"},
		{Name: "上证医药", Code: "000037.SH"},
		{Name: "中证白酒", Code: "399997.SZ"},
		{Name: "富时中国三倍做多", Code: "USYINN.AM", LegacyCode: "usYINN.AM", UseLegacyCLI: true},
		{Name: "VIX恐慌指数", Code: "USUVXY.AM", LegacyCode: "usUVXY.AM", UseLegacyCLI: true, Note: "UVXY代理"},
	}
}

func majorIndexCode(name string) string {
	spec, ok := majorIndexSpecByName(name)
	if !ok {
		return ""
	}
	return spec.Code
}

func majorIndexSpecByName(name string) (majorIndexSpec, bool) {
	normalized := strings.TrimSpace(name)
	for _, spec := range majorIndexCatalog() {
		if spec.Name == normalized {
			return spec, true
		}
		for _, alias := range spec.Aliases {
			if alias == normalized {
				return spec, true
			}
		}
	}
	return majorIndexSpec{}, false
}

func majorIndexSpecByCode(code string) (majorIndexSpec, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	for _, spec := range majorIndexCatalog() {
		if strings.ToUpper(spec.Code) == normalized || strings.ToUpper(spec.LegacyCode) == normalized {
			return spec, true
		}
	}
	return majorIndexSpec{}, false
}

func runMajorIndexWithFallback(spec majorIndexSpec, limit int) string {
	if limit <= 0 {
		limit = 60
	}
	attempts := []string{}
	if strings.TrimSpace(spec.Code) != "" {
		result := data.FetchKLineWithFallback(spec.Code, spec.Name, "101", limit, "", "none")
		attempts = append(attempts, majorIndexAttemptLabel(spec.Code, result))
		if result != nil && result.Data != nil && len(*result.Data) > 0 {
			return formatMajorIndexKLine(spec, spec.Code, result.Source, *result.Data)
		}
	}
	if strings.TrimSpace(spec.LegacyCode) != "" {
		list := data.NewStockDataApi().GetHK_KLineData(spec.LegacyCode, "day", int64(limit))
		attempts = append(attempts, spec.LegacyCode+"（legacy 腾讯源）")
		if list != nil && len(*list) > 0 {
			return formatMajorIndexKLine(spec, spec.LegacyCode, "tencent-legacy", *list)
		}
	}
	return fmt.Sprintf("%s：未获取到 K 线数据。已尝试：%s。若 GUI 同样为空，通常是东财/腾讯 HTTPS 行情源当前不可达或返回为空。", spec.Name, strings.Join(attempts, "；"))
}

func majorIndexAttemptLabel(code string, result *data.KLineSourceResult) string {
	source := "统一回退链"
	if result != nil && strings.TrimSpace(result.Source) != "" {
		source += "/" + result.Source
	}
	return code + "（" + source + "）"
}

func formatMajorIndexKLine(spec majorIndexSpec, code string, source string, list []data.KLineData) string {
	type row struct {
		Date   string `md:"日期"`
		Open   string `md:"开盘价"`
		Close  string `md:"收盘价"`
		High   string `md:"最高价"`
		Low    string `md:"最低价"`
		Volume string `md:"成交量"`
	}
	rows := make([]row, 0, len(list))
	for _, item := range list {
		rows = append(rows, row{
			Date:   item.Day,
			Open:   item.Open,
			Close:  item.Close,
			High:   item.High,
			Low:    item.Low,
			Volume: item.Volume,
		})
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "unknown"
	}
	title := fmt.Sprintf("%s %s K线（共 %d 条，数据源：%s）", spec.Name, code, len(rows), source)
	if spec.Note != "" {
		title += "（" + spec.Note + "）"
	}
	out := util.MarkdownTableWithTitle(title, rows)
	if summary := buildKLineSummaryFromList(code, list, source, ""); summary != "" {
		out += "\n\n" + summary
	}
	return out
}

func runConceptList(context.Context, map[string]any) (string, error) {
	codes := data.NewConceptFundFlowApi().GetAllConceptCodes()
	if len(codes) == 0 {
		return "暂无概念代码数据", nil
	}
	type row struct {
		Rank int    `md:"序号"`
		Code string `md:"概念代码"`
		Name string `md:"概念名称"`
	}
	rows := make([]row, 0, len(codes))
	for i, item := range codes {
		rows = append(rows, row{Rank: i + 1, Code: item["code"], Name: item["name"]})
	}
	return util.MarkdownTableWithTitle("概念资金流向可用概念代码", rows), nil
}

func runConceptLatest(_ context.Context, args map[string]any) (string, error) {
	list := data.NewConceptFundFlowApi().GetConceptFundFlowTopList(optionalInt(args, "topN", 20))
	if len(list) == 0 {
		return "暂无概念资金流向快照数据。该命令只读取本地数据库，不触发采集写入。", nil
	}
	return util.MarkdownTableWithTitle("最新概念资金流向排名", conceptFundFlowRows(list)), nil
}

func runConceptDate(_ context.Context, args map[string]any) (string, error) {
	date, err := requiredString(args, "date")
	if err != nil {
		return "", err
	}
	list := data.NewConceptFundFlowApi().GetConceptFundFlowTopListByDate(date, optionalInt(args, "topN", 20))
	if len(list) == 0 {
		return "暂无" + date + "概念资金流向快照数据。该命令只读取本地数据库，不触发采集写入。", nil
	}
	return util.MarkdownTableWithTitle(date+" 概念资金流向排名", conceptFundFlowRows(list)), nil
}

func runConceptTrend(_ context.Context, args map[string]any) (string, error) {
	code, err := requiredString(args, "code")
	if err != nil {
		return "", err
	}
	date := optionalString(args, "date", "")
	var points []models.ConceptFundFlowPoint
	if date != "" {
		points = data.NewConceptFundFlowApi().GetConceptFundFlowListByDate(code, date)
	} else {
		points = data.NewConceptFundFlowApi().GetConceptFundFlowList(code, optionalInt(args, "limit", 240))
	}
	if len(points) == 0 {
		return "暂无概念 " + code + " 的资金流向历史数据。该命令只读取本地数据库，不触发采集写入。", nil
	}
	return util.MarkdownTableWithTitle(code+" 概念资金流向历史", conceptFundFlowPointRows(points)), nil
}

type conceptFundFlowRow struct {
	Rank      int    `md:"排名"`
	Code      string `md:"概念代码"`
	Name      string `md:"概念名称"`
	NetInflow int64  `md:"主力净流入(元)"`
	SnapTime  string `md:"快照时间"`
}

func conceptFundFlowRows(list []models.ConceptFundFlow) []conceptFundFlowRow {
	rows := make([]conceptFundFlowRow, 0, len(list))
	for i, item := range list {
		rows = append(rows, conceptFundFlowRow{Rank: i + 1, Code: item.Code, Name: item.Name, NetInflow: item.NetInflow, SnapTime: item.SnapTime})
	}
	return rows
}

type conceptFundFlowPointRow struct {
	SnapTime  string `md:"快照时间"`
	NetInflow int64  `md:"主力净流入(元)"`
}

func conceptFundFlowPointRows(points []models.ConceptFundFlowPoint) []conceptFundFlowPointRow {
	rows := make([]conceptFundFlowPointRow, 0, len(points))
	for _, item := range points {
		rows = append(rows, conceptFundFlowPointRow{SnapTime: item.SnapTime, NetInflow: item.NetInflow})
	}
	return rows
}

func runHotTopic(context.Context, map[string]any) (string, error) {
	return "前端“当前热门 > 热门话题”对应 App/Wails 的 HotTopic(size)，底层是东方财富股吧话题接口；当前 CLI 先保留菜单入口。若需要近似热点事件，可调用 `market hot timeline`。", nil
}

func runKlineRecent(context.Context, map[string]any) (string, error) {
	return "K线最近查看保存在前端 localStorage (`kline-recent-stocks`)；当前 CLI 无法读取 GUI WebView localStorage。", nil
}

func runKlineShow(r *toolRunner) Handler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		normalizeCommonArgs(args)
		code, err := requiredString(args, "stockCode")
		if err != nil {
			return "", err
		}
		mapped := cloneArgs(args)
		mapped["stockCode"] = code
		kLineType := optionalString(args, "kLineType", optionalString(args, "period", "day"))
		adjustFlag, err := kLineAdjustFlag(args, kLineType)
		if err != nil {
			return "", err
		}
		if data.IsTHSIndexCode(code) {
			if raw := strings.TrimSpace(optionalString(args, "adjustFlag", "")); raw != "" && !strings.EqualFold(raw, "none") && raw != "0" {
				return "", fmt.Errorf(".TI 指数没有复权语义；请省略 --adjust 或使用 --adjust none")
			}
			adjustFlag = "none"
		}
		mapped["kLineType"] = kLineType
		mapped["limit"] = optionalInt(args, "limit", 120)
		mapped["adjustFlag"] = adjustFlag
		if _, ok := mapped["maPeriods"]; !ok {
			mapped["maPeriods"] = "5,10,20,60,120"
		}
		out, err := r.call("GetEastMoneyKLineWithMA", nil)(ctx, mapped)
		if err != nil {
			return "", err
		}
		indicators := optionalString(args, "indicators", "")
		if indicators != "" {
			out += "\n\n## 请求的技术指标\n\n" + indicators + "\n"
		}
		if hasAny(args, "entryPrice", "stopLossPrice", "takeProfitPrice") {
			out += "\n\n## 价位线\n\n"
			out += fmt.Sprintf("- 开仓价：%v\n- 止损价：%v\n- 止盈价：%v\n", args["entryPrice"], args["stopLossPrice"], args["takeProfitPrice"])
		}
		if summary, ok := buildKLineSummary(code, fmt.Sprint(mapped["kLineType"]), fmt.Sprint(mapped["maPeriods"]), optionalInt(args, "limit", 120), adjustFlag); ok {
			out += "\n\n" + summary
		}
		return out, nil
	}
}

func buildKLineSummary(code, kLineType, maPeriods string, limit int, adjustFlag string) (string, bool) {
	periods := parseKLineSummaryPeriods(maPeriods)
	if len(periods) == 0 {
		periods = []int{5, 10, 20, 60}
	}
	fetchLimit := kLineSummaryFetchLimit(limit, periods)
	result := data.FetchKLineWithFallback(code, "", data.NormalizeKLineType(kLineType), fetchLimit, "", adjustFlag)
	if result.Data == nil || len(*result.Data) == 0 {
		return "", false
	}
	summary := buildKLineSummaryText(code, *result.Data, periods, result.Source, adjustFlag)
	if summary == "" {
		return "", false
	}
	return summary, true
}

func buildKLineSummaryFromList(code string, list []data.KLineData, source string, adjustFlag string) string {
	return buildKLineSummaryText(code, list, []int{5, 10, 20, 60}, source, adjustFlag)
}

func buildKLineSummaryText(code string, list []data.KLineData, periods []int, source string, adjustFlag string) string {
	if len(list) == 0 {
		return ""
	}
	latest := list[len(list)-1]
	current, ok := parseKLineFloat(latest.Close)
	if !ok || current <= 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## K线摘要\n\n")
	b.WriteString(fmt.Sprintf("- 最新收盘价：%.2f（%s）\n", current, cliValueOrDash(latest.Day)))
	for _, period := range periods {
		ma, ok := averageClose(list, period)
		if !ok || period > 60 {
			continue
		}
		b.WriteString(fmt.Sprintf("- MA%d：%.2f，当前价%s MA%d %.2f%%\n", period, ma, priceRelation(current, ma), period, percentDistance(current, ma)))
	}
	if pct, ok := kLineReturnPercent(list, 5); ok {
		b.WriteString(fmt.Sprintf("- 近5根涨幅：%.2f%%\n", pct))
	}
	if pct, ok := kLineReturnPercent(list, 20); ok {
		b.WriteString(fmt.Sprintf("- 近20根涨幅：%.2f%%\n", pct))
	}
	if text, ok := kLineVolumeSummary(list); ok {
		b.WriteString("- 量能：" + text + "\n")
	}
	if strings.TrimSpace(source) != "" {
		b.WriteString("- 摘要数据源：" + source + "\n")
	}
	if adjustFlag != "" {
		b.WriteString("- 复权：" + kLineAdjustLabel(adjustFlag) + "\n")
	}
	return strings.TrimSpace(b.String())
}

func kLineAdjustFlag(args map[string]any, kLineType string) (string, error) {
	raw := strings.ToLower(strings.TrimSpace(optionalString(args, "adjustFlag", "")))
	kType := data.NormalizeKLineType(kLineType)
	if raw == "" {
		if isDailyLikeKLineType(kType) {
			return "qfq", nil
		}
		return "", nil
	}
	switch raw {
	case "qfq", "hfq", "none", "0":
		if isDailyLikeKLineType(kType) {
			return raw, nil
		}
		return "", nil
	default:
		return "", fmt.Errorf("adjust must be one of: qfq, hfq, none")
	}
}

func isDailyLikeKLineType(kType string) bool {
	switch data.NormalizeKLineType(kType) {
	case "101", "102", "103", "104", "106":
		return true
	default:
		return false
	}
}

func kLineAdjustLabel(adjustFlag string) string {
	switch strings.ToLower(strings.TrimSpace(adjustFlag)) {
	case "qfq":
		return "前复权(qfq)"
	case "hfq":
		return "后复权(hfq)"
	case "none", "0":
		return "不复权(none)"
	default:
		return adjustFlag
	}
}

func parseKLineSummaryPeriods(value string) []int {
	var periods []int
	for _, part := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == ' ' || r == '\t' || r == '\n'
	}) {
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(part), "MA")))
		if err == nil && n > 0 {
			periods = append(periods, n)
		}
	}
	return periods
}

func kLineSummaryFetchLimit(limit int, periods []int) int {
	if limit < 80 {
		limit = 80
	}
	for _, period := range periods {
		if period+20 > limit {
			limit = period + 20
		}
	}
	return limit
}

func averageClose(list []data.KLineData, period int) (float64, bool) {
	if period <= 0 || len(list) < period {
		return 0, false
	}
	start := len(list) - period
	var sum float64
	for _, item := range list[start:] {
		value, ok := parseKLineFloat(item.Close)
		if !ok {
			return 0, false
		}
		sum += value
	}
	return sum / float64(period), true
}

func kLineReturnPercent(list []data.KLineData, bars int) (float64, bool) {
	if bars <= 0 || len(list) <= bars {
		return 0, false
	}
	latest, ok := parseKLineFloat(list[len(list)-1].Close)
	if !ok || latest <= 0 {
		return 0, false
	}
	base, ok := parseKLineFloat(list[len(list)-1-bars].Close)
	if !ok || base <= 0 {
		return 0, false
	}
	return (latest - base) / base * 100, true
}

func kLineVolumeSummary(list []data.KLineData) (string, bool) {
	if len(list) < 2 {
		return "", false
	}
	latest, ok := parseKLineFloat(list[len(list)-1].Volume)
	if !ok || latest <= 0 {
		return "", false
	}
	count := 5
	if len(list)-1 < count {
		count = len(list) - 1
	}
	var sum float64
	for _, item := range list[len(list)-1-count : len(list)-1] {
		value, ok := parseKLineFloat(item.Volume)
		if !ok {
			return "", false
		}
		sum += value
	}
	avg := sum / float64(count)
	if avg <= 0 {
		return "", false
	}
	ratio := latest / avg
	state := "接近前5根均量"
	if ratio >= 1.2 {
		state = "放量"
	} else if ratio <= 0.8 {
		state = "缩量"
	}
	return fmt.Sprintf("%s，最新成交量为前%d根均量的 %.2f 倍", state, count, ratio), true
}

func parseKLineFloat(value string) (float64, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" || value == "-" {
		return 0, false
	}
	f, err := strconv.ParseFloat(value, 64)
	return f, err == nil
}

func priceRelation(price, ma float64) string {
	if ma <= 0 {
		return "相对"
	}
	if price > ma {
		return "高于"
	}
	if price < ma {
		return "低于"
	}
	return "等于"
}

func percentDistance(price, base float64) float64 {
	if base <= 0 {
		return 0
	}
	return (price - base) / base * 100
}

func cliValueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func hasAny(args map[string]any, names ...string) bool {
	for _, name := range names {
		if _, ok := args[name]; ok {
			return true
		}
	}
	return false
}

func runFundFollow(_ context.Context, args map[string]any) (string, error) {
	action := optionalString(args, "action", "list")
	api := data.NewFundApi()
	switch action {
	case "list":
		result := api.GetFollowedFundPaged(optionalInt(args, "pageIndex", 1), optionalInt(args, "pageSize", 20), optionalString(args, "keyword", ""))
		return markdownJSONTitle("基金自选", result.Items)
	case "add":
		code, err := requiredString(args, "fundCode", "code")
		if err != nil {
			return "", err
		}
		return api.FollowFund(code), nil
	case "remove":
		code, err := requiredString(args, "fundCode", "code")
		if err != nil {
			return "", err
		}
		return api.UnFollowFund(code), nil
	default:
		return "", fmt.Errorf("unknown fund follow action %q; use list, add, or remove", action)
	}
}

func runFundRanking(_ context.Context, args map[string]any) (string, error) {
	result, err := data.NewFundApi().GetFundRanking(
		optionalString(args, "marketType", "kf"),
		optionalString(args, "fundType", "all"),
		optionalString(args, "sortField", "jnzf"),
		optionalString(args, "sortOrder", "desc"),
		optionalInt(args, "pageIndex", 1),
		optionalInt(args, "pageSize", 20),
	)
	if err != nil {
		return "", err
	}
	if result == nil || len(result.Items) == 0 {
		return "暂无基金排行数据", nil
	}
	return util.MarkdownTableWithTitle("基金排行", result.Items), nil
}

func runResearchRecommend(ctx context.Context, args map[string]any) (string, error) {
	defaults := map[string]any{"page": 1, "pageSize": 20}
	for k, v := range defaults {
		if _, ok := args[k]; !ok {
			args[k] = v
		}
	}
	r := newToolRunner()
	return r.call("AiRecommendStocks", nil)(ctx, args)
}

func markdownJSONTitle(title string, v any) (string, error) {
	dataBytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	if string(dataBytes) == "null" || string(dataBytes) == "[]" {
		return "暂无" + title + "数据", nil
	}
	md, err := data.JSONToMarkdownTable(dataBytes)
	if err != nil {
		return "", err
	}
	return "# " + title + "\n\n" + md, nil
}

func SupportedCommandPaths() []string {
	r, err := NewRunner()
	if err != nil {
		return nil
	}
	paths := make([]string, 0, len(r.commands))
	for path := range r.commands {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
