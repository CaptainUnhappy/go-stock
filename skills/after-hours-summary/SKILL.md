---
name: after-hours-summary
description: Generate a Chinese A-share after-hours market summary image from local go-stock CLI market data. Use when the user asks for 盘后总结, 收盘复盘, A股盘后图, 每日市场复盘图片, or wants market/index/industry/theme/fund-flow data rendered as HTML and converted to a PNG/JPEG image, especially when the final deliverable should be only the image.
---

# After Hours Summary

## Workflow

1. Read the repository CLI instructions named in `AGENTS.md` if they are not already loaded.
2. Use `.\scripts\go-stock-cli.ps1 ...` from the repository root in Codex/sandbox sessions. Direct `go run ./cmd/go-stock-cli ...` is also valid when `GOCACHE` is writable.
3. Use only read-only CLI commands. Do not call write, notification, recommendation-record, MCP-management, or skill-management tools.
4. Collect the report data with the CLI call plan below.
5. Normalize the result to the JSON shape in `references/report-schema.md`.
6. Render the card-style HTML and PNG:

```powershell
python skills/after-hours-summary/scripts/render_after_hours_summary.py --input <summary.json> --output-dir outputs/after-hours-summary
```

7. Reply with the PNG path only unless the user asks for details. The HTML is an intermediate artifact.

## CLI Call Plan

Use the latest intended trading date. If the user gives a date, use that date. If not, use `tool GetCurrentTime` and `tool IsTradingDay`; after market close use today, otherwise use the latest available trading-day data.

Market and indices:
- `market news` or `tool GetMarketData` for major indices, turnover,涨跌家数,涨跌停家数, market breadth, and short market conclusion.
- `market global-index`, `tool GlobalStockIndexesReadable`, and `tool QueryStockConnect` only when the report needs outside-market or northbound context.

Industry, concepts, and fund flow:
- `market industry-rank gain` or `tool GetIndustryRank` for industry涨幅排行.
- `market industry-rank money` / `market industry-rank concept-money`, or `tool GetIndustryMoneyRank` with `fenlei=0` for industry fund ranking and `fenlei=1` for concept fund ranking.
- `market money-flow bk date/latest` or `tool GetBKFundFlowTopListByDate` / `tool GetBKFundFlowTopList` for top-level板块资金流向.

Themes and market activity:
- `tool GetChangeRank --days 1 --topN ...` for active concepts/industries/stocks.
- If `GetChangeRank` is empty, use `tool GetStockChanges` and, when concept aggregation is needed, map stocks with `tool GetStockConceptInfo`.
- `tool HotspotDiscovery`, `tool GetHotEventList`, `tool GetUplimitHotPlates`, `tool GetUplimitHotStocks`, `research uplimit`, `tool GetUplimitLadder`, and `tool GetUplimitExplodedStocks` for main-line themes, limit-up structure, divergence, and market emotion.

Hot topics and catalysts:
- Use `tool GetHotEventList` and `tool HotspotDiscovery` first for current 热门话题 / 热门事件.
- Use `tool FinanceSearch` only to supplement evidence behind a selected hot topic.
- Use `tool SearchReport`, `tool SearchAnnouncement`, or `tool SearchNews` only when the source type is explicit.

Optional holdings impact:
- If the user supplies holdings/watchlist, confirm each code with `portfolio search` or `tool QueryStockCodeInfo`, then use `tool GetStockInfo`, `tool GetStockConceptInfo`, and fund-flow/announcement/news tools as needed.

## Writing Rules

- Keep conclusions probabilistic: use "可能", "偏", "需要观察", "风险在于"; do not give deterministic buy/sell instructions.
- Separate observation from inference. Put raw data in tables and inference in short `逻辑/明日含义/失败信号` fields.
- Make the top conclusion one sentence: market style, main line, weak line, and next-session focus.
- Prefer short table cells so the rendered image stays readable.
- Include actual upstream data/source names in the final "来源" section, not CLI command names. Examples: 东方财富行情、东方财富资金流、同花顺热门题材、财联社资讯、巨潮资讯公告、交易所公告。

## Visual Output

Use `render_after_hours_summary.py` for layout. The renderer:
- Writes a dark dashboard HTML file.
- Converts the HTML to a full-page PNG using Playwright when available.
- Falls back to local Chrome/Edge headless screenshot if Playwright is unavailable.

The visual style should resemble the provided reference: dark navy background, compact information-dense cards, red/green market numbers, table-heavy sections, and a strong top "收盘结论" strip.
