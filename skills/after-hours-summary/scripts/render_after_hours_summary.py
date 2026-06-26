#!/usr/bin/env python3
"""Render an after-hours A-share summary JSON file to HTML and PNG."""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
from datetime import date
from html import escape
from pathlib import Path
from typing import Any, Iterable


def sample_data() -> dict[str, Any]:
    today = date.today().isoformat()
    return {
        "date": today,
        "title": f"{today} A股盘后总结",
        "subtitle": f"Data Time {today} 15:30 Asia/Shanghai | 数据来自本地只读行情源",
        "verdict": "收盘结论：今天题材强于指数，资金集中在存储芯片、PCB、机器人，明日重点看量能与前排承接。",
        "market_indices": [
            {"name": "上证指数", "close": "4108.08", "change": "+0.40%", "turnover": "1.40万亿", "breadth": "785 / 1534 / 28", "note": "指数震荡偏强"},
            {"name": "深证成指", "close": "15880.95", "change": "+1.31%", "turnover": "1.69万亿", "breadth": "841 / 2055 / 25", "note": "成长风格占优"},
            {"name": "创业板指", "close": "4167.05", "change": "+1.56%", "turnover": "8304亿", "breadth": "476 / 912 / 8", "note": "半导体链修复"},
            {"name": "沪深300", "close": "4931.39", "change": "+0.97%", "turnover": "8504亿", "breadth": "136 / 161 / 3", "note": "权重同步回暖"},
        ],
        "industry_inflow": [
            {"name": "电子", "change": "+3.25%", "money": "+259.08亿", "logic": "全天强势资金合流"},
            {"name": "半导体", "change": "+4.13%", "money": "+154.09亿", "logic": "早盘分歧后修复"},
            {"name": "消费电子", "change": "+3.30%", "money": "+65.60亿", "logic": "AI端侧带动"},
            {"name": "通信设备", "change": "+4.76%", "money": "+61.60亿", "logic": "光模块延续强势"},
        ],
        "industry_outflow": [
            {"name": "有色金属", "change": "-1.06%", "money": "-99.17亿", "logic": "避险资金回撤"},
            {"name": "小金属", "change": "-2.45%", "money": "-67.68亿", "logic": "涨价线兑现"},
            {"name": "基础化工", "change": "-0.62%", "money": "-55.05亿", "logic": "周期资金转弱"},
            {"name": "汽车", "change": "-0.96%", "money": "-29.66亿", "logic": "板块承压"},
        ],
        "themes": [
            {"rank": "1", "name": "存储芯片 / 半导体", "strength": "涨停8家，主力+170亿", "leaders": "兆易创新、香农芯创、德明利", "status": "主线确认", "meaning": "明天看分歧后承接"},
            {"rank": "2", "name": "PCB / CCL", "strength": "涨停6家，资金+61亿", "leaders": "景旺电子、胜宏科技、沪电股份", "status": "强趋势", "meaning": "前排低吸比追高稳"},
            {"rank": "3", "name": "机器人", "strength": "+4.94%，主力+55亿", "leaders": "中大力德、柯力传感、绿的谐波", "status": "确认回流", "meaning": "看中军持续性"},
            {"rank": "4", "name": "800V / 电力设备", "strength": "电力设备+97亿", "leaders": "英诺激光、阿特斯、阳光电源", "status": "轮动修复", "meaning": "偏低位补涨"},
        ],
        "hot_topics": [
            {"direction": "存储芯片", "time": "13:04", "source": "同花顺热门题材 / 财联社资讯", "evidence": "供需改善", "content": "DRAM/NAND 价格修复预期升温。", "validation": "板块放量，核心股涨停"},
            {"direction": "PCB/CCL", "time": "午后", "source": "同花顺热门题材 / 行业资讯", "evidence": "AI服务器需求", "content": "高端板材订单景气度延续。", "validation": "趋势股继续新高"},
            {"direction": "机器人", "time": "盘中", "source": "热门事件 / 市场异动", "evidence": "事件催化", "content": "人形机器人链条扩散。", "validation": "前排承接强于后排"},
        ],
        "lineup": [
            {"theme": "存储芯片", "front": "香农芯创、德明利", "middle": "兆易创新、澜起科技", "back": "设备/封测补涨", "review": "核心股不走弱则主线仍在"},
            {"theme": "PCB链", "front": "景旺电子、胜宏科技", "middle": "沪电股份、生益科技", "back": "覆铜板扩散", "review": "趋势股承接仍好"},
            {"theme": "机器人", "front": "中大力德、柯力传感", "middle": "绿的谐波", "back": "零部件轮动", "review": "需要量能配合"},
        ],
        "divergence": [
            {"event": "高位题材", "direction": "存储芯片", "intensity": "中等", "key_stocks": "前排继续强，后排分化", "next_watch": "高开后承接"},
            {"event": "周期资源", "direction": "小金属/有色", "intensity": "偏弱", "key_stocks": "资金净流出居前", "next_watch": "止跌信号"},
        ],
        "tomorrow": [
            {"direction": "存储芯片", "watch": "兆易创新、香农芯创、德明利", "success": "前排高开不炸，中军放量承接", "failure": "高开低走，后排补跌"},
            {"direction": "PCB/CCL", "watch": "景旺电子、胜宏科技、沪电股份", "success": "趋势股继续沿均线上行", "failure": "冲高回落且放量"},
            {"direction": "机器人", "watch": "中大力德、柯力传感", "success": "前排晋级，中军补量", "failure": "只剩后排脉冲"},
        ],
        "risk_status": "截至本报告生成时，主线集中度较高；风险在于题材轮动过快、量能不足和高位股分歧扩大。",
        "sources": ["东方财富行情", "东方财富资金流", "新浪资金流", "同花顺热门题材", "财联社资讯", "巨潮资讯公告"],
    }


def as_list(value: Any) -> list[dict[str, Any]]:
    if isinstance(value, list):
        return [x for x in value if isinstance(x, dict)]
    return []


def text(value: Any, default: str = "") -> str:
    if value is None:
        return default
    return str(value)


def cls_for(value: Any) -> str:
    s = text(value).strip()
    if s.startswith("+") or "净流入" in s:
        return "num pos"
    if s.startswith("-") or "净流出" in s:
        return "num neg"
    return "num"


def e(value: Any) -> str:
    return escape(text(value), quote=True)


def table(headers: list[str], rows: Iterable[Iterable[Any]], widths: list[str] | None = None) -> str:
    width_attr = ""
    if widths:
        cols = "".join(f'<col style="width:{escape(w)}">' for w in widths)
        width_attr = f"<colgroup>{cols}</colgroup>"
    head = "".join(f"<th>{escape(h)}</th>" for h in headers)
    body_parts = []
    for row in rows:
        cells = []
        for cell in row:
            value = text(cell)
            cells.append(f'<td class="{cls_for(value)}">{escape(value)}</td>')
        body_parts.append("<tr>" + "".join(cells) + "</tr>")
    return f"<table>{width_attr}<thead><tr>{head}</tr></thead><tbody>{''.join(body_parts)}</tbody></table>"


def section(title: str, inner: str, accent: str = "blue", half: bool = False) -> str:
    half_cls = " half" if half else ""
    return f'<section class="card {accent}{half_cls}"><h2>{escape(title)}</h2>{inner}</section>'


def render_html(data: dict[str, Any]) -> str:
    title = text(data.get("title") or f"{data.get('date', '')} A股盘后总结").strip()
    subtitle = text(data.get("subtitle") or "数据来自本地只读行情源").strip()
    verdict = text(data.get("verdict") or "收盘结论：数据不足，需补充市场总览、行业资金和热点证据。")

    market = table(
        ["指数", "收盘", "涨跌", "成交额", "上涨/下跌/平盘", "备注"],
        ([r.get("name"), r.get("close"), r.get("change"), r.get("turnover"), r.get("breadth"), r.get("note")] for r in as_list(data.get("market_indices"))),
        ["15%", "13%", "10%", "14%", "18%", "30%"],
    )

    inflow = table(
        ["行业", "涨跌", "主力净流入", "意义"],
        ([r.get("name"), r.get("change"), r.get("money"), r.get("logic")] for r in as_list(data.get("industry_inflow"))),
        ["22%", "15%", "22%", "41%"],
    )
    outflow = table(
        ["行业", "涨跌", "主力净流出", "意义"],
        ([r.get("name"), r.get("change"), r.get("money"), r.get("logic")] for r in as_list(data.get("industry_outflow"))),
        ["22%", "15%", "22%", "41%"],
    )
    themes = table(
        ["排序", "主题", "强度/资金", "核心标的", "状态", "明日含义"],
        ([r.get("rank"), r.get("name"), r.get("strength"), r.get("leaders"), r.get("status"), r.get("meaning")] for r in as_list(data.get("themes"))),
        ["7%", "21%", "18%", "26%", "12%", "16%"],
    )
    hot_topic_rows = as_list(data.get("hot_topics")) or as_list(data.get("catalysts"))
    hot_topics = table(
        ["方向", "时间", "来源", "证据强度", "内容", "盘面验证"],
        ([r.get("direction"), r.get("time"), r.get("source"), r.get("evidence"), r.get("content"), r.get("validation")] for r in hot_topic_rows),
        ["14%", "12%", "15%", "16%", "27%", "16%"],
    )
    lineup = table(
        ["主题", "前排", "中军", "后排", "判断"],
        ([r.get("theme"), r.get("front"), r.get("middle"), r.get("back"), r.get("review")] for r in as_list(data.get("lineup"))),
        ["15%", "24%", "22%", "19%", "20%"],
    )
    divergence = table(
        ["事件", "方向", "强度", "关键股票/证据", "明日观察"],
        ([r.get("event"), r.get("direction"), r.get("intensity"), r.get("key_stocks"), r.get("next_watch")] for r in as_list(data.get("divergence"))),
        ["18%", "17%", "12%", "33%", "20%"],
    )
    holdings_rows = as_list(data.get("holdings"))
    holdings = ""
    if holdings_rows:
        holdings = section(
            "持仓影响",
            table(
                ["持仓", "收盘涨跌", "主力资金", "逻辑", "明日策略"],
                ([r.get("stock"), r.get("change"), r.get("money"), r.get("reason"), r.get("next")] for r in holdings_rows),
                ["19%", "13%", "15%", "31%", "22%"],
            ),
            "blue",
            True,
        )
    tomorrow = table(
        ["方向", "明日锚点", "继续信号", "失败信号"],
        ([r.get("direction"), r.get("watch"), r.get("success"), r.get("failure")] for r in as_list(data.get("tomorrow"))),
        ["15%", "30%", "28%", "27%"],
    )
    sources = data.get("sources") or []
    if isinstance(sources, list):
        source_items = []
        for source in sources:
            if isinstance(source, dict):
                source_items.append(source.get("name") or source.get("source") or source.get("label") or "")
            else:
                source_items.append(source)
        source_text = " / ".join(escape(text(s)) for s in source_items if text(s).strip())
    else:
        source_text = e(sources)
    risk_status = e(data.get("risk_status") or "暂无额外风险状态。")

    body_sections = [
        section("指数与市场风格", market),
        '<div class="grid2">' + section("行业主力净流入", inflow, "green", True) + section("行业主力净流出", outflow, "red", True) + "</div>",
        section("概念与主线排序", themes, "green"),
        section("热门话题与催化", hot_topics, "red"),
        section("前排 / 中军 / 后排", lineup, "yellow"),
        '<div class="grid2">' + section("分歧监测", divergence, "red", True) + holdings + "</div>" if holdings else section("分歧监测", divergence, "red"),
        section("明日锚点与失败信号", tomorrow, "blue"),
        section("龙虎机制状态", f'<p class="paragraph">{risk_status}</p>', "yellow"),
        section("来源", f'<p class="sources">{source_text}</p>', "blue"),
    ]

    return f"""<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{escape(title)}</title>
<style>
* {{ box-sizing: border-box; }}
body {{
  margin: 0;
  background:
    radial-gradient(circle at 18% 0%, rgba(42, 86, 129, .23), transparent 28rem),
    linear-gradient(180deg, #07101d 0%, #0a1220 55%, #070d17 100%);
  color: #dce8f7;
  font-family: "Microsoft YaHei", "PingFang SC", "Noto Sans CJK SC", Arial, sans-serif;
  letter-spacing: 0;
}}
.page {{
  width: 1080px;
  margin: 0 auto;
  padding: 18px 18px 34px;
}}
.hero {{
  border: 1px solid rgba(98, 135, 188, .32);
  background: linear-gradient(135deg, rgba(20, 35, 58, .98), rgba(8, 15, 28, .98));
  border-radius: 8px;
  padding: 18px 20px 14px;
  box-shadow: 0 18px 48px rgba(0, 0, 0, .28);
}}
h1 {{
  margin: 0 0 9px;
  font-size: 27px;
  line-height: 1.18;
  font-weight: 800;
  color: #f3f7ff;
}}
.subtitle {{
  color: #7f93ad;
  font-size: 11px;
  margin-bottom: 12px;
}}
.verdict {{
  border: 1px solid rgba(252, 205, 94, .65);
  background: rgba(251, 199, 80, .08);
  color: #f8e5aa;
  padding: 10px 13px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 700;
  line-height: 1.55;
}}
.card {{
  margin-top: 14px;
  border: 1px solid rgba(83, 118, 166, .24);
  border-radius: 8px;
  overflow: hidden;
  background: rgba(10, 20, 35, .92);
  box-shadow: inset 0 1px 0 rgba(255,255,255,.03), 0 12px 36px rgba(0,0,0,.2);
}}
.card h2 {{
  margin: 0;
  padding: 10px 13px;
  font-size: 14px;
  line-height: 1.2;
  color: #e8f1ff;
  background: linear-gradient(90deg, rgba(31, 58, 91, .95), rgba(18, 32, 52, .86));
  border-left: 4px solid #3fa7ff;
}}
.card.green h2 {{ border-left-color: #38d8a1; }}
.card.red h2 {{ border-left-color: #ff5c7c; }}
.card.yellow h2 {{ border-left-color: #f6c95b; }}
.grid2 {{
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}}
table {{
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-size: 12px;
}}
th {{
  color: #a9bbd2;
  font-weight: 700;
  text-align: left;
  background: rgba(30, 54, 86, .72);
  padding: 9px 10px;
  white-space: nowrap;
}}
td {{
  border-top: 1px solid rgba(83, 118, 166, .16);
  padding: 9px 10px;
  color: #d7e2f0;
  line-height: 1.42;
  vertical-align: middle;
  word-break: break-word;
}}
tbody tr:nth-child(even) td {{ background: rgba(255,255,255,.015); }}
.num.pos {{ color: #ff5d7e; font-weight: 800; }}
.num.neg {{ color: #27d89b; font-weight: 800; }}
.paragraph, .sources {{
  margin: 0;
  padding: 13px 14px;
  font-size: 13px;
  line-height: 1.65;
  color: #d7e2f0;
}}
.sources {{ color: #879ab2; font-size: 11px; }}
</style>
</head>
<body>
<main class="page">
  <header class="hero">
    <h1>{escape(title)}</h1>
    <div class="subtitle">{escape(subtitle)}</div>
    <div class="verdict">{escape(verdict)}</div>
  </header>
  {''.join(body_sections)}
</main>
</body>
</html>
"""


def write_outputs(data: dict[str, Any], output_dir: Path, basename: str | None) -> tuple[Path, Path]:
    output_dir.mkdir(parents=True, exist_ok=True)
    name = basename or f"{data.get('date') or date.today().isoformat()}-a-share-after-hours"
    html_path = output_dir / f"{name}.html"
    png_path = output_dir / f"{name}.png"
    html_path.write_text(render_html(data), encoding="utf-8")
    return html_path, png_path


def screenshot_with_playwright(html_path: Path, png_path: Path, width: int) -> bool:
    try:
        from playwright.sync_api import sync_playwright  # type: ignore
    except Exception:
        return False
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=True)
            page = browser.new_page(viewport={"width": width, "height": 1600}, device_scale_factor=1)
            page.goto(html_path.resolve().as_uri(), wait_until="load")
            page.screenshot(path=str(png_path), full_page=True)
            browser.close()
        return True
    except Exception as exc:
        sys.stderr.write(f"Playwright screenshot failed, trying browser fallback: {exc}\n")
        return False


def find_browser() -> str | None:
    env = os.environ.get("CHROME_PATH")
    if env and Path(env).exists():
        return env
    for exe in ("msedge", "chrome", "chromium", "chrome.exe", "msedge.exe"):
        found = shutil.which(exe)
        if found:
            return found
    candidates = [
        Path(os.environ.get("ProgramFiles", "")) / "Google/Chrome/Application/chrome.exe",
        Path(os.environ.get("ProgramFiles(x86)", "")) / "Google/Chrome/Application/chrome.exe",
        Path(os.environ.get("ProgramFiles", "")) / "Microsoft/Edge/Application/msedge.exe",
        Path(os.environ.get("ProgramFiles(x86)", "")) / "Microsoft/Edge/Application/msedge.exe",
    ]
    for candidate in candidates:
        if candidate.exists():
            return str(candidate)
    return None


def screenshot_with_browser(html_path: Path, png_path: Path, width: int, height: int) -> bool:
    browser = find_browser()
    if not browser:
        return False
    with tempfile.TemporaryDirectory(prefix="after-hours-summary-") as tmp:
        cmd = [
            browser,
            "--headless=new",
            "--no-sandbox",
            "--disable-gpu",
            "--disable-crash-reporter",
            "--disable-breakpad",
            "--disable-dev-shm-usage",
            "--hide-scrollbars",
            f"--user-data-dir={tmp}",
            f"--window-size={width},{height}",
            f"--screenshot={png_path}",
            html_path.resolve().as_uri(),
        ]
        completed = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        if completed.returncode != 0:
            sys.stderr.write(completed.stderr)
            return False
    return png_path.exists()


def main() -> int:
    parser = argparse.ArgumentParser(description="Render after-hours A-share summary HTML and PNG.")
    parser.add_argument("--input", help="Path to normalized summary JSON. Omit with --sample.")
    parser.add_argument("--output-dir", default="outputs/after-hours-summary", help="Output directory.")
    parser.add_argument("--basename", help="Output base filename without extension.")
    parser.add_argument("--sample", action="store_true", help="Render built-in sample data.")
    parser.add_argument("--html-only", action="store_true", help="Only write HTML, skip PNG screenshot.")
    parser.add_argument("--width", type=int, default=1080, help="Screenshot viewport width.")
    parser.add_argument("--fallback-height", type=int, default=2600, help="Browser fallback screenshot height.")
    args = parser.parse_args()

    if args.sample:
        data = sample_data()
    elif args.input:
        data = json.loads(Path(args.input).read_text(encoding="utf-8"))
    else:
        parser.error("Provide --input <summary.json> or --sample.")

    output_dir = Path(args.output_dir)
    html_path, png_path = write_outputs(data, output_dir, args.basename)
    if args.html_only:
        print(json.dumps({"html": str(html_path), "png": None}, ensure_ascii=False, indent=2))
        return 0

    ok = screenshot_with_playwright(html_path, png_path, args.width)
    if not ok:
        ok = screenshot_with_browser(html_path, png_path, args.width, args.fallback_height)
    if not ok:
        print(json.dumps({"html": str(html_path), "png": None, "error": "PNG screenshot failed; install Playwright or Chrome/Edge."}, ensure_ascii=False, indent=2))
        return 2

    print(json.dumps({"html": str(html_path), "png": str(png_path)}, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
