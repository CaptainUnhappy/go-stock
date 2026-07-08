# go-stock CLI tool 分类建议

本清单基于当前 `.\go-stock-cli.exe tool list --includeBlocked true` 输出整理，用于判断归档兼容层里的旧工具应迁移到哪个 CLI 板块，或继续作为底层原子能力保留在 `tool` 中。

状态说明：

- `可归类`: 建议补充或继续维护对应语义化 CLI 命令。
- `保留 tool`: 更适合作为底层查询、数据源兜底、调试或跨板块原子能力，不建议膨胀到主功能树。
- `禁用`: 有写入、通知、AI 分析/推荐或历史 MCP denylist 风险，不开放 raw tool 直连。

## Agent CLI 使用引导

优先顺序：

1. 先用语义化 CLI 路径，例如 `market news`、`kline show`、`portfolio view detail`、`fund ranking`。
2. 语义化路径没有覆盖时，再用 `tool info --name <Tool>` 查看参数。
3. 最后才调用 `tool <Tool>`。raw tool 是兼容层，不是主入口。

股票参数统一优先写成 `--stock-code`：

```powershell
.\go-stock-cli.exe tool GetStockInfo --stock-code sz002335
.\go-stock-cli.exe tool GetStockInfo --stock-code "sz002335,sz002506,sh603690"
.\go-stock-cli.exe tool GetStockInfo --stock-code sz002335 sz002506 sh603690
```

管道股票池统一使用 `--stock-code -` 或 `--stdin`：

```powershell
Get-Content .\watchlist.txt | .\go-stock-cli.exe tool GetStockLatestFinance --stock-code -
.\go-stock-cli.exe portfolio list | .\go-stock-cli.exe tool GetStockInfo --stdin
```

stdin 会自动提取 `sz002335`、`sh603690`、`bj430047`、`002335.SZ` 和裸 6 位代码，去重并保持顺序。只有显式写 `--stdin` 或 `--stock-code -`/`--stock-codes -` 时才读取管道。

## 归类总览

### 股票自选 / 个股快捷查看

优先语义化命令：

```powershell
.\go-stock-cli.exe portfolio list
.\go-stock-cli.exe portfolio view detail --stock-code 600237
.\go-stock-cli.exe portfolio view minute --stock-code 600237
.\go-stock-cli.exe portfolio view daily-k --stock-code 600237
.\go-stock-cli.exe portfolio view money --stock-code 600237
.\go-stock-cli.exe portfolio view notice --stock-code 600237
.\go-stock-cli.exe portfolio view report --stock-code 600237
```

相关工具（语义化命令 + raw 兼容层）：

- `GetFollowedStocks`: 自选/持仓列表，对应 `portfolio list`、`portfolio position get`。
- `GetStockInfo`: 实时行情、五档盘口、成交额/换手/估值等，优先用于盯盘。
- `GetStockOrderBook`: 专用五档盘口，盘口为空时 CLI 会用 `GetStockInfo` 兜底。
- `GetStockMinuteData`: 分时。
- `GetEastMoneyKLineWithMA`, `GetEastMoneyKLine`, `GetStockKLine`: K 线与均线。
- `GetStockMoneyData`, `GetStockHistoryMoneyData`, `GetMACCapitalFlow`: 个股资金。
- `GetStockNotice`, `StockNotice`: 公告。
- `GetStockResearchReport`: 个股研报。
- `GetStockLatestFinance`, `GetStockQtrMainFinance`, `GetStockFinancialInfo`, `GetTdxFinanceInfo`: 财务。
- `GetStockBillboard`, `GetLongTigerList`, `GetStockOperationDeptTrade`: 龙虎榜和营业部。
- `GetStockBlockTrade`, `GetStockCallAuction`, `GetStockMarginTrading`, `GetStockRZRQInfo`, `GetStockHolderNum`, `GetStockHolderTrend`, `GetStockOrgPredict`, `GetStockPredictSummary`, `GetStockValuationPercentile`, `GetTdxXDXRInfo`: 个股 F10 扩展。
- `GetStockConceptInfo`, `GetTdxSymbolBelongBoard`: 概念/板块归属。
- `GetTdxCompanyInfo`, `GetTdxCompanyCategory`: 通达信 F10 兜底。

### 市场行情

优先语义化命令：

```powershell
.\go-stock-cli.exe market news
.\go-stock-cli.exe market major-index
.\go-stock-cli.exe market global-index
.\go-stock-cli.exe market industry-rank gain
.\go-stock-cli.exe market industry-rank concept-money --sort netamount --limit 20
.\go-stock-cli.exe market money-flow stock --sort r0_net --limit 20
.\go-stock-cli.exe market hot cn
.\go-stock-cli.exe market hot calendar --year-month 2026-07
.\go-stock-cli.exe market announcement
.\go-stock-cli.exe market announcement --stock-code 600237
```

相关工具（语义化命令 + raw 兼容层）：

- `GetMarketData`: 市场总览，主要股指、涨跌家数、涨跌停、涨跌分布、新股申购。
- `GlobalStockIndexesReadable`, `GetGlobalMarketStatus`: 全球股指和开盘状态。
- `GetIndustryRank`: 行业涨幅排名。
- `GetIndustryMoneyRank`: 行业资金、证监会行业资金、概念板块资金。
- `GetMoneyRankSina`: 个股资金流向 9 个子标签。
- `GetAllBKCodes`, `GetBKFundFlowTopList`, `GetBKFundFlowTopListByDate`, `GetBKFundFlowList`, `GetBKFundFlowListByDate`: 板块/概念资金流向。
- `GetChangeRank`, `GetDailyChangeStats`, `GetDailyDimensionStats`, `GetTypeStatsByDate`, `GetStockChanges`, `GetStockChangeHistoryList`: 市场异动。
- `GetNewsListData`, `QueryStockNewsTool`, `GetWallstreetcnLives`: 快讯和新闻兜底。
- `GetHotStockList`, `HotStockTable`, `GetHotEventList`, `GetInvestCalendar`: 当前热门、热门股、事件和财经日历。
- `GetMutualTop10Deal`: 沪深港通十大成交。
- `GetEconomicData`, `GetSecuritiesCompanyOpinion`, `GetIndustryValuation`: 宏观、券商观点和行业估值。`IndustryResearch` 是内部研究能力名，当前不在 `tool list` 直连清单中，不能当作 raw tool 调用。

### K线分析 / 形态与指标筛选

优先语义化命令：

```powershell
.\go-stock-cli.exe kline show --stock-code 002335 --k-line-type day --adjust qfq --limit 120
.\go-stock-cli.exe kline signals --stock-code 002335 --k-line-type day --adjust qfq --limit 250
.\go-stock-cli.exe research pattern-screen --form MACD金叉
.\go-stock-cli.exe research indicator-screen --query "近20日放量突破且ROE大于10"
```

相关工具（语义化命令 + raw 兼容层）：

- `GetEastMoneyKLineWithMA`: K 线展示主工具，带均线摘要。
- `GetEastMoneyKLine`, `GetStockKLine`: K 线数据源兜底或多股 K 线。
- `FilterStocks`: 形态选股。
- `SearchStockByIndicators`: 自然语言指标选股。
- `HotStrategyTable`: 热门策略参考，保留为研究辅助，不使用 AI 推荐写入。

### 研究中心 / 涨停热点 / 异动

优先语义化命令：

```powershell
.\go-stock-cli.exe research changes
.\go-stock-cli.exe research uplimit
.\go-stock-cli.exe market hot timeline
```

相关工具（语义化命令 + raw 兼容层）：

- `GetUplimitLadder`: 涨停梯队。
- `GetUplimitHotPlates`: 涨停热门板块。
- `GetUplimitHotStocks`: 涨停热门个股。
- `GetUplimitExplodedStocks`: 炸板股。
- `GetUplimitPlateStocks`: 指定板块涨停股。
- `GetStockChanges`, `GetStockChangeHistoryList`: 实时/历史异动。
- `GetChangeRank`: 异动排行榜。

### 基金 / ETF

优先语义化命令：

```powershell
.\go-stock-cli.exe fund ranking --page-size 20
.\go-stock-cli.exe fund search --keyword 半导体
.\go-stock-cli.exe fund info --fund-code 161725
.\go-stock-cli.exe fund nav --fund-code 161725
.\go-stock-cli.exe fund holdings --fund-code 161725
```

相关工具（语义化命令 + raw 兼容层）：

- `SearchFund`: 基金搜索。
- `SearchETF`: ETF 搜索。
- `GetFundInfo`: 基金详情。
- `GetFundHistoryNetValue`: 基金净值。
- `GetFundKLine`: 基金 K 线。
- `GetFundTop10Holdings`: 基金持仓。

### 交易日历 / 时间

这些工具已归到独立 `calendar` 一级菜单；raw `tool` 仍作为兼容入口保留：

```powershell
.\go-stock-cli.exe calendar now
.\go-stock-cli.exe calendar is-trading-day --date 2026-07-03
.\go-stock-cli.exe calendar next-trading-day --date 2026-07-03
.\go-stock-cli.exe calendar holiday --date 2026-10-01
.\go-stock-cli.exe calendar holiday-year --year 2026
.\go-stock-cli.exe calendar holiday-batch --dates "2026-07-03,2026-10-01"
```

- `IsTradingDay`: 是否 A 股交易日。
- `GetNextTradingDay`: 下一交易日。
- `GetHolidayInfo`: 单日节假日。
- `GetHolidayYear`: 年度节假日。
- `GetHolidayBatch`: 批量节假日。
- `GetCurrentTime`: 本地时间。
- `GetWallstreetcnCalendar`: 全球财经日历兜底。

### 基础查询 / 数据源兜底

这些工具跨多个板块使用，保留为底层原子能力更合适：

- `QueryStockCodeInfo`: 股票/指数代码查询。
- `QueryBKDictInfo`: 板块、行业、概念字典。
- `SearchBk`: 自然语言板块/概念/指数查询。
- `GetWallstreetcnKline`, `GetWallstreetcnMarketReal`: 全球品种 K 线和实时行情兜底。

### 禁用或受控替代

不进入 CLI raw 直连：

- `AiRecommendStocks`, `CreateAiRecommendStocks`, `BatchCreateAiRecommendStocks`
- `GetAIAnalysisHistory`, `GetAIAnalysisDetail`, `GetAIAnalysisContent`
- `SetTradingPrice`, `SetFollowedStockPosition`
- 通知类：`SendDingDingMessage`, `SendToDingDing`

持仓成本、数量、止损、止盈、涨跌提醒、股价提醒统一走：

```powershell
.\go-stock-cli.exe portfolio position set --stock-code 600237 --cost-price 12.56 --volume 300
```

该命令必须遵守预览/确认令牌二次确认流程。

## 完整工具审计表

| Tool | 简单解释 | 可参考分类板块 | 建议 |
| --- | --- | --- | --- |
| `AiRecommendStocks` | 查询 AI 推荐股票记录。 | 研究中心 / 股票推荐记录 | 禁用 |
| `BatchCreateAiRecommendStocks` | 批量创建 AI 推荐记录。 | 研究中心 / 股票推荐记录 | 禁用 |
| `CreateAiRecommendStocks` | 创建 AI 推荐记录。 | 研究中心 / 股票推荐记录 | 禁用 |
| `FilterStocks` | 按 K 线形态、技术指标、人气、连涨连跌等筛选股票。 | 研究中心 / 形态选股 | 可归类。 |
| `GetAIAnalysisContent` | 获取某股票最新 AI 分析报告正文。 | 研究中心 / AI分析报告 / 报告正文 | 禁用 |
| `GetAIAnalysisDetail` | 按 ID 获取历史 AI 分析详情。 | 研究中心 / AI分析报告 / 报告详情 | 禁用 |
| `GetAIAnalysisHistory` | 查询历史 AI 分析报告列表。 | 研究中心 / AI分析报告 | 禁用 |
| `GetAllBKCodes` | 获取板块资金流向可用板块代码。 | 市场行情 / 板块资金流向 / 板块列表 | 可归类。 |
| `GetBKFundFlowList` | 获取某板块资金流历史趋势。 | 市场行情 / 板块资金流向 / 板块资金趋势 | 可归类。 |
| `GetBKFundFlowListByDate` | 获取某板块指定日期资金流趋势。 | 市场行情 / 板块资金流向 / 指定日期趋势 | 可归类。 |
| `GetBKFundFlowTopList` | 获取最新板块资金流排名。 | 市场行情 / 板块资金流向 / 最新资金排名 | 可归类。 |
| `GetBKFundFlowTopListByDate` | 获取指定日期板块资金流排名。 | 市场行情 / 板块资金流向 / 指定日期资金排名 | 可归类。 |
| `GetChangeRank` | 获取异动次数排行榜，支持股票、行业、概念维度。 | 市场行情 / 市场快讯 / 异动排行 | 可归类。 |
| `GetCurrentTime` | 获取本地当前时间和星期。 | 交易日历 / 当前时间 | 已归类到 `calendar now`。 |
| `GetDailyChangeStats` | 获取近 N 日每日异动统计趋势。 | 市场行情 / 市场快讯 / 每日异动统计 | 可归类。 |
| `GetDailyDimensionStats` | 按股票、行业、概念、异动类型查询异动趋势。 | 市场行情 / 市场快讯 / 维度异动统计 | 可归类。 |
| `GetEastMoneyKLine` | 获取股票 K 线，支持多周期和复权。 | K线分析 / K线展示 | 可归类。 |
| `GetEastMoneyKLineWithMA` | 获取带均线的 K 线。 | K线分析 / K线展示 / 均线 | 可归类。 |
| `GetEconomicData` | 获取 GDP、CPI、PPI、PMI 等宏观数据。 | 市场行情 / 宏观经济 | 可归类。 |
| `GetFollowedStocks` | 获取自选股和持仓设置。 | 股票自选 / 自选列表 | 可归类。 |
| `GetFundHistoryNetValue` | 获取基金历史净值。 | 基金 / 基金净值 | 可归类。 |
| `GetFundInfo` | 获取基金详情、净值、涨跌幅、评级等。 | 基金 / 基金详情 | 可归类。 |
| `GetFundKLine` | 获取基金 K 线。 | 基金 / 基金K线 | 可归类。 |
| `GetFundTop10Holdings` | 获取基金前十大持仓。 | 基金 / 基金持仓 | 可归类。 |
| `GetGlobalMarketStatus` | 获取全球主要指数和开盘状态。 | 市场行情 / 全球股指 | 可归类。 |
| `GetHolidayBatch` | 批量查询多个日期节假日信息。 | 交易日历 / 批量节假日 | 已归类到 `calendar holiday-batch`。 |
| `GetHolidayInfo` | 查询指定日期节假日信息。 | 交易日历 / 节假日查询 | 已归类到 `calendar holiday`。 |
| `GetHolidayYear` | 查询指定年份全部节假日。 | 交易日历 / 年度节假日 | 已归类到 `calendar holiday-year`。 |
| `GetHotEventList` | 获取雪球热门话题和事件。 | 市场行情 / 当前热门 / 重大事件时间轴 | 可归类；注意不是前端东财热门话题。 |
| `GetHotStockList` | 获取雪球热门股票榜。 | 市场行情 / 当前热门 / 全球/沪深/港股/美股 | 可归类。 |
| `GetIndustryMoneyRank` | 查询行业、证监会行业、概念板块资金排名。 | 市场行情 / 行业排名 / 资金排名 | 可归类。 |
| `GetIndustryRank` | 查询行业涨幅排名。 | 市场行情 / 行业排名 / 行业涨幅排名 | 可归类。 |
| `GetIndustryValuation` | 获取行业或板块估值均值、中值。 | 市场行情 / 行业研究 / 行业估值 | 可归类。 |
| `GetInvestCalendar` | 获取财报、股东大会、IPO 等投资日历。 | 市场行情 / 当前热门 / 财经日历 | 已归类到 `market hot calendar`；数据源无数据时会用 `GetWallstreetcnCalendar` 兜底。 |
| `GetLongTigerList` | 获取市场龙虎榜营业部排行。 | 市场行情 / 龙虎榜 | 可归类。 |
| `GetMACCapitalFlow` | 通过通达信 MAC 获取个股资金流。 | tool / 个股资金底层数据源 | 保留 tool。 |
| `GetMarketData` | 获取指数、涨跌家数、涨跌停、涨跌分布、新股申购等市场总览。 | 市场行情 / 市场快讯 / 市场总览 | 可归类。 |
| `GetMoneyRankSina` | 获取新浪个股资金流向 9 类排名。 | 市场行情 / 个股资金流向 | 可归类。 |
| `GetMutualTop10Deal` | 获取北向/南向资金十大成交股。 | 市场行情 / 互联互通资金 | 可归类。 |
| `GetNewsListData` | 获取新闻资讯列表。 | 市场行情 / 市场快讯 / 快讯列表 | 可归类。 |
| `GetNextTradingDay` | 查询指定日期后的下一个 A 股交易日。 | 交易日历 / 下一交易日 | 已归类到 `calendar next-trading-day`。 |
| `GetSecuritiesCompanyOpinion` | 获取券商或机构市场观点。 | 市场行情 / 行业研究 / 券商观点 | 可归类。 |
| `GetStockBillboard` | 获取个股龙虎榜数据。 | 股票自选 / 个股快捷查看 / 龙虎榜 | 可归类。 |
| `GetStockBlockTrade` | 获取个股大宗交易数据。 | 股票自选 / 个股快捷查看 / 大宗交易 | 可归类。 |
| `GetStockCallAuction` | 查询集合竞价明细。 | 股票自选 / 个股快捷查看 / 集合竞价 | 可归类。 |
| `GetStockChangeHistoryList` | 查询股票异动历史记录。 | 研究中心 / 异动监控 / 历史异动 | 可归类。 |
| `GetStockChanges` | 获取实时股票异动。 | 研究中心 / 异动监控 | 可归类。 |
| `GetStockConceptInfo` | 查询个股概念板块信息。 | 股票自选 / 个股快捷查看 / 概念板块 | 可归类。 |
| `GetStockFinancialInfo` | 获取个股财务报表。 | 股票自选 / 个股快捷查看 / 财务报表 | 可归类。 |
| `GetStockHistoryMoneyData` | 获取个股历史资金流。 | 股票自选 / 个股快捷查看 / 资金 | 可归类。 |
| `GetStockHolderNum` | 获取股东人数。 | 股票自选 / 个股快捷查看 / 股东人数 | 可归类。 |
| `GetStockHolderTrend` | 获取股东户数和户均持股趋势。 | 股票自选 / 个股快捷查看 / 股东趋势 | 可归类。 |
| `GetStockInfo` | 获取个股实时行情和五档盘口概览。 | 股票自选 / 个股快捷查看 / 详情 | 可归类。 |
| `GetStockKLine` | 获取股票日 K 线，支持多股。 | K线分析 / K线展示 | 保留 tool；优先使用 `GetEastMoneyKLineWithMA` 或 `kline show`。 |
| `GetStockLatestFinance` | 获取最新财务核心指标。 | 股票自选 / 个股快捷查看 / 最新财务 | 可归类。 |
| `GetStockMarginTrading` | 获取融资融券明细。 | 股票自选 / 个股快捷查看 / 融资融券 | 可归类。 |
| `GetStockMinuteData` | 获取当日分时数据。 | 股票自选 / 个股快捷查看 / 分时 | 可归类。 |
| `GetStockMoneyData` | 获取今日个股资金流 Top 数据。 | 股票自选 / 个股快捷查看 / 资金 | 可归类。 |
| `GetStockNotice` | 获取个股公告。 | 股票自选 / 个股快捷查看 / 公告 | 已归类到 `portfolio view notice`，也可用 `market announcement --stock-code`。 |
| `GetStockOperationDeptTrade` | 获取龙虎榜营业部买卖明细。 | 股票自选 / 个股快捷查看 / 营业部买卖 | 可归类。 |
| `GetStockOrderBook` | 获取五档盘口。 | 股票自选 / 个股快捷查看 / 盘口 | 可归类。 |
| `GetStockOrgPredict` | 获取机构预测明细。 | 股票自选 / 个股快捷查看 / 机构预测 | 可归类。 |
| `GetStockPredictSummary` | 获取机构预测汇总。 | 股票自选 / 个股快捷查看 / 机构预测汇总 | 可归类。 |
| `GetStockQtrMainFinance` | 获取季度主要财务指标。 | 股票自选 / 个股快捷查看 / 季度财务 | 可归类。 |
| `GetStockRZRQInfo` | 获取融资融券概览。 | 股票自选 / 个股快捷查看 / 融资融券概览 | 可归类。 |
| `GetStockResearchReport` | 获取个股研究报告。 | 股票自选 / 个股快捷查看 / 研报 | 可归类。 |
| `GetStockValuationPercentile` | 获取估值历史百分位。 | 股票自选 / 个股快捷查看 / 估值百分位 | 可归类。 |
| `GetTdxCompanyCategory` | 获取通达信 F10 分类内容。 | tool / 通达信 F10 原子能力 | 保留 tool；适合作为 F10 兜底。 |
| `GetTdxCompanyInfo` | 获取通达信 F10 公司资料。 | tool / 通达信 F10 原子能力 | 保留 tool；适合作为 F10 兜底。 |
| `GetTdxFinanceInfo` | 获取通达信财务指标。 | tool / 通达信财务原子能力 | 保留 tool；适合作为财务兜底。 |
| `GetTdxSymbolBelongBoard` | 获取通达信股票所属板块。 | tool / 通达信板块原子能力 | 保留 tool；适合作为概念/行业兜底。 |
| `GetTdxXDXRInfo` | 获取通达信除权除息信息。 | 股票自选 / 个股快捷查看 / 除权除息 | 可归类。 |
| `GetTypeStatsByDate` | 查询某日异动类型分布。 | 市场行情 / 市场快讯 / 异动类型统计 | 可归类。 |
| `GetUplimitExplodedStocks` | 获取炸板股。 | 研究中心 / 涨停梯队 / 炸板股 | 可归类。 |
| `GetUplimitHotPlates` | 获取涨停热门板块。 | 研究中心 / 涨停梯队 / 热门板块 | 可归类。 |
| `GetUplimitHotStocks` | 获取涨停热门个股。 | 研究中心 / 涨停梯队 / 热门个股 | 可归类。 |
| `GetUplimitLadder` | 获取连板梯队。 | 研究中心 / 涨停梯队 | 可归类。 |
| `GetUplimitPlateStocks` | 获取指定板块涨停股。 | 研究中心 / 涨停梯队 / 板块涨停股 | 可归类。 |
| `GetWallstreetcnCalendar` | 获取华尔街见闻财经日历。 | tool / 华尔街见闻扩展数据源 | 保留 tool。 |
| `GetWallstreetcnKline` | 获取华尔街见闻品种 K 线。 | tool / 华尔街见闻扩展数据源 | 保留 tool。 |
| `GetWallstreetcnLives` | 获取华尔街见闻实时快讯。 | tool / 华尔街见闻扩展数据源 | 保留 tool；可作为资讯兜底。 |
| `GetWallstreetcnMarketReal` | 获取华尔街见闻全球实时行情。 | tool / 华尔街见闻扩展数据源 | 保留 tool；可作为外盘兜底。 |
| `GlobalStockIndexesReadable` | 获取全球主要指数概览。 | 市场行情 / 全球股指 | 可归类。 |
| `HotStockTable` | 获取当前热门股票排名表。 | 市场行情 / 当前热门 / 热门股票表 | 可归类。 |
| `HotStrategyTable` | 获取热门选股策略。 | 研究中心 / 股票推荐记录 / 热门策略 | 可归类。 |
| `InteractiveAnswer` | 获取投资者互动问答。 | 市场行情 / 公司公告 / 互动问答 | 可归类。 |
| `IsTradingDay` | 判断指定日期是否 A 股交易日。 | 交易日历 / 是否交易日 | 已归类到 `calendar is-trading-day`。 |
| `QueryBKDictInfo` | 查询板块、行业、概念字典。 | tool / 基础查询原子能力 | 保留 tool；也可作为 `market board dict`。 |
| `QueryStockCodeInfo` | 查询股票或指数代码、名称、拼音、交易所。 | tool / 基础查询原子能力 | 保留 tool；也可作为 `security search`。 |
| `QueryStockNewsTool` | 按关键词搜索市场新闻。 | tool / 新闻搜索原子能力 | 保留 tool；可作为资讯兜底。 |
| `SearchBk` | 用自然语言查询板块、概念、指数整体数据。 | tool / 板块搜索原子能力 | 保留 tool；跨市场行情和研究中心。 |
| `SearchETF` | 用自然语言查询 ETF 数据。 | 基金 / ETF搜索 | 可归类。 |
| `SearchFund` | 搜索基金代码或名称。 | 基金 / 基金搜索 | 可归类。 |
| `SearchStockByIndicators` | 自然语言指标选股。 | 研究中心 / 指标选股 | 可归类。 |
| `SetFollowedStockPosition` | 设置自选股持仓、成本和提醒。 | 股票自选 / 持仓提醒设置 | 禁用 raw；使用 `portfolio position set` 二次确认。 |
| `SetTradingPrice` | 设置开仓、止盈、止损、成本价。 | 股票自选 / 持仓提醒设置 | 禁用 raw；使用 `portfolio position set` 二次确认。 |
| `StockNotice` | 获取上市公司公告列表。 | 市场行情 / 公司公告 | 已归类到 `market announcement`。 |

## 优先迁移建议

第一批适合补成语义化命令的工具：

- 个股快捷查看：`GetStockOrderBook`, `GetStockCallAuction`, `GetStockBlockTrade`, `GetStockMarginTrading`, `GetStockValuationPercentile`, `GetStockOrgPredict`, `GetStockPredictSummary`
- 市场行情扩展：`GetEconomicData`, `GetMutualTop10Deal`, `GetSecuritiesCompanyOpinion`
- 研究中心扩展：`GetStockChangeHistoryList`, `GetDailyChangeStats`, `GetTypeStatsByDate`

建议继续保留在 `tool` 的原子能力：

- 基础查询：`QueryStockCodeInfo`, `QueryBKDictInfo`, `SearchBk`
- 数据源兜底：`GetMACCapitalFlow`, `GetStockKLine`, `GetTdxCompanyCategory`, `GetTdxCompanyInfo`, `GetTdxFinanceInfo`, `GetTdxSymbolBelongBoard`
- 扩展资讯源：`GetWallstreetcnCalendar`, `GetWallstreetcnKline`, `GetWallstreetcnLives`, `GetWallstreetcnMarketReal`

必须保持禁用或受控替代：

- `CreateAiRecommendStocks`, `BatchCreateAiRecommendStocks`: 写入 AI 推荐记录。
- `AiRecommendStocks`: AI 推荐记录工具不使用。
- `GetAIAnalysisHistory`, `GetAIAnalysisDetail`, `GetAIAnalysisContent`: AI 分析历史/详情/正文工具不使用。
- `SetFollowedStockPosition`, `SetTradingPrice`: raw 入口禁用，统一走 `portfolio position set` 的预览和二次确认流程。
