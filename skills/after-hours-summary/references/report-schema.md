# After-hours summary JSON schema

Create one normalized JSON file before rendering. Fields are permissive; missing optional sections render as empty or are skipped.

```json
{
  "date": "2026-06-18",
  "title": "2026-06-18 A股盘后总结",
  "subtitle": "Data Time 2026-06-18 15:30 Asia/Shanghai | 数据来自本地只读行情源",
  "verdict": "收盘结论：今天市场偏轮动，主线集中在存储芯片/PCB/机器人，明日观察量能与主线延续。",
  "market_indices": [
    {
      "name": "上证指数",
      "close": "4108.08",
      "change": "+0.40%",
      "turnover": "1.40万亿",
      "breadth": "785 / 1534 / 28",
      "note": "指数震荡偏强"
    }
  ],
  "industry_inflow": [
    { "name": "电子", "change": "+3.25%", "money": "+259.08亿", "logic": "全天强势资金合流" }
  ],
  "industry_outflow": [
    { "name": "有色金属", "change": "-1.06%", "money": "-99.17亿", "logic": "避险资金回撤" }
  ],
  "themes": [
    {
      "rank": "1",
      "name": "存储芯片 / 半导体",
      "strength": "涨停8家，主力+170亿",
      "leaders": "兆易创新、香农芯创、德明利",
      "status": "主线确认",
      "meaning": "明天看分歧后承接"
    }
  ],
  "hot_topics": [
    {
      "direction": "存储芯片",
      "time": "6月17日13:04",
      "source": "同花顺热门题材 / 财联社资讯",
      "evidence": "涨价/供给收缩预期",
      "content": "DRAM/NAND 产业链延续景气修复。",
      "validation": "板块放量，核心股涨停"
    }
  ],
  "lineup": [
    {
      "theme": "大消费细分",
      "front": "香农芯创、兆易创新",
      "middle": "涨停梯队",
      "back": "东方+31.75亿，兆易+25.56亿",
      "review": "核心个股承接决定持续性"
    }
  ],
  "divergence": [
    {
      "event": "玻璃基板",
      "direction": "政策消息",
      "intensity": "强",
      "key_stocks": "中国巨石、旗滨集团",
      "next_watch": "高开强度与承接"
    }
  ],
  "holdings": [
    {
      "stock": "瑞芯微 603893",
      "change": "+1.20%",
      "money": "+1.3亿",
      "reason": "半导体边缘受益",
      "next": "观察放量突破"
    }
  ],
  "tomorrow": [
    {
      "direction": "存储芯片",
      "watch": "兆易创新、香农芯创、德明利",
      "success": "前排高开不炸，中军放量承接",
      "failure": "高开低走，后排补跌"
    }
  ],
  "risk_status": "截至本报告生成时，亮点是主线资金集中；风险是题材轮动过快与量能不足。",
  "sources": [
    "东方财富行情",
    "东方财富资金流",
    "新浪资金流",
    "同花顺热门题材",
    "财联社资讯",
    "巨潮资讯公告"
  ]
}
```

## Section mapping

| JSON field | Rendered section |
|---|---|
| `market_indices` | 指数与市场风格 |
| `industry_inflow`, `industry_outflow` | 行业主力净流入 / 净流出 |
| `themes` | 概念与主线排序 |
| `hot_topics` | 热门话题与催化 |
| `lineup` | 前排/中军/后排 |
| `divergence` | 分歧监测 |
| `holdings` | 持仓影响 |
| `tomorrow` | 明日锚点与失败信号 |
| `risk_status` | 龙虎机制状态 |
| `sources` | 来源 |

## Normalization guidance

- Use strings for displayed numbers so units stay explicit.
- Prefix positive values with `+` and negative values with `-`; the renderer colors them automatically.
- Keep each text field under roughly 30 Chinese characters when possible.
- Omit `holdings` if the user did not provide positions or a watchlist.
- Prefer `hot_topics` over legacy `catalysts`; keep `catalysts` only for backward compatibility.
- `sources` must be actual upstream source names, not method/tool names.
