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
go run ./cmd/go-stock-cli portfolio position set --stock-code 600237 --cost-price 12.56 --volume 300
go run ./cmd/go-stock-cli fund ranking --page-size 20
go run ./cmd/go-stock-cli kline show --stock-code 002335 --k-line-type day --limit 120
```

`market major-index` 不带参数时返回适合盯盘的市场总览；传 `--name` 或 `--code` 时查询单个指数 K 线。

## 归档工具层

所有允许迁移的原对外 MCP 工具都已归档到 CLI，可以通过 `tool <原工具名>` 调用：

```powershell
go run ./cmd/go-stock-cli tool list
go run ./cmd/go-stock-cli tool info --name GetStockOrderBook
go run ./cmd/go-stock-cli tool GetStockOrderBook --stock-code 600237
go run ./cmd/go-stock-cli tool GetStockLatestFinance --stock-code sz002335,sz002506,sh603690
```

CLI 也支持 kebab 写法：

```powershell
go run ./cmd/go-stock-cli tool get-stock-order-book --stock-code 600237
```

`tool info` 会从原 Eino `ToolInfo.ParamsOneOf` 读取 JSON Schema，因此参数说明和原工具定义保持一致。

F10/概念等单股语义工具遇到多只股票时会自动逐只查询并分段输出，避免底层接口把多代码合并成不可读结果。

## 写入边界

长期 CLI 主入口只允许一个受控写入路径：

```powershell
go run ./cmd/go-stock-cli portfolio position set --stock-code 600237 --cost-price 12.56 --volume 300
```

持仓、成本、数量、止损、止盈、涨跌提醒、股价提醒必须走预览/确认令牌流程。原写入/通知/配置工具不会作为 raw `tool <name>` 入口开放。

默认禁用的 raw 工具包括：

- `SetTradingPrice`
- `SendDingDingMessage`
- `SendToDingDing`
- `CreateAiRecommendStocks`
- `BatchCreateAiRecommendStocks`
- `AiRecommendStocks`
- `SetFollowedStockPosition`，已迁移到 `portfolio position set`
