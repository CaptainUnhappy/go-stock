# go-stock CLI 功能词典

本文件是 `go-stock` Skill 的功能目录。当前长期主入口是本地 CLI：

```powershell
.\go-stock-cli.exe help
```

在 Codex/沙箱环境和日常使用中都默认直接运行仓库根目录的 `go-stock-cli.exe`。如果 exe 不存在或需要更新，先运行 `.\scripts\build-windows.ps1` 重新打包；只有源码调试或 exe 缺失时才使用 `.\scripts\go-stock-cli.ps1` 备用包装器，它会把 Go 构建缓存放到仓库 `.gocache`。

Agent 应先用 `go-stock-cli help` 读取功能树，再调用具体命令路径。GUI 主树没有覆盖但原对外 MCP 有的能力，使用 CLI 归档工具兼容层：

```powershell
.\go-stock-cli.exe tool list
.\go-stock-cli.exe tool info --name GetStockInfo
.\go-stock-cli.exe tool GetStockInfo --stock-code 600237
.\go-stock-cli.exe tool GetStockInfo --stock-code "sz002335,sz002506"
Get-Content .\watchlist.txt | .\go-stock-cli.exe tool GetStockInfo --stock-code -
```

项目本地对外 MCP 服务已归档；旧的 90+ 原始 MCP tools/list 不再是新集成目标。

## 默认 CLI 功能树

```text
股票自选
├─ portfolio list/search/add/remove
├─ portfolio position get/set
├─ portfolio group list/add/rename/assign/remove
└─ portfolio view minute/daily-k/multi-k/money/detail/notice/report

市场行情
├─ market news
├─ market global-index
├─ market major-index
├─ market industry-rank gain/money/csrc-money/concept-money
├─ market money-flow stock
├─ market money-flow bk list/latest/date/trend
├─ market money-flow concept list/latest/date/trend
├─ market billboard/stock-report/announcement/industry-research
└─ market hot global/cn/hk/us/topic/timeline/calendar

K线分析
├─ kline search
├─ kline recent
├─ kline show
└─ kline signals

基金
├─ fund follow
├─ fund ranking
└─ fund search/info/kline/nav/holdings

交易日历
├─ calendar now
├─ calendar is-trading-day
├─ calendar next-trading-day
└─ calendar holiday/holiday-year/holiday-batch

研究中心
├─ research ai-report(禁用)/recommend(禁用)/changes/uplimit
├─ research prompt-template(禁用)/prompt-plaza(禁用)/qa-plaza(禁用)
└─ research pattern-screen/indicator-screen/cron-task(禁用)/trade-log(禁用)

归档工具兼容层
├─ tool list
├─ tool info
└─ tool <原工具名>
```

`名站优选` 不进入 CLI 主功能树。`MCP服务` 是 go-stock GUI 接入外部工具的管理页，也不进入当前项目对外 MCP/CLI 功能树。

## CLI 归档工具兼容层

常用 GUI 对齐命令示例：

```powershell
.\go-stock-cli.exe market news
.\go-stock-cli.exe market major-index
.\go-stock-cli.exe market major-index --name 上证指数
.\go-stock-cli.exe market industry-rank money --sort netamount --limit 20
.\go-stock-cli.exe market industry-rank csrc-money --sort netamount --limit 20
.\go-stock-cli.exe market industry-rank concept-money --sort netamount --limit 20
.\go-stock-cli.exe market money-flow stock --sort r0_net --limit 20
.\go-stock-cli.exe kline show --stock-code 002335 --k-line-type day --adjust qfq --limit 120
.\go-stock-cli.exe kline signals --stock-code 002335 --k-line-type day --adjust qfq --limit 250
.\go-stock-cli.exe calendar is-trading-day --date 2026-07-03
.\go-stock-cli.exe calendar next-trading-day --date 2026-07-03
.\go-stock-cli.exe tool GetStockLatestFinance --stockCode='sz002335,sz002506,sh603690'
```

`market major-index` 不带参数时返回适合盯盘的市场总览；传 `--name` 或 `--code` 时查询单个指数 K 线。K 线命令支持 `002335` 这类深市前导 0 代码，不需要强制改成 `sz002335`。
`kline show` 和 `kline signals` 对齐 GUI K线复权选择：日K及更长周期默认 `--adjust qfq` 前复权，可传 `--adjust hfq` 后复权或 `--adjust none` 不复权；分钟线忽略复权。
股票池来自文件、上一条 go-stock 输出或 Agent 生成列表时，用 `--stock-code -` 或 `--stdin` 从管道读取；CLI 会自动提取 `sz002335`、`002335.SZ` 和裸 6 位代码并去重保序。

`kline signals` 对齐 GUI K线分析页“指标信号汇总”，输出看多/看空/震荡/中性统计和逐指标标签。

| 命令 | 作用 | 常见输入 |
|---|---|---|
| `tool list` | 查看全部已迁移的归档工具、CLI 调用路径和状态。 | `includeBlocked`: 是否显示被禁用工具。 |
| `tool info` | 查看归档工具的作用和 JSON Schema 参数。 | `name`: 原工具名，例如 `GetStockOrderBook`。 |
| `tool <原工具名>` | 调用已迁移的归档工具。 | 参数沿用原工具；可先用 `tool info` 查看。 |

持仓/提醒设置必须通过 `portfolio position set` 二次确认。原写入工具 `SetFollowedStockPosition` 不提供 raw `tool SetFollowedStockPosition` 直连入口。

raw tool 参数和盯盘注意事项：

- 股票代码参数可用 `--stockCode`、`--stock-code`、`--stock_code`、`--stockcode`；缺失时 `GetStockInfo` / `GetStockOrderBook` 会直接提示参数错误。
- PowerShell 多股参数建议加引号，例如 `--stockCode='sh600237,sz002335'`，也可用 `--args-json '{"stockCode":"sh600237,sz002335"}'`。
- 管道输入用 `--stock-code -` 或 `--stdin`，例如 `Get-Content .\watchlist.txt | .\go-stock-cli.exe tool GetStockInfo --stock-code -`。
- 盯盘优先用 `tool GetStockInfo`；它包含行情和五档盘口概览。`GetStockOrderBook` 是专用盘口工具，返回空盘口时 CLI 会自动尝试 `GetStockInfo` 兜底。
- Codex 审批或执行链路出现 `stream disconnected before completion` 时，属于执行失败，不等同于 go-stock 数据源为空；只能复用上一轮成功数据并标明时间。
- 收盘后盘口仍可能返回最近快照，只能按收盘附近快照理解，不代表仍可成交。

## 归档工具词典

以下工具已迁移到 CLI `tool <原工具名>` 兼容层。被禁用的写入/通知工具在文末单独列出。

## 基础查询与交易日历

| 工具 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `QueryStockCodeInfo` | 查询股票/指数名称、代码、拼音、交易所等基础信息；本地表未命中时会用内置 `stock_basic.json` 和代码格式兜底。 | 用户给的是简称、中文名、拼音或不完整代码。 | `searchWord`: 股票名、代码、拼音，如 `中晶科技`、`003026`。 |
| `QueryBKDictInfo` | 查询板块、行业、概念名称和代码字典。 | 需要确认行业/概念/板块代码。 | 无参数。 |
| `GetCurrentTime` | 只获取当前本地时间和星期，不附带行情表。 | 需要给行情或报告标注查询时间。 | 无参数。 |
| `GetFollowedStocks` | 查询本地关注/自选股票，返回股票代码、名称、成本价格、持仓数量、开仓价、止盈价、止损价、涨跌提醒、股价提醒和排序。它是 go-stock 本地元数据，不是券商真实持仓同步。 | 用户问“我的自选股”“关注列表”，或在设置持仓提醒前读取当前值。 | `groupId`: 可选分组 ID；不传返回所有自选。 |
| `GetHolidayInfo` | 查询某天是否节假日、是否调休。 | 判断某个日期是否休市或节假日。 | `date`: `YYYY-MM-DD`，可省略。 |
| `GetHolidayYear` | 查询某年的节假日安排。 | 年度交易日/节假日分析。 | `year`: 年份。 |
| `GetHolidayBatch` | 批量查询多个日期节假日信息。 | 需要一次判断多个日期。 | 日期列表。 |
| `IsTradingDay` | 判断指定日期是否 A 股交易日。 | 回测、计划任务、查询交易日状态。 | `date`: `YYYY-MM-DD`。 |
| `GetNextTradingDay` | 查询下一个交易日。 | 用户问“下个交易日”“下一开盘日”。 | `date`: `YYYY-MM-DD`，可省略。 |

优先使用独立 `calendar` 语义命令：

```powershell
.\go-stock-cli.exe calendar now
.\go-stock-cli.exe calendar is-trading-day --date 2026-07-03
.\go-stock-cli.exe calendar next-trading-day --date 2026-07-03
.\go-stock-cli.exe calendar holiday --date 2026-10-01
.\go-stock-cli.exe calendar holiday-year --year 2026
.\go-stock-cli.exe calendar holiday-batch --dates "2026-07-03,2026-10-01"
```

## 市场行情、指数、宏观

| 工具 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `GetMarketData` | 获取市场整体行情、指数、涨跌分布、新股申购等；若数据源返回全 0，会在输出中提示异常，不应直接当作真实盘面。 | 做每日市场总览、盘中快照。 | 通常无参数。 |
| `GetGlobalMarketStatus` | 获取全球主要股指行情和开盘状态。 | 关注外盘、全球风险偏好，或需要市场状态但不想污染 `GetCurrentTime` 输出。 | `limit`: 返回数量，默认 30。 |
| `GlobalStockIndexesReadable` | 获取全球主要股指行情。 | 兼容旧工具名；新流程优先用 `GetGlobalMarketStatus`。 | 通常无参数。 |
| `GetStockChanges` | 查询当前或近期股票异动。 | 用户问“异动股”“突然拉升/跳水”；也可作为当日本地异动历史为空时的实时兜底数据源。 | `changeTypes`: 异动类型；`pageSize`: 条数。需要概念维度时再用 `GetStockConceptInfo` 补映射。 |
| `GetStockChangeHistoryList` | 查询历史异动列表。 | 复盘某日或某只股票异动。 | 日期、代码、类型等。 |
| `GetDailyChangeStats` | 查询每日异动统计。 | 判断当日市场异动结构。 | 日期。 |
| `GetChangeRank` | 查询异动排行，返回股票、行业、概念三个维度的异动次数排行。 | 找当日/近 N 日异动最活跃股票、行业或概念；用户问“当日异动次数最多的概念”时优先用它。 | `days`: 1 表示当日；`topN`: 返回数量。若本地 `stock_change_history`/`all_stock_info` 为空，改用 `GetStockChanges` 全量异动 + `GetStockConceptInfo` 逐股概念映射后聚合。 |
| `GetDailyDimensionStats` | 按维度统计每日市场数据。 | 做市场维度统计分析。 | 日期、维度。 |
| `GetTypeStatsByDate` | 按日期查询异动类型统计。 | 分析某天异动类别分布。 | 日期。 |
| `QueryIwencai` | 调用同花顺问财自然语言查询。 | 用户提出自然语言市场/股票筛选问题。 | `query` 或自然语言条件。 |
| `QueryZhishu` | 查询指数相关数据。 | 用户问大盘指数、行业指数、核心指数。 | 指数名称或代码。 |
| `QueryMacro` | 查询宏观经济数据。 | GDP、CPI、PMI、社融、利率等问题。 | 宏观指标关键词。 |
| `QueryFutures` | 查询期货/期权相关数据。 | 用户问商品、股指期货、期权行情。 | 品种或条件。 |
| `QueryStockConnect` | 查询沪深港通、北向/南向资金。 | 分析外资流入、港股通成交。 | 日期、通道类型。 |
| `HotspotDiscovery` | 发现市场热点、题材、热门板块；依赖东方财富妙想 `EmApiKey`。 | 用户问“今天热点是什么”，且本地已配置 `EmApiKey`。 | 可用关键词或日期；未配置 Key 时用 `GetUplimitHotPlates`、`GetHotEventList`、`QueryStockNewsTool` 兜底。 |
| `GetIndustryRank` | 查询“行业排名 > 行业涨幅排名”。 | 用户问行业涨幅排名、涨幅最高行业、行业 5 日/20 日涨幅和领涨股。 | `sort`: `0` 涨幅降序、`1` 涨幅升序；`limit`: 前端默认 150。 |
| `GetWallstreetcnMarketReal` | 获取华尔街见闻实时行情。 | 需要外部实时市场数据补充。 | 品种或市场。 |
| `GetWallstreetcnKline` | 获取华尔街见闻 K 线数据。 | 查询外部市场品种走势。 | 品种、周期。 |
| `GetIndustryValuation` | 查询行业估值数据。 | 判断行业估值高低、行业对比。 | 行业/板块名称。 |
| `GetEconomicData` | 查询经济指标数据。 | 宏观数据、经济指标分析。 | 指标、日期范围。 |

## 个股行情与 K 线

| 工具 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `GetStockInfo` | 查询个股实时行情，涨跌额/涨跌幅按当前价和昨收计算，并附带五档盘口概览；可得时同时展示带单位的成交量/成交额、换手率、量比、PE/PB、总市值、流通市值。 | 盯盘首选；看个股价格、涨跌幅、成交量、成交额、盘口概要和基础估值。 | `stockCode`: 股票代码，支持 `sz003026`、`003026.SZ` 等；CLI 推荐 `--stock-code`，也兼容 `--stockCode`、`--stock_code`、`--stockcode`。PowerShell 多股建议写 `--stock-code "sz300308,sz300502"`。 |
| `GetStockOrderBook` | 查询五档盘口，包含买一至买五、卖一至卖五、当前价、涨跌额、涨跌幅、更新时间。若返回空盘口，CLI 会自动尝试 `GetStockInfo` 兜底。 | 判断封单、委托队列、买卖盘深度、为什么委托未成交；盯盘默认先用 `GetStockInfo`。 | `stockCode`: 股票代码，多只用英文逗号分隔；PowerShell 中建议加引号。 |
| `GetStockKLine` | 查询个股 K 线数据。 | 技术走势、历史行情。 | 股票代码、周期、复权、数量。 |
| `GetEastMoneyKLine` | 查询东方财富 K 线。 | 需要东方财富行情源。 | 股票代码、周期、复权；CLI 可用 `--adjust-flag qfq|hfq|none`。 |
| `GetEastMoneyKLineWithMA` | 查询带均线的 K 线，输出列顺序稳定，便于人读和程序解析。 | 做趋势、均线、技术面分析。 | 股票代码、周期、均线参数，如 `maPeriods: 5,10,20,60`；CLI 可用 `--adjust-flag qfq|hfq|none`。若需要自动摘要，优先用 `kline show`。 |
| `kline signals` | CLI 子功能，按 K 线数据计算 GUI “指标信号汇总”口径的看多、看空、震荡、中性统计和逐指标标签。 | 用户问 K线分析页里的“看多/看空/震荡/中性”信号，或 Agent 需要快速判断技术指标共振。 | `--stock-code`: 股票代码；`--k-line-type`: 周期；`--adjust`: qfq/hfq/none；`--limit`: 默认建议 250。 |
| `GetStockMinuteData` | 查询个股分时/分钟数据。 | 盘中走势、短周期波动。 | 股票代码、日期、分钟周期。 |
| `GetStockConceptInfo` | 查询个股所属概念/板块；东方财富概念无数据时会尝试通达信 MAC 板块归属兜底。 | 分析个股题材、板块归属。 | 股票代码；CLI 归档层遇到多只代码会逐只查询并分段输出。 |

## 个股基本面、F10、财务、估值

| 工具 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `GetStockFinancialInfo` | 查询股票财务报表信息。 | 看资产负债、利润、现金流等报表。 | `stockCode`。 |
| `GetStockHolderNum` | 查询股东人数。 | 分析筹码集中度变化。 | 股票代码。 |
| `GetStockHolderTrend` | 查询户均持股趋势。 | 看股东户数、户均持股随时间变化。 | 股票代码。 |
| `GetStockRZRQInfo` | 查询融资融券信息。 | 看杠杆资金、两融余额。 | 股票代码。 |
| `GetStockLatestFinance` | 查询最新财务核心指标。 | 快速看 EPS、ROE、营收、净利润。 | 股票代码；CLI 归档层遇到多只代码会逐只查询并分段输出。 |
| `GetStockQtrMainFinance` | 查询季度主要财务指标。 | 对比季度业绩趋势。 | 股票代码。 |
| `GetStockOrgPredict` | 查询机构预测明细。 | 看券商/机构 EPS、PE 预测。 | 股票代码。 |
| `GetStockPredictSummary` | 查询机构预测汇总。 | 看一致预期和年度预测均值。 | 股票代码。 |
| `GetStockValuationPercentile` | 查询估值百分位。 | 判断 PE/PB 等是否处于历史高低位。 | 股票代码。 |
| `GetStockMarginTrading` | 查询融资融券日度数据。 | 看融资买入、融券卖出、余额变化。 | 股票代码。 |
| `GetStockBlockTrade` | 查询大宗交易。 | 分析机构大额成交和折溢价。 | 股票代码。 |
| `GetStockBillboard` | 查询个股龙虎榜。 | 分析个股上榜原因和资金博弈。 | 股票代码。 |
| `GetStockOperationDeptTrade` | 查询龙虎榜营业部买卖明细。 | 看营业部买入/卖出金额和占比。 | 股票代码。 |
| `GetTdxCompanyInfo` | 查询通达信 F10 公司资料。 | 获取公司概况、财务摘要、除权除息等。 | 股票代码。 |
| `GetTdxCompanyCategory` | 查询通达信 F10 指定分类。 | 精确查看公司概况、股本结构、经营分析等；遇到超时时先不传 `category` 获取分类列表，再单独查一个分类，避免批量连续调用。 | 股票代码、分类名。 |
| `GetTdxFinanceInfo` | 查询通达信财务信息。 | 获取 EPS、净资产、营收、净利润等。 | 股票代码。 |
| `GetTdxXDXRInfo` | 查询通达信除权除息。 | 查分红、配股、送转历史。 | 股票代码。 |
| `GetTdxSymbolBelongBoard` | 查询通达信 MAC 股票所属板块。 | 补充行业板块、概念板块、地域板块、风格板块，以及板块指数、涨停/跌停家数等归属信息。 | `stockCode`: 股票代码，如 `600519.SH`、`000001.SZ`、`00700.HK`。 |
| `QueryBasicInfo` | 查询基础资料。 | 公司基本资料、上市信息等。 | 股票代码或公司名。 |
| `QueryFinance` | 查询财务资料。 | 基本面、财报、财务指标问题。 | 股票代码或查询条件。 |
| `QueryIndustry` | 查询行业资料。 | 个股行业归属、行业资料。 | 股票代码或行业关键词。 |
| `QueryManagement` | 查询管理层资料。 | 董监高、管理层信息。 | 股票代码或公司名。 |
| `QueryFundFinance` | 查询基金财务/业绩资料。 | 基金维度基本面数据。 | 基金代码或关键词。 |
| `QueryBusinessData` | 查询主营业务数据。 | 收入结构、业务构成、客户供应商等。 | 股票代码。 |
| `QueryEvent` | 查询公司事件。 | 重大事项、事件驱动分析。 | 股票代码、关键词。 |
| `StockEarningsReview` | 个股业绩点评。 | 解读财报、业绩快报、业绩变化。 | 股票代码或业绩问题。 |

## 选股与筛选

| 工具 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `FilterStocks` | 按技术形态、关注排名、连涨连跌等条件筛选。 | 结构化条件选股。 | 技术条件、分页、关键词。 |
| `SearchStockByIndicators` | 用自然语言按指标筛选股票。 | 用户用一句话描述选股条件。 | `words`: 自然语言条件。 |
| `SelectAStock` | A 股条件筛选。 | A 股范围内多因子/条件选股。 | 筛选条件。 |
| `SelectSector` | 板块筛选。 | 找强势行业、概念或板块。 | 条件、排序字段。 |
| `SelectETF` | ETF 筛选。 | 找行业 ETF、宽基 ETF、主题 ETF。 | ETF 条件。 |
| `SelectFundManager` | 基金经理筛选。 | 查优秀基金经理、管理规模、业绩。 | 经理条件。 |
| `SelectConvertibleBond` | 可转债筛选。 | 按溢价率、价格、评级筛转债。 | 转债条件。 |
| `SelectFundCompany` | 基金公司筛选。 | 基金公司规模、业绩、排名。 | 公司条件。 |
| `SelectFund` | 基金筛选。 | 按类型、收益、回撤、规模筛基金。 | 基金条件。 |
| `SelectFuturesOption` | 期货/期权筛选。 | 期货期权市场条件筛选。 | 品种、条件。 |
| `SelectHKStock` | 港股筛选。 | 港股市场选股。 | 港股条件。 |
| `SelectUSStock` | 美股筛选。 | 美股市场选股。 | 美股条件。 |
| `SearchBk` | 板块/概念/指数自然语言搜索。 | 查询板块整体数据。 | `words`: 板块/概念描述。 |
| `SearchETF` | ETF 自然语言搜索。 | 查询 ETF 数据。 | `words`: ETF 条件。 |
| `HotStrategyTable` | 查询热门选股策略。 | 用户问当前流行策略。 | 通常无参数。 |
| `HotStockTable` | 查询热门股票排行。 | 用户问热股、人气股。 | `pageSize` 等。 |

## 资金流与交易席位

| 工具 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `GetMoneyRankSina` | 查询新浪个股资金流向排名，对应前端“个股资金流向”的 9 个子标签。 | 用户明确问净流入额、流出资金、净流入率、主力/散户净流入或流出排名。 | `sort`: `netamount`, `outamount`, `ratioamount`, `r0_net`, `r0_out`, `r0_ratio`, `r3_net`, `r3_out`, `r3_ratio`；`limit` 最多 20。 |
| `GetStockMoneyData` | 查询东方财富今日个股资金流入 Top50。 | 找今日主力资金流入个股，但不要求前端 9 个新浪排名标签。 | 通常无参数。 |
| `GetStockHistoryMoneyData` | 查询个股历史资金流；东方财富历史接口无数据时会尝试新浪资金趋势兜底，并在标题中标注来源。默认输出最近 20 条，并提供近 3/5/10 日主力净额、连续净流入天数、最近一日主力净占比摘要。 | 分析某股资金持续性，避免直接阅读 `f62`、`f184` 等原始字段。 | 股票代码，如 `003026.SZ`、`sz003026`；`limit`: 最近条数，默认 20。 |
| `GetAllBKCodes` | 查询“板块资金流向”可用板块代码和名称。 | 调用板块资金流向历史或折线数据前确认 `BK` 代码。 | 通常无参数。 |
| `GetBKFundFlowTopList` | 查询最新快照的板块资金流向排名。 | 用户问顶层“市场行情 > 板块资金流向”的最新排名。 | `topN`: 返回前 N 名。 |
| `GetBKFundFlowTopListByDate` | 查询指定日期最新快照的板块资金流向排名。 | 用户问某天顶层“板块资金流向”排名。 | `date`: `YYYY-MM-DD`；`topN`。 |
| `GetBKFundFlowList` | 查询某个板块的资金流向历史序列。 | 画板块资金流折线或分析资金持续流入/流出。 | `code`: 板块代码；`limit`: 最近记录数。 |
| `GetBKFundFlowListByDate` | 查询某个板块指定日期的资金流向历史序列。 | 分析某天某板块盘中资金流变化。 | `code`: 板块代码；`date`: `YYYY-MM-DD`。 |
| `GetIndustryMoneyRank` | 查询“行业排名”里的资金类子页：行业资金排名、证监会行业资金排名、概念板块资金排名。 | 判断资金流向哪些行业分类、证监会行业或概念；不要把它当作顶层“板块资金流向”。 | `fenlei`: `0` 行业资金排名、`2` 证监会行业资金排名、`1` 概念板块资金排名、`3` 地域板块资金排名；`sort`: `netamount`, `netbuy`, `change`。CLI 命令已映射：`money=0`, `csrc-money=2`, `concept-money=1`。 |
| `GetMutualTop10Deal` | 查询沪股通/深股通/港股通十大成交。 | 分析北向/南向资金偏好。 | 通道类型、交易日期。 |
| `GetLongTigerList` | 查询市场龙虎榜。 | 找当天龙虎榜上榜股票。 | 日期、分页。 |

## 新闻、研报、公告、互动

| 工具 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `QueryStockNewsTool` | 按关键词搜索市场资讯/新闻。 | 快速搜某股、行业、事件新闻。 | `searchWords`。 |
| `GetNewsListData` | 获取新闻列表。 | 查看财经新闻流。 | 类别、分页。 |
| `FinanceSearch` | 综合搜索新闻、公告、研报、政策、交易所动态；依赖东方财富妙想 `EmApiKey`。 | 不确定信息源时优先使用，且已配置 `EmApiKey`。 | `query`: 自然语言搜索；未配置 Key 时用 `QueryStockNewsTool`、`GetStockResearchReport`、`GetStockNotice` 兜底。 |
| `SearchNews` | 搜索新闻。 | 明确只要新闻资讯。 | 关键词。 |
| `SearchReport` | 搜索研报。 | 明确只要研究报告。 | 关键词、公司/行业。 |
| `SearchAnnouncement` | 搜索公告；依赖同花顺问财 `IwencaiApiKey`。 | 明确只要上市公司公告，且已配置 `IwencaiApiKey`。 | 关键词、公司、日期；未配置 Key 时优先用 `GetStockNotice`。 |
| `GetStockResearchReport` | 获取个股研报。 | 深入分析单只股票。 | 股票代码。 |
| `GetSecuritiesCompanyOpinion` | 查询券商观点。 | 看券商近期观点、评级倾向。 | 起止日期等。 |
| `StockNotice` | 查询公告数据。 | 按条件查公告。 | 股票代码、公告类型。 |
| `GetStockNotice` | 获取个股公告，输出公告时间、公告类型、标题、股票代码、公告 ID。 | 查某股公告列表；免问财 Key 的公告兜底工具。 | `stockCodes`: 股票代码列表，逗号分隔。 |
| `InteractiveAnswer` | 查询投资者互动问答。 | 用户问互动易、投资者关系。 | 关键词、分页。 |
| `GetInvestCalendar` | 查询投资日历。 | 财报日、股东大会、IPO、解禁等。 | 日期范围、事件类型。 |
| `GetHotStockList` | 查询热门股票列表。 | 找市场人气股票。 | 榜单类型、数量。 |
| `GetHotEventList` | 查询雪球热门话题/事件。 | 找雪球讨论热度高的事件和题材；不要把它当作前端“当前热门 > 热门话题”。 | `size`: 返回数量。 |
| `GetWallstreetcnLives` | 查询华尔街见闻 7x24 快讯。 | 全球宏观、海外市场快讯。 | 可用关键词或分页。 |
| `GetWallstreetcnCalendar` | 查询华尔街见闻财经日历。 | 海外重要经济数据和事件。 | 日期。 |
| `QueryInsResearch` | 查询机构调研。 | 查上市公司调研活动。 | 股票代码、日期、关键词。 |
| `SearchInvestor` | 搜索投资者关系内容。 | 投关活动、问答、路演等。 | 关键词。 |

## App/Wails 前端专用热门数据

| 入口 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `HotTopic(size)` | 获取前端“当前热门 > 热门话题”。底层为东方财富股吧话题接口 `newtopic/api/Topic/HomePageListRead`。 | 用户明确要求“当前热门”里的“热门话题”，或需要和前端页面完全一致的数据。 | `size`: 返回数量。常用字段：`nickname` 标题、`desc` 描述、`postNumber` 讨论数、`clickNumber` 浏览量、`stock_list` 关联标的、`htid` 话题 ID。链接格式：`https://gubatopic.eastmoney.com/topic_v3.html?htid={htid}`。 |
| `HotEvent(size)` | 获取 App 侧热门事件。 | 前端或 Wails 上下文需要热门事件数据。 | `size`: 返回数量。 |
| `HotStock(marketType)` | 获取 App 侧当前热门股票。 | 前端“当前热门”股票榜。 | `marketType`: `10` 全球、`12` 沪深、`13` 港股、`11` 美股。 |

## 涨停、连板、题材热点

| 工具 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `GetUplimitLadder` | 查询涨停梯队、连板高度。 | 分析短线情绪和连板结构。 | 日期。若通过 `research uplimit` 调用且接口返回空或 0，CLI 会附带 `GetMarketData` 市场总览作为交叉验证。 |
| `GetUplimitHotPlates` | 查询涨停热门板块。 | 找涨停集中方向。 | 日期。 |
| `GetUplimitHotStocks` | 查询涨停热门个股。 | 找涨停核心人气股。 | 日期。 |
| `GetUplimitExplodedStocks` | 查询炸板/破板股票。 | 识别分歧、风险和资金撤退。 | 日期。 |
| `GetUplimitPlateStocks` | 查询涨停板块内个股。 | 分析某涨停板块的成分和强弱。 | 日期、板块。 |
| `HotspotDiscovery` | 发现市场热点；依赖东方财富妙想 `EmApiKey`。 | 题材热度、主线识别，且已配置 `EmApiKey`。 | 关键词或日期。 |

## 基金

| 工具 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `SearchFund` | 搜索基金。 | 用户给基金名称、简称或代码。 | 基金关键词。 |
| `GetFundInfo` | 查询基金基础信息。 | 基金类型、规模、经理、费率等。 | 基金代码。 |
| `GetFundKLine` | 查询基金 K 线/走势。 | 看基金净值走势。 | 基金代码、周期、数量。 |
| `GetFundHistoryNetValue` | 查询基金历史净值。 | 分析收益、回撤、净值变化。 | 基金代码、日期范围。 |
| `GetFundTop10Holdings` | 查询基金前十大持仓。 | 看基金持仓风格和集中度。 | 基金代码。 |

## 研究分析

| 工具 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `FinancialQA` | 金融问答。 | 用户问金融概念、数据解释。 | 问题文本。 |
| `ComparableCompanyAnalysis` | 可比公司分析。 | 比较同业估值和财务指标。 | 公司名或股票代码。 |
| `IndustryResearch` | 东方财富 AI 行业研究生成能力，当前不作为 CLI 数据能力开放。 | 不建议调用。 | 改用 `GetIndustryValuation`、`GetSecuritiesCompanyOpinion`、`QueryStockNewsTool` 等只读工具。 |
| `TrackingReport` | 跟踪报告。 | 个股或行业持续跟踪。 | 对象名称或代码。 |
| `FinanceDataQuery` | 金融数据自然语言查询。 | 泛化金融数据查询。 | `query`: 自然语言问题。 |

## CLI 受控写入命令

这不是只读数据能力。原 `SetFollowedStockPosition` 已归档为 CLI 命令 `portfolio position set`，不再作为 raw tool 直连。

| 命令 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `portfolio position set` | 设置本地自选股持仓和提醒字段：成本价、持仓数量、开仓价、止盈价、止损价、涨跌提醒、股价提醒、排序。第一次调用只生成预览和 `confirmToken`，不会写入；第二次必须带确认 token 才写入。未显式传入 `sort` 时，新关注/新建持仓默认排序为 `99`。 | 用户把持仓告诉 Agent，希望 Agent 自动设置成本和提醒；用户实际买卖后也要用它更新本地数量。Agent 可自行判断止损、止盈、涨跌提醒，但需要二次确认。 | 必填：`stockCode`, `costPrice`, `volume`。常用可选：`stockName`, `entryPrice`, `takeProfitPrice`, `stopLossPrice`, `alarmChangePercent`, `alarmPrice`, `sort`, `reason`, `confirm`, `confirmToken`。确认推荐用 CLI flag `--confirm`。 |

## CLI 自选分组命令

这些命令对齐 GUI 的“股票自选 > 分组管理”。只在用户明确要求管理自选分组时使用。

| 命令 | 作用 | 适合什么时候用 | 常见输入提示 |
|---|---|---|---|
| `portfolio group list` | 查看当前自选分组 ID、名称、排序。 | 需要确认分组 ID，或用户问有哪些自选分组。 | 无参数。 |
| `portfolio group add` | 新增自选分组。 | 用户要求创建观察池、短线、长线、基金替代等分组。 | `name`: 分组名称；`sort`: 可选排序。 |
| `portfolio group rename` | 修改已有分组名称。 | 远程 GUI 新增的“分组名称修改”能力；用户要求把某个分组改名。 | `groupId`: 分组 ID；`name` 或 `newName`: 新名称。 |
| `portfolio group assign` | 把股票加入指定分组。 | 用户要求把某只自选股归入某个分组。 | `groupId`: 分组 ID；`stockCode`: 股票代码。 |
| `portfolio group remove` | 从指定分组移出股票。 | 用户要求把某只股票从某个分组移出。 | `groupId`: 分组 ID；`stockCode`: 股票代码；`stockName`: 可选。 |

推荐流程：

1. 用 `QueryStockCodeInfo` 确认股票代码和名称。
2. 用 `GetFollowedStocks` 读取当前自选股成本和提醒字段。
3. Agent 根据用户持仓、成本、风险偏好或近期波动提出 `takeProfitPrice`、`stopLossPrice`、`alarmChangePercent`、`alarmPrice`，并在 `reason` 中说明依据。
4. 先调用 `portfolio position set`，不确认写入，获取预览和确认令牌。
5. 把预览复述给用户；只有用户明确同意后，才用完全相同参数加 `--confirm`、`confirmToken=<预览令牌>` 再次调用。

## 默认不开放或不建议调用

这些工具有写入、通知、推荐记录创建或配置管理副作用，不应作为财经 CLI raw 能力开放给外部 Agent：

| 工具 | 原因 |
|---|---|
| `SetTradingPrice` | 设置交易价格/预警，有写入副作用。 |
| `SendDingDingMessage` | 发送钉钉消息，有外部通知副作用。 |
| `SendToDingDing` | 发送钉钉消息，有外部通知副作用。 |
| `CreateAiRecommendStocks` | 创建 AI 推荐记录，有写入副作用。 |
| `BatchCreateAiRecommendStocks` | 批量创建 AI 推荐记录，有写入副作用。 |
| `AiRecommendStocks` | AI 推荐记录工具不使用。 |
| `GetAIAnalysisHistory` | AI 分析历史工具不使用。 |
| `GetAIAnalysisDetail` | AI 分析详情工具不使用。 |
| `GetAIAnalysisContent` | AI 分析正文工具不使用。 |
| `ListMCPServers` / `CreateMCPServer` / `UpdateMCPServer` / `DeleteMCPServer` / `EnableMCPServer` / `TestMCPServer` | 内部 MCP 服务配置管理，不是财经数据工具。 |
| `ListSkills` / `CreateSkill` / `UpdateSkill` / `DeleteSkill` / `EnableSkill` | 内部 Skill 配置管理，不是财经数据工具。 |
