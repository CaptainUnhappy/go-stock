# go-stock CLI

`go-stock-cli` 是 go-stock 对 Agent 和本机脚本的主入口。GUI 菜单对齐功能和原对外 MCP 工具都已迁移到 CLI 中；项目本地对外 MCP 服务已归档，不再作为集成方式。

## 启动方式

```powershell
go run ./cmd/go-stock-cli help
```

在沙箱环境或 Codex 里，优先使用仓库脚本，它会把 Go 构建缓存放到项目 `.gocache`，避免 `C:\Users\...\AppData\Local\go-build` 权限问题：

```powershell
.\scripts\go-stock-cli.ps1 help
.\scripts\go-stock-cli.ps1 market news
```

可选环境变量：

```text
GO_STOCK_DB=可选；不设置时默认读取系统用户目录下的 go-stock/data/stock.db
GO_STOCK_CLI_VERBOSE=true；打开 CLI 调试日志，默认关闭以避免污染 stdout
```

输出格式：

```powershell
go run ./cmd/go-stock-cli --json market major-index
```

## GUI 对齐命令

```powershell
go run ./cmd/go-stock-cli market news
go run ./cmd/go-stock-cli market major-index
go run ./cmd/go-stock-cli market major-index --name 上证指数
go run ./cmd/go-stock-cli market industry-rank concept-money --sort netamount --limit 20
go run ./cmd/go-stock-cli market money-flow stock --sort r0_net --limit 20
go run ./cmd/go-stock-cli portfolio list
go run ./cmd/go-stock-cli portfolio group rename --group-id 1 --new-name 短线观察
go run ./cmd/go-stock-cli portfolio position set --stock-code 600237 --cost-price 12.56 --volume 300
go run ./cmd/go-stock-cli fund ranking --page-size 20
go run ./cmd/go-stock-cli kline show --stock-code 002335 --k-line-type day --adjust qfq --limit 120
go run ./cmd/go-stock-cli kline signals --stock-code 002335 --k-line-type day --adjust qfq --limit 250
```

`market major-index` 不带参数时返回适合盯盘的市场总览；传 `--name` 或 `--code` 时查询单个指数 K 线。

## 归档工具层

所有允许迁移的原对外 MCP 工具都已归档到 CLI，可以通过 `tool <原工具名>` 调用：

```powershell
go run ./cmd/go-stock-cli tool list
go run ./cmd/go-stock-cli tool info --name GetStockInfo
go run ./cmd/go-stock-cli tool GetStockInfo --stock-code 600237
go run ./cmd/go-stock-cli tool GetStockInfo --stock-code "sz300308,sz300502"
go run ./cmd/go-stock-cli tool GetStockLatestFinance --stockCode='sz002335,sz002506,sh603690'
```

CLI 也支持 kebab 写法：

```powershell
go run ./cmd/go-stock-cli tool get-stock-order-book --stock-code 600237
```

`tool info` 会从原 Eino `ToolInfo.ParamsOneOf` 读取 JSON Schema，因此参数说明和原工具定义保持一致；对常见股票参数会额外展示推荐 CLI 参数名，例如 `stockCode` 推荐写成 `--stock-code`。

F10/概念等单股语义工具遇到多只股票时会自动逐只查询并分段输出，避免底层接口把多代码合并成不可读结果。

### raw tool 参数和盯盘注意事项

- `--stockCode`、`--stock-code`、`--stock_code`、`--stockcode` 都会归一为 `stockCode`。如果股票参数缺失，`GetStockInfo` / `GetStockOrderBook` 会直接报参数错误，不再伪装成“行情源无数据”。
- PowerShell 多股票参数建议加引号，例如 `--stock-code "sh600237,sz002335"`；CLI 也会尽量兼容被拆开的裸股票代码参数。更稳妥时可用 `--args-json '{"stockCode":"sh600237,sz002335"}'`。
- 盯盘优先用 `tool GetStockInfo`，它包含行情、带单位的成交量/成交额、可得的换手率/量比/PE/PB/市值字段和五档盘口概览；`GetStockOrderBook` 是专用盘口工具，返回空盘口时 CLI 会自动尝试用 `GetStockInfo` 兜底。
- `GetStockHistoryMoneyData` 默认只输出最近 20 条，并附带近 3/5/10 日主力净额、连续净流入天数和最近一日主力净占比摘要；可用 `--limit` 调整。
- `kline show` 会在 K 线表后追加当前价相对 MA5/10/20/60、近 5/20 根涨幅和量能摘要。日K/周K/月K/季K/年K默认 `--adjust qfq` 前复权，可传 `--adjust hfq` 后复权或 `--adjust none` 不复权；分钟线会忽略复权参数。
- `kline signals` 会输出 GUI K线分析页“指标信号汇总”口径的看多、看空、震荡、中性统计和逐指标标签，同样支持 `--adjust qfq|hfq|none`。
- `market hot cn/hk/us/global` 返回热门股票榜，对应 `GetHotStockList` 的沪深/港股/美股/全球市场类型；不要用 `market major-index` 或 `GetMarketData` 替代热门股。
- `GetStockLatestFinance` 等单股语义工具遇到多股票输入时，CLI 会逐只拆分调用并分段输出。
- 15:00 收盘后盘口仍可能返回最近快照，只能按收盘附近快照理解，不代表仍可成交。

## 写入边界

长期 CLI 主入口只允许一个受控写入路径：

```powershell
go run ./cmd/go-stock-cli portfolio position set --stock-code 600237 --cost-price 12.56 --volume 300
```

持仓、成本、数量、止损、止盈、涨跌提醒、股价提醒必须走预览/确认令牌流程。原写入/通知/配置工具不会作为 raw `tool <name>` 入口开放。

`portfolio list` 是 go-stock 本地自选/持仓元数据，不会自动同步真实券商交易。如果用户实际买入或卖出，需要通过 `portfolio position set` 预览并二次确认后更新本地成本、数量和提醒字段。确认时使用 `--confirm` 作为 flag，也兼容 `--confirm true`，但推荐只写 `--confirm`。

未显式传入 `--sort` 时，新关注/新建持仓记录默认排序为 `99`；明确更新持仓排序时，可按当前持有成本从高到低传入 `1/2/3...`。

分组管理命令用于对齐 GUI 的自选分组能力：

```powershell
go run ./cmd/go-stock-cli portfolio group list
go run ./cmd/go-stock-cli portfolio group add --name 短线观察 --sort 1
go run ./cmd/go-stock-cli portfolio group rename --group-id 1 --new-name 趋势观察
go run ./cmd/go-stock-cli portfolio group assign --group-id 1 --stock-code 600237
go run ./cmd/go-stock-cli portfolio group remove --group-id 1 --stock-code 600237
```

默认禁用的 raw 工具包括：

- `SetTradingPrice`
- `SendDingDingMessage`
- `SendToDingDing`
- `CreateAiRecommendStocks`
- `BatchCreateAiRecommendStocks`
- `AiRecommendStocks`
- `SetFollowedStockPosition`，已迁移到 `portfolio position set`
