# Agent Instructions

## go-stock CLI + Skill

This project exposes stock, market, research, fund, K-line, portfolio, and trading-calendar data through a GUI-aligned local CLI. The previous project-local outbound MCP server has been archived; do not restart or depend on it.

Before using go-stock data tools, read:

- `skills/go-stock/SKILL.md`
- `skills/go-stock/references/tool-catalog.md` when exact tool purpose or inputs matter
- `docs/go-stock-cli.md`

Use the local CLI from the repository root:

```powershell
.\go-stock-cli.exe help
.\go-stock-cli.exe market news
.\go-stock-cli.exe tool list
```

Default to the checked-in `go-stock-cli.exe`. If the exe is missing or stale, run `.\scripts\build-windows.ps1` to regenerate it. The source wrapper `.\scripts\go-stock-cli.ps1` remains available only as a fallback; it sets `GOCACHE` to the repo-local `.gocache` directory before running from source.

All migrated archived tools are available through the CLI compatibility layer:

```powershell
.\go-stock-cli.exe tool info --name GetStockOrderBook
.\go-stock-cli.exe tool GetStockOrderBook --stock-code 600237
```

Do not expose or call write/notification/configuration tools such as `SetTradingPrice`, `SendDingDingMessage`, `SendToDingDing`, `CreateAiRecommendStocks`, `BatchCreateAiRecommendStocks`, `AiRecommendStocks`, or MCP/Skill management tools.

The only allowed write path is portfolio/position management through `portfolio position set`. It must follow the preview/confirm-token flow: first call without confirmation, show the preview to the user, wait for explicit confirmation, then call again with the same arguments plus confirmation and the matching `confirmToken`.
