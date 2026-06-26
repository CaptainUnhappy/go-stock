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
		write("portfolio position set", "设置持仓提醒", r.call("SetFollowedStockPosition", nil)),
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
		read("portfolio view notice", "查看个股公告", r.call("GetStockNotice", mapArgs("code", "stockCode"))),
		read("portfolio view report", "查看个股研报", r.call("GetStockResearchReport", mapArgs("code", "stockCode"))),

		read("market news", "市场快讯", runMarketNews),
		read("market global-index", "全球股指", r.call("GlobalStockIndexesReadable", nil)),
		read("market major-index", "重大指数", runMajorIndex),
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
		read("market announcement", "公司公告", r.call("GetStockNotice", nil)),
		read("market industry-research", "行业研究", r.call("IndustryResearch", nil)),
		read("market hot global", "当前热门-全球", r.call("GetGlobalMarketStatus", nil)),
		read("market hot cn", "当前热门-沪深", r.call("GetMarketData", nil)),
		read("market hot hk", "当前热门-港股", r.call("GlobalStockIndexesReadable", nil)),
		read("market hot us", "当前热门-美股", r.call("GlobalStockIndexesReadable", nil)),
		read("market hot topic", "当前热门-热门话题", runHotTopic),
		read("market hot timeline", "当前热门-重大事件时间轴", r.call("GetHotEventList", nil)),
		read("market hot calendar", "当前热门-财经日历", r.call("GetGlobalMarketStatus", nil)),

		read("kline search", "K线标的搜索", r.call("QueryStockCodeInfo", mapArgs("keyword", "searchWord"))),
		read("kline recent", "K线最近查看", runKlineRecent),
		read("kline show", "K线展示", runKlineShow(r)),

		read("fund follow", "基金自选", runFundFollow),
		read("fund ranking", "基金排行", runFundRanking),
		read("fund search", "基金搜索", r.call("SearchFund", mapArgs("keyword", "keyword"))),
		read("fund info", "基金详情", r.call("GetFundInfo", mapArgs("code", "fundCode"))),
		read("fund kline", "基金K线", r.call("GetFundKLine", mapArgs("code", "fundCode"))),
		read("fund nav", "基金净值", r.call("GetFundHistoryNetValue", mapArgs("code", "fundCode"))),
		read("fund holdings", "基金持仓", r.call("GetFundTop10Holdings", mapArgs("code", "fundCode"))),

		read("research ai-report", "AI分析报告", r.call("GetAIAnalysisHistory", nil)),
		read("research recommend", "股票推荐记录", runResearchRecommend),
		read("research changes", "异动监控", r.call("GetStockChanges", nil)),
		read("research uplimit", "涨停梯队", runResearchUplimit(r)),
		read("research prompt-template", "提示词模板", infoCommand("提示词模板属于 GUI 本地管理功能；当前 CLI 不开放模板写入。")),
		read("research prompt-plaza", "提示词广场", infoCommand("提示词广场依赖 go-stock 应用内服务；当前 CLI 只保留菜单对齐入口。")),
		read("research qa-plaza", "问答广场", infoCommand("问答广场依赖 go-stock 应用内服务；当前 CLI 只保留菜单对齐入口。")),
		read("research pattern-screen", "形态选股", r.call("FilterStocks", nil)),
		read("research indicator-screen", "指标选股", r.call("SearchStockByIndicators", mapArgs("query", "query"))),
		read("research cron-task", "定时任务", infoCommand("定时任务含本地调度副作用；当前 CLI 只保留只读菜单入口。")),
		read("research trade-log", "交易日志", infoCommand("交易日志含用户交易记录写入；当前 CLI 只保留菜单入口，后续可单独设计确认流。")),
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
	case "GetStockConceptInfo",
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
			"- `stockCode` 可用 `--stockCode`、`--stock-code`、`--stock_code` 或 `--args-json '{\"stockCode\":\"sz002335\"}'` 传入。",
		)
	case "GetStockOrderBook":
		hints = append(hints,
			"- 该工具专查五档盘口；若数据源返回空，CLI 会自动尝试用 `GetStockInfo` 兜底。",
			"- 盯盘优先使用 `tool GetStockInfo`；需要单独盘口字段时再调用本工具。",
			"- `stockCode` 可用 `--stockCode`、`--stock-code`、`--stock_code` 或 `--args-json '{\"stockCode\":\"sz002335\"}'` 传入。",
		)
	case "GetStockLatestFinance":
		hints = append(hints,
			"- CLI 已对多股票输入做逐只拆分，避免底层单股接口把多代码合并或 panic。",
			"- PowerShell 中多股票建议写成 `--stockCode='sz002335,sz002506'`，或使用 `--args-json`。",
		)
	}
	if hasStockCodeLikeInput(name) {
		hints = append(hints, "- PowerShell 中逗号分隔参数建议加引号，例如 `--stockCode='sh600237,sz002335'`。")
	}
	if len(hints) == 0 {
		return ""
	}
	return strings.Join(hints, "\n")
}

func hasStockCodeLikeInput(name string) bool {
	switch name {
	case "GetStockInfo", "GetStockOrderBook", "GetStockLatestFinance", "GetStockConceptInfo", "GetEastMoneyKLine", "GetEastMoneyKLineWithMA", "GetStockKLine":
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
	"AiRecommendStocks":            "旧对外 MCP denylist 工具；CLI 保留 `research recommend` 作为业务菜单入口，不提供 raw tool 直连。",
	"SetFollowedStockPosition":     "受控持仓写入已迁移到 `portfolio position set`，不提供 raw tool 直连。",
}

func read(path, title string, handler Handler) Command {
	return Command{Path: path, Title: title, ReadOnly: true, Handler: handler}
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
	}
	for from, to := range aliases {
		if v, ok := args[from]; ok {
			args[to] = v
		}
	}
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
	return data.NewStockDataApi().Follow(code), nil
}

func runPortfolioRemove(_ context.Context, args map[string]any) (string, error) {
	normalizeCommonArgs(args)
	code, err := requiredString(args, "stockCode")
	if err != nil {
		return "", err
	}
	return data.NewStockDataApi().UnFollow(code), nil
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
			names = append(names, "- "+node.Label)
		}
		return out + "\n\n## 重大指数单独查询\n\n传入 `name` 或 `code` 可查询指定指数 K 线。支持：\n\n" + strings.Join(names, "\n"), nil
	}
	if code == "" {
		code = majorIndexCode(name)
	}
	if code == "" {
		return "", fmt.Errorf("unknown major index %q; pass code explicitly", name)
	}
	r := newToolRunner()
	return r.call("GetEastMoneyKLineWithMA", withDefaults(nil, map[string]any{
		"stockCode": code,
		"kLineType": "day",
		"limit":     optionalInt(args, "limit", 60),
	}))(ctx, args)
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

func majorIndexCode(name string) string {
	switch strings.TrimSpace(name) {
	case "上证指数":
		return "000001.SH"
	case "深证指数", "深证成指":
		return "399001.SZ"
	case "创业板指":
		return "399006.SZ"
	case "恒生指数":
		return "100.HSI"
	case "道琼斯":
		return "100.DJIA"
	case "标普500", "标普 500":
		return "100.SPX"
	case "纳斯达克":
		return "100.NDX"
	case "沪深300":
		return "000300.SH"
	case "上证50":
		return "000016.SH"
	case "中证A500":
		return "000510.SH"
	case "中证1000":
		return "000852.SH"
	case "科创50":
		return "000688.SH"
	case "科创芯片":
		return "000685.SH"
	case "证券龙头":
		return "399437.SZ"
	case "高端装备":
		return "399437.SZ"
	case "中证银行":
		return "399986.SZ"
	case "上证医药":
		return "000037.SH"
	case "中证白酒":
		return "399997.SZ"
	case "富时中国三倍做多":
		return "USYINN.AM"
	case "VIX恐慌指数":
		return "USUVXY.AM"
	default:
		return ""
	}
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
		mapped["kLineType"] = optionalString(args, "kLineType", optionalString(args, "period", "day"))
		mapped["limit"] = optionalInt(args, "limit", 120)
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
		return out, nil
	}
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
