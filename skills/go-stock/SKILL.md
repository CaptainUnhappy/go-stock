---
name: go-stock
description: Use when an agent needs to work with local go-stock financial data through go-stock-cli. Default usage is the local CLI with GUI-aligned command paths and the migrated raw tool layer for Chinese stock, market行情, K线, 基金, 自选持仓, 研报, 公告, 资金流, 条件选股, 涨停热点, and research workflows.
---

# Go Stock CLI

## 默认入口

当前长期主入口是本地 CLI，不是 MCP：

1. `.\scripts\go-stock-cli.ps1 help`：读取 GUI 对齐的 CLI 功能树和命令路径；Codex/沙箱环境优先用这个脚本。
2. `.\scripts\go-stock-cli.ps1 <commandPath> ...`：调用具体功能。
3. GUI 主树没有覆盖但归档工具层有的能力，用 `tool list`、`tool info`、`tool <原工具名>`。

旧的原始对外 MCP 工具已经迁移到 CLI 归档工具层：先用 `tool list` 查看清单，用 `tool info --name 工具名` 查看参数，再用 `tool 工具名` 调用。例如 `tool GetStockOrderBook`。项目本地对外 MCP 服务已归档，不再作为新集成目标。

## 快速使用

在仓库根目录运行：

```powershell
.\scripts\go-stock-cli.ps1 help
.\scripts\go-stock-cli.ps1 market news
.\scripts\go-stock-cli.ps1 market major-index
.\scripts\go-stock-cli.ps1 market major-index --name 上证指数
.\scripts\go-stock-cli.ps1 market industry-rank concept-money --sort netamount --limit 20
.\scripts\go-stock-cli.ps1 market money-flow stock --sort r0_net --limit 20
.\scripts\go-stock-cli.ps1 kline show --stock-code 002335 --k-line-type day --limit 120
.\scripts\go-stock-cli.ps1 portfolio list
.\scripts\go-stock-cli.ps1 portfolio group rename --group-id 1 --new-name 短线观察
.\scripts\go-stock-cli.ps1 portfolio position set --stock-code 600237 --cost-price 12.56 --volume 300
.\scripts\go-stock-cli.ps1 fund ranking --page-size 20
.\scripts\go-stock-cli.ps1 tool list
.\scripts\go-stock-cli.ps1 tool info --name GetStockOrderBook
.\scripts\go-stock-cli.ps1 tool GetStockOrderBook --stock-code 600237
```

JSON 输出：

```powershell
.\scripts\go-stock-cli.ps1 --json market major-index
```

常用环境变量：

```text
GO_STOCK_DB=可选；不设置时默认读取系统用户目录下的 go-stock/data/stock.db
GO_STOCK_CLI_VERBOSE=true；可选，打开 CLI 调试日志
```

如果直接运行 `go run ./cmd/go-stock-cli ...`，需要确保 `GOCACHE` 可写；Codex/沙箱内优先使用 `scripts\go-stock-cli.ps1`，它会自动把 Go 构建缓存放到仓库 `.gocache`。

## 功能目录

完整 CLI 功能树、中文作用、命令路径和 legacy 兼容说明见：

```text
skills/go-stock/references/tool-catalog.md
```

当任务需要选择不熟悉的功能、解释能力覆盖、生成调用计划、排查工具范围，先读取这个 reference。`SKILL.md` 只保留高频规则和流程。

## 功能选择规则

- 默认先运行 `go-stock-cli help` 看功能树，再调用具体命令路径。
- GUI 主树没有覆盖但归档工具层有的能力，用 `tool list` -> `tool info` -> `tool 工具名`。例如五档盘口可调用 `tool GetStockOrderBook`。
- 股票名称、简称、拼音或代码不确定时，用 `portfolio search` 或 `kline search` 确认。
- 市场复盘优先：`market news` -> `market major-index` -> `market money-flow stock` -> `research uplimit`。`market major-index` 不带参数时返回适合盯盘的市场总览；传 `--name` 或 `--code` 时查询单个指数 K 线。
- 个股分析优先：`portfolio view detail` -> `portfolio view daily-k`/`kline show` -> `portfolio view money` -> `portfolio view notice`/`portfolio view report`。
- 资金流按 GUI 对齐：个股资金 9 标签用 `market money-flow stock` 的 `sort` 参数；板块资金用 `market money-flow bk ...`；概念资金用 `market money-flow concept ...`。
- 基金使用 `fund follow` 和 `fund ranking`；基金搜索、详情、K线、净值、持仓分别用 `fund search/info/kline/nav/holdings`。
- 用户明确要管理自选分组时，用 `portfolio group list/add/rename/assign/remove`；重命名分组用 `portfolio group rename --group-id <ID> --new-name <新名称>`。
- 用户给持仓并要求 Agent 设置提醒时，使用 `portfolio position set`，先生成预览，用户二次确认后再带确认令牌写入。
- `MCP服务` 和 `名站优选` 不属于新 CLI 主功能树。

## 归档工具层选择规则

以下规则适用于 CLI 兼容层里的 `tool 工具名` 调用。

- 股票名称、简称、拼音或代码不确定时，先用 `QueryStockCodeInfo` 确认标准代码。
- 板块、行业、概念不确定时，先用 `QueryBKDictInfo` 或 `SearchBk`。
- 个股分析按顺序组合：行情/K线 -> 财务/F10 -> 资金流/龙虎榜 -> 新闻/公告/研报 -> 风险点。
- 只需要当前时间时用 `GetCurrentTime`；需要全球指数和开盘状态时用 `GetGlobalMarketStatus`，不要把两类信息混在一个上下文里。
- 需要买一/卖一、五档委托、封单或盘口深度时，用 `GetStockOrderBook`；普通个股行情用 `GetStockInfo`。
- 市场复盘按顺序组合：市场总览 -> 全球股指/北向资金 -> 异动排行 -> 热点事件 -> 涨停梯队。
- 条件选股优先用 `SearchStockByIndicators` 处理自然语言条件；需要结构化技术形态时用 `FilterStocks`。
- 研报、公告、政策、新闻搜索优先用 `FinanceSearch`，再按对象细分到 `SearchReport`、`SearchAnnouncement`、`SearchNews`。
- `FinanceSearch`、`HotspotDiscovery` 等东方财富妙想工具依赖 `EmApiKey`；`SearchAnnouncement` 和多数问财工具依赖 `IwencaiApiKey`。未配置时不要把它们当作稳定可用能力，优先改用 `GetStockNotice`、`QueryStockNewsTool`、`GetStockResearchReport`、`GetHotEventList` 等免 Key 工具兜底。
- 用户明确问前端“当前热门 > 热门话题”时，目标是 App/Wails 的 `HotTopic(size)`，底层是东方财富股吧话题接口；不要误用雪球“热门话题/事件”的 `GetHotEventList`。如果只能走 CLI 归档工具层，需说明 `HotTopic` 当前未直接迁移，`GetHotEventList` 只是近似替代。
- 涨停和题材分析优先用 `GetUplimitLadder`、`GetUplimitHotPlates`、`GetUplimitHotStocks`、`GetUplimitExplodedStocks`。
- 用户问“行业排名”下的“行业涨幅排名”时，用 `GetIndustryRank`；问“行业资金排名、证监会行业资金排名、概念板块资金排名”时，用 `GetIndustryMoneyRank` 并选择 `fenlei`，或直接用 `market industry-rank money/csrc-money/concept-money --sort netamount`。
- 用户问“个股资金流向”的 9 个排名标签时，用 `GetMoneyRankSina` 并选择 `sort`；不要用 `GetStockMoneyData` 替代这组前端标签。
- 用户问顶层“市场行情 > 板块资金流向”时，用 `GetBKFundFlowTopListByDate`、`GetBKFundFlowListByDate` 或 `GetAllBKCodes`；不要误用“行业排名 > 概念板块资金排名”的 `GetIndustryMoneyRank`。
- 用户问“涨跌家数比”“涨跌停家数比”时，优先用 `GetMarketData`；若在 App/Wails 内部，可直接用 `GetTodayMarketStatistic` 的 `upDownRatio`、`limitRatio` 字段。
- 用户问“当日异动次数最多的概念”时，优先用 `GetChangeRank(days=1, topN=...)`。如果本地 `stock_change_history` 或 `all_stock_info` 为空，改用实时 `GetStockChanges` 全量异动股票，再逐只用 `GetStockConceptInfo` 补概念并按概念聚合异动次数。
- 基金分析先 `SearchFund` 确认代码，再查 `GetFundInfo`、`GetFundHistoryNetValue`、`GetFundTop10Holdings`。
- 用户把持仓告诉 Agent 并要求设置提醒时，先用 `QueryStockCodeInfo` 确认代码，再用 `GetFollowedStocks` 查看已有设置；Agent 可以自行提出止损价、止盈价、涨跌提醒和股价提醒，但必须先调用 `portfolio position set` 生成预览和确认令牌，复述给用户并等待二次确认后，才允许用同一组参数加 `confirm=true` 和 `confirmToken` 写入。
- K 线命令支持 `002335` 这类深市前导 0 纯数字代码，也支持 `sz002335`、`002335.SZ`。
- `research uplimit` 若涨停梯队接口返回空或 0，会附带市场总览交叉验证；盯盘时不要单独依赖涨停梯队接口判断市场情绪。
- 涉及时效数据时，在回答中标明查询时间、日期参数或交易日。
- 输出研究结论时使用“数据解读/可能原因/风险提示”口径，不给确定性买卖建议。

## 高频工具速查

| 场景 | 优先工具 | 作用 |
|---|---|---|
| 查股票/指数代码 | `QueryStockCodeInfo` | 按名称、代码、拼音查基础证券信息。 |
| 查板块/行业字典 | `QueryBKDictInfo` | 获取板块、行业、概念名称和代码。 |
| 看市场全貌 | `GetMarketData` | 市场指数、涨跌分布、新股申购等总览。 |
| 看全球指数 | `GetGlobalMarketStatus` | 全球主要股指行情和开盘状态。 |
| 查行业涨幅排名 | `GetIndustryRank` | 对应“行业排名 > 行业涨幅排名”。 |
| 看个股行情 | `GetStockInfo` | 个股实时行情，涨跌按昨收计算，并附带盘口概览。 |
| 看盘口/封单 | `GetStockOrderBook` | 买一至买五、卖一至卖五、当前价、涨跌幅。 |
| 看K线趋势 | `GetEastMoneyKLineWithMA` | K 线并带均线，输出列顺序稳定。 |
| 查最新财务 | `GetStockLatestFinance` | EPS、ROE、营收、净利润等核心指标。 |
| 查估值位置 | `GetStockValuationPercentile` | PE 等估值历史分位。 |
| 查研报 | `GetStockResearchReport` / `SearchReport` | 个股研报或全局研报搜索。 |
| 查公告 | `SearchAnnouncement` / `GetStockNotice` | 公司公告搜索或个股公告。 |
| 查新闻政策 | `FinanceSearch` / `SearchNews` | 财经新闻、政策、公告、研报综合搜索。 |
| 查“当前热门 > 热门话题” | App/Wails `HotTopic(size)` | 前端热门话题页；底层为东方财富股吧话题。CLI 归档工具层未直接迁移时不要混同 `GetHotEventList`。 |
| 查雪球热门事件 | `GetHotEventList` | 雪球热门话题/事件，和前端“当前热门 > 热门话题”不是同一数据源。 |
| 查个股资金流榜单 | `GetMoneyRankSina` | 对应前端个股资金流向 9 个排名标签，可按 `sort` 切换。 |
| 查个股资金流Top | `GetStockMoneyData` | 东方财富今日个股资金流向 Top50。 |
| 查板块资金流向 | `GetBKFundFlowTopListByDate` / `GetBKFundFlowListByDate` | 对应前端顶层“板块资金流向”排名和折线数据。 |
| 查行业/概念资金排名 | `GetIndustryMoneyRank` | 对应“行业排名”内行业/概念/地域资金排名。 |
| 查龙虎榜 | `GetLongTigerList` / `GetStockBillboard` | 市场龙虎榜和个股龙虎榜。 |
| 看涨停梯队 | `GetUplimitLadder` | 涨停梯队、连板高度、市场情绪。 |
| 看热点板块 | `HotspotDiscovery` / `GetUplimitHotPlates` | 市场热点和涨停热门板块。 |
| 看涨跌/涨跌停家数比 | `GetMarketData` | 上涨/下跌家数、涨停/跌停家数；比值需要自行计算。 |
| 看当日异动概念排行 | `GetChangeRank` / `GetStockChanges` + `GetStockConceptInfo` | 优先查本地异动排行；本地表为空时用实时异动逐股映射概念后聚合。 |
| 自然语言选股 | `SearchStockByIndicators` | 用自然语言按指标筛选股票。 |
| 技术条件筛选 | `FilterStocks` | MACD、KDJ、均线、连涨连跌等条件筛选。 |
| 基金资料 | `SearchFund` / `GetFundInfo` | 搜索基金并查询基础信息。 |
| 基金持仓 | `GetFundTop10Holdings` | 查询基金前十大持仓。 |
| 行业研究 | `IndustryResearch` / `GetIndustryValuation` | 行业研究内容和估值。 |
| 可比公司 | `ComparableCompanyAnalysis` | 对公司做可比公司分析。 |
| 设置持仓提醒 | `portfolio position set` | CLI 受控写入命令；必须二次确认。 |

## 常用流程

个股画像：
1. `QueryStockCodeInfo` 确认股票代码。
2. `GetStockInfo` 和 `GetEastMoneyKLineWithMA` 查看行情与趋势；若问题涉及封单、买卖盘或委托队列，再调用 `GetStockOrderBook`。
3. `GetStockLatestFinance`, `GetStockQtrMainFinance`, `GetStockValuationPercentile` 查看财务和估值。
4. `GetMoneyRankSina`, `GetStockMoneyData`, `GetStockBillboard`, `GetStockResearchReport`, `SearchAnnouncement`, `FinanceSearch` 补充资金、龙虎榜、研报、公告和新闻。
5. 按“行情表现、基本面、资金/情绪、催化因素、风险点”输出。

市场复盘：
1. `GetMarketData` 查看市场整体状态。
2. `GetGlobalMarketStatus` 和 `QueryStockConnect` 补充外盘与资金。
3. `GetChangeRank`, `GetDailyChangeStats`, `HotspotDiscovery` 找异动和热点。
4. `GetUplimitLadder`, `GetHotEventList`, `FinanceSearch` 解释题材和可能原因。
5. 若需要“异动次数最多的概念”且 `GetChangeRank` 为空，实时兜底流程是：取 `GetStockChanges` 全量异动 -> 按股票代码去重 -> 对每只股票调用 `GetStockConceptInfo` -> 将每条异动计入该股所属概念 -> 输出概念总异动、利好异动、利空异动、涉及股票数。

当前热门：
1. 前端“当前热门 > 热门话题”用 App/Wails `HotTopic(size)`，展示字段通常是 `nickname`, `desc`, `postNumber`, `clickNumber`, `stock_list`, `htid`。
2. 话题链接格式为 `https://gubatopic.eastmoney.com/topic_v3.html?htid={htid}`。
3. 归档工具层中的 `GetHotEventList` 是雪球热门话题/事件，不等同于前端热门话题；只有用户接受近似替代时再使用。

热点涨停：
1. `GetUplimitLadder` 看连板高度和梯队。
2. `GetUplimitHotPlates` 和 `GetUplimitHotStocks` 看板块与个股。
3. `GetUplimitExplodedStocks` 识别分歧和炸板风险。
4. `QueryStockNewsTool` 或 `FinanceSearch` 补充题材催化。

条件选股：
1. 自然语言条件优先用 `SearchStockByIndicators`。
2. 结构化技术条件用 `FilterStocks`。
3. 按资产类型选择 `SelectAStock`, `SelectETF`, `SelectFund`, `SelectHKStock`, `SelectUSStock` 等。
4. 输出筛选条件、结果数量、排序依据和局限。

行业研究：
1. `QueryBKDictInfo` 或 `SearchBk` 确认行业/板块。
2. “行业排名”页：行业涨幅用 `GetIndustryRank`；行业资金、证监会行业资金、概念板块资金用 `GetIndustryMoneyRank`。
3. 顶层“板块资金流向”用 `GetAllBKCodes`, `GetBKFundFlowTopListByDate`, `GetBKFundFlowListByDate`。
4. `GetIndustryValuation`, `IndustryResearch` 查看估值和研究内容。
5. `FinanceSearch` 搜索政策、公告、研报和新闻。
6. 归纳产业逻辑、景气度、核心公司、风险点。

基金分析：
1. `SearchFund` 确认基金代码。
2. `GetFundInfo`, `GetFundHistoryNetValue`, `GetFundTop10Holdings` 查看基础信息、净值和持仓。
3. `GetFundKLine` 查看走势。
4. 输出基金类型、持仓集中度、回撤风险和适用场景。

持仓提醒设置：
1. 从用户输入中提取股票、成本价、持仓数量、风险偏好或提醒意图。
2. `QueryStockCodeInfo` 确认标准代码；必要时询问用户确认股票是否正确。
3. `GetFollowedStocks` 查看当前自选股成本、数量、开仓价、止盈价、止损价、涨跌提醒和股价提醒。
4. Agent 可以根据成本价、近期波动、用户风险偏好提出止盈价、止损价、涨跌提醒和股价提醒，并在 `reason` 中说明依据。
5. 首次调用 `portfolio position set` 时不确认写入，得到预览和 `confirmToken`，不得写库。
6. 只有用户明确确认后，才使用完全相同的参数加 `confirm=true` 和 `confirmToken` 再次调用写入。

## 安全边界

不要调用或建议开放这些有副作用的工具：`SetTradingPrice`, `SendDingDingMessage`, `SendToDingDing`, `CreateAiRecommendStocks`, `BatchCreateAiRecommendStocks`, `AiRecommendStocks`。

`portfolio position set` 是唯一受控写入例外；必须遵守“预览 -> 用户二次确认 -> 带确认令牌写入”的流程。

不要把 MCP/Skill 配置管理工具当作财经数据服务迁移到 CLI raw 入口，例如 `CreateMCPServer`, `UpdateMCPServer`, `DeleteMCPServer`, `CreateSkill`, `UpdateSkill`, `DeleteSkill`。
