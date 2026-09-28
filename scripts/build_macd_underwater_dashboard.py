#!/usr/bin/env python3
"""Build an offline HTML dashboard for the MACD underwater-cross backtest."""

from __future__ import annotations

import argparse
import html as html_module
import json
import re
from datetime import datetime
from pathlib import Path
from typing import Any

import pandas as pd


REQUIRED_FILES = (
    "summary.json",
    "gostock_metrics.csv",
    "gostock_metrics_by_year.csv",
    "gostock_cohort_metrics.csv",
    "gostock_signal_date_cohorts.csv",
    "gostock_filter_funnel.csv",
    "csv_snapshot_metrics.csv",
)


def records(path: Path) -> list[dict[str, Any]]:
    frame = pd.read_csv(path)
    return json.loads(frame.to_json(orient="records", force_ascii=False))


def extract_echarts(reference_html: Path) -> str:
    source = reference_html.read_text(encoding="utf-8")
    scripts = re.findall(r"<script(?:\s[^>]*)?>(.*?)</script>", source, flags=re.DOTALL | re.IGNORECASE)
    for script in scripts:
        if "echarts" in script and "zrender" in script and "Apache License" in script:
            return script.strip()
    raise ValueError(f"未能从参考页提取内嵌 ECharts: {reference_html}")


def build_report_data(output_dir: Path) -> dict[str, Any]:
    summary = json.loads((output_dir / "summary.json").read_text(encoding="utf-8"))
    metrics = records(output_dir / "gostock_metrics.csv")
    annual = records(output_dir / "gostock_metrics_by_year.csv")
    cohort_metrics = records(output_dir / "gostock_cohort_metrics.csv")
    cohorts = [
        row
        for row in records(output_dir / "gostock_signal_date_cohorts.csv")
        if row.get("variant") == "strict"
    ]
    funnel = records(output_dir / "gostock_filter_funnel.csv")
    snapshot = records(output_dir / "csv_snapshot_metrics.csv")
    strict_excess = [
        float(row["avg_gross_excess"])
        for row in metrics
        if row.get("variant") == "strict" and row.get("avg_gross_excess") is not None
    ]
    verdict = (
        "完整过滤在5日、10日和20日三个持有期均未跑赢同期等权主板。"
        if strict_excess and all(value < 0 for value in strict_excess)
        else "部分持有期出现正超额，需要结合年度稳定性继续判断。"
    )
    return {
        "schema_version": 1,
        "title": "MACD水下金叉历史验证",
        "generated_at": datetime.now().astimezone().isoformat(timespec="seconds"),
        "backtest_generated_at": summary["generated_at"],
        "verdict": verdict,
        "method": summary["method"],
        "gostock": summary["gostock"],
        "snapshot_meta": summary["snapshot_csv"],
        "metrics": metrics,
        "annual": annual,
        "cohort_metrics": cohort_metrics,
        "cohorts": cohorts,
        "funnel": funnel,
        "snapshot_metrics": snapshot,
        "downloads": [
            ["回测摘要", "summary.json"],
            ["GoStock总体指标", "gostock_metrics.csv"],
            ["年度指标", "gostock_metrics_by_year.csv"],
            ["信号日聚类", "gostock_signal_date_cohorts.csv"],
            ["过滤漏斗", "gostock_filter_funnel.csv"],
            ["GoStock交易明细", "gostock_trades.csv"],
            ["CSV月度指标", "csv_snapshot_metrics.csv"],
            ["CSV月度信号", "csv_snapshot_signals.csv"],
            ["文字报告", "report.md"],
        ],
    }


HTML_TEMPLATE = r'''<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="description" content="GoStock日线与本地CSV驱动的MACD水下金叉历史验证">
  <title>MACD水下金叉历史验证</title>
  <link rel="icon" href="data:,">
  <style>
    :root {
      color-scheme: light;
      --bg: #f4f6f9;
      --surface: #ffffff;
      --surface-subtle: #f8fafc;
      --ink: #172033;
      --muted: #657086;
      --line: #dfe4ec;
      --accent: #2775d8;
      --accent-soft: #e8f1fd;
      --positive: #087a55;
      --negative: #bd3f3f;
      --warning: #8a5b12;
      --warning-soft: #fff7e8;
      --table-head: #58677b;
      --table-stripe: #eef1f5;
      --shadow: 0 0 0 1px rgba(23,32,51,.05), 0 2px 8px rgba(23,32,51,.06);
    }
    [data-theme="dark"] {
      color-scheme: dark;
      --bg: #0e1420;
      --surface: #151d2b;
      --surface-subtle: #1b2535;
      --ink: #edf2f8;
      --muted: #9ba8bb;
      --line: #2b3748;
      --accent: #64a7ff;
      --accent-soft: #1a3556;
      --positive: #4cc99c;
      --negative: #f07d7d;
      --warning: #e8b967;
      --warning-soft: #2a2418;
      --table-head: #263449;
      --table-stripe: #1c2737;
      --shadow: 0 0 0 1px rgba(255,255,255,.08);
    }
    * { box-sizing: border-box; }
    html { scroll-behavior: smooth; -webkit-font-smoothing: antialiased; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--ink);
      font-family: "Segoe UI", "Microsoft YaHei UI", "Microsoft YaHei", system-ui, sans-serif;
      line-height: 1.55;
    }
    button, a { font: inherit; }
    a { color: inherit; text-decoration: none; }
    button:focus-visible, a:focus-visible, summary:focus-visible { outline: 3px solid color-mix(in srgb, var(--accent) 55%, transparent); outline-offset: 3px; }
    h1, h2, h3, p { margin-top: 0; }
    .shell { width: min(1480px, calc(100% - 40px)); margin: 0 auto; }
    .topbar {
      position: sticky; top: 0; z-index: 10; min-height: 64px;
      background: color-mix(in srgb, var(--bg) 88%, transparent);
      backdrop-filter: blur(14px); border-bottom: 1px solid var(--line);
    }
    .topbar-inner { min-height: 64px; display: flex; align-items: center; justify-content: space-between; gap: 20px; }
    .brand { display: flex; align-items: baseline; gap: 12px; min-width: 0; }
    .brand strong { font-size: 16px; letter-spacing: -.01em; }
    .brand span { color: var(--muted); font-size: 13px; }
    nav { display: flex; align-items: center; gap: 4px; white-space: nowrap; }
    nav a, .theme-toggle {
      min-height: 40px; display: inline-flex; align-items: center; padding: 0 12px;
      color: var(--muted); border-radius: 9px; border: 0; background: transparent;
      font-size: 13px; cursor: pointer; transition: color 150ms, background-color 150ms, transform 150ms;
    }
    nav a:hover, .theme-toggle:hover { color: var(--ink); background: var(--surface); }
    nav a:active, .theme-toggle:active, .tab:active { transform: scale(.97); }
    .hero { padding: 64px 0 34px; display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(300px, .55fr); gap: 52px; align-items: end; }
    .eyebrow { margin-bottom: 12px; color: var(--accent); font-size: 12px; font-weight: 700; letter-spacing: .12em; }
    .hero h1 { margin-bottom: 0; max-width: 18ch; font-size: clamp(34px, 4.4vw, 62px); line-height: 1.05; letter-spacing: -.045em; }
    .hero p { max-width: 67ch; margin: 18px 0 0; color: var(--muted); font-size: 16px; }
    .asof { padding: 22px; border-radius: 16px; background: var(--surface); box-shadow: var(--shadow); }
    .asof-label, .asof-note { color: var(--muted); font-size: 12px; }
    .asof-value { margin-top: 3px; font-size: 27px; font-weight: 700; letter-spacing: -.02em; font-variant-numeric: tabular-nums; }
    .asof-note { margin-top: 8px; }
    .notice { margin: 0 0 18px; padding: 13px 16px; border-left: 3px solid var(--warning); background: color-mix(in srgb, var(--warning) 9%, var(--surface)); color: var(--muted); border-radius: 0 10px 10px 0; font-size: 13px; }
    .verdict { display: grid; grid-template-columns: auto 1fr; gap: 13px; align-items: baseline; margin: 0 0 28px; padding: 17px 19px; border-radius: 12px; background: var(--surface); box-shadow: var(--shadow); }
    .verdict strong { color: var(--negative); font-size: 13px; }
    .verdict span { font-size: 14px; }
    section { scroll-margin-top: 82px; margin-bottom: 66px; }
    .toolbar { display: flex; justify-content: space-between; align-items: center; gap: 18px; margin-bottom: 14px; }
    .toolbar p { margin: 0; color: var(--muted); font-size: 13px; }
    .tabs { display: inline-flex; gap: 2px; padding: 3px; border-radius: 10px; background: var(--surface-subtle); box-shadow: inset 0 0 0 1px var(--line); }
    .tab { min-height: 34px; padding: 0 13px; border: 0; border-radius: 7px; color: var(--muted); background: transparent; cursor: pointer; transition: color 150ms, background-color 150ms, transform 150ms; }
    .tab[aria-selected="true"] { color: var(--ink); background: var(--surface); box-shadow: var(--shadow); }
    .metric-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
    .metric { min-height: 132px; padding: 20px; background: var(--surface); border-radius: 14px; box-shadow: var(--shadow); }
    .metric-label, .metric-note { color: var(--muted); font-size: 12px; }
    .metric-value { margin: 8px 0 5px; font-size: clamp(25px, 2.5vw, 37px); font-weight: 720; letter-spacing: -.035em; font-variant-numeric: tabular-nums; }
    .positive { color: var(--positive) !important; }
    .negative { color: var(--negative) !important; }
    .section-head { margin-bottom: 22px; }
    .section-head h2 { margin-bottom: 0; font-size: 27px; letter-spacing: -.025em; }
    .section-head p { max-width: 78ch; margin: 7px 0 0; color: var(--muted); font-size: 14px; }
    .panel { padding: 24px; border-radius: 16px; background: var(--surface); box-shadow: var(--shadow); }
    .split { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
    .split > *, .panel { min-width: 0; }
    .chart-wrap { width: 100%; min-height: 350px; }
    .chart-large { min-height: 430px; }
    .chart-hint { margin: 0 0 12px; color: var(--muted); font-size: 12px; }
    .spacer { height: 16px; }
    .table-wrap { overflow-x: auto; border-radius: 16px; background: var(--surface); box-shadow: var(--shadow); }
    table { width: 100%; border-collapse: collapse; font-size: 13px; font-variant-numeric: tabular-nums; }
    th { padding: 13px 15px; color: var(--muted); background: var(--surface-subtle); text-align: left; font-size: 12px; font-weight: 650; border-bottom: 1px solid var(--line); }
    td { padding: 13px 15px; border-bottom: 1px solid var(--line); }
    tbody tr:last-child td { border-bottom: 0; }
    tbody tr:hover { background: color-mix(in srgb, var(--accent) 4%, transparent); }
    .number { text-align: right; }
    .funnel-row { display: grid; grid-template-columns: 190px 1fr 100px; align-items: center; gap: 14px; margin: 16px 0; }
    .funnel-label { font-size: 13px; }
    .funnel-track { height: 10px; border-radius: 5px; background: var(--surface-subtle); overflow: hidden; box-shadow: inset 0 0 0 1px var(--line); }
    .funnel-fill { height: 100%; border-radius: inherit; background: var(--accent); transform-origin: left center; }
    .funnel-value { text-align: right; color: var(--muted); font-size: 12px; font-variant-numeric: tabular-nums; }
    .boundary { background: var(--warning-soft); border: 1px solid color-mix(in srgb, var(--warning) 28%, var(--line)); }
    .boundary-badge { display: inline-block; margin-bottom: 11px; color: var(--warning); font-size: 12px; font-weight: 700; }
    .methods { display: grid; grid-template-columns: repeat(2, minmax(0,1fr)); gap: 12px; margin: 0; }
    .method { padding: 18px; border-radius: 12px; background: var(--surface); box-shadow: var(--shadow); }
    .method dt { margin-bottom: 5px; font-weight: 700; }
    .method dd { margin: 0; color: var(--muted); font-size: 13px; }
    .risk-grid { display: grid; grid-template-columns: .75fr 1.25fr; gap: 16px; }
    .coverage-list { display: grid; gap: 15px; }
    .coverage-item { display: flex; justify-content: space-between; gap: 20px; align-items: baseline; }
    .coverage-item span { color: var(--muted); font-size: 13px; }
    .coverage-item strong { font-variant-numeric: tabular-nums; }
    .risk-list { margin: 0; padding-left: 20px; color: var(--muted); font-size: 13px; }
    .risk-list li + li { margin-top: 8px; }
    details { margin-top: 16px; border-radius: 12px; background: var(--surface); box-shadow: var(--shadow); }
    summary { padding: 16px 18px; cursor: pointer; font-weight: 650; }
    .downloads { display: grid; grid-template-columns: repeat(3, minmax(0,1fr)); gap: 10px; padding: 0 18px 18px; }
    .download { display: flex; justify-content: space-between; gap: 12px; padding: 12px 14px; border-radius: 9px; background: var(--surface-subtle); color: var(--muted); font-size: 13px; }
    .download:hover { color: var(--accent); }
    footer { margin-top: 30px; padding: 24px 0 34px; border-top: 1px solid var(--line); color: var(--muted); font-size: 12px; }
    .chart-fallback { min-height: inherit; display: grid; place-items: center; color: var(--muted); text-align: center; }
    @media (max-width: 1000px) {
      .metric-grid { grid-template-columns: repeat(2, minmax(0,1fr)); }
      .split, .risk-grid { grid-template-columns: 1fr; }
      nav a { display: none; }
    }
    @media (max-width: 720px) {
      .shell { width: min(100% - 24px, 1480px); }
      .hero { grid-template-columns: 1fr; gap: 22px; padding-top: 42px; }
      .hero h1 { font-size: 38px; }
      .toolbar { align-items: flex-start; flex-direction: column; }
      .metric-grid, .methods, .downloads { grid-template-columns: 1fr; }
      .funnel-row { grid-template-columns: 1fr 74px; gap: 7px 12px; }
      .funnel-track { grid-column: 1 / -1; grid-row: 2; }
      .verdict { grid-template-columns: 1fr; }
      .chart-wrap, .chart-large { min-height: 330px; }
    }
    @media (prefers-reduced-motion: reduce) { html { scroll-behavior: auto; } * { transition-duration: 0.01ms !important; } }
    @media print { .topbar, .tabs { display: none !important; } body { background: #fff; } .panel, .metric, .table-wrap, .method { box-shadow: 0 0 0 1px #ddd; break-inside: avoid; } }
  </style>
</head>
<body>
  <header class="topbar">
    <div class="shell topbar-inner">
      <div class="brand"><strong>MACD水下金叉研究</strong><span>GoStock日线 + 本地CSV</span></div>
      <nav aria-label="报告导航">
        <a href="#overview">概览</a><a href="#performance">表现</a><a href="#stability">稳定性</a><a href="#long-sample">长样本</a><a href="#methodology">口径</a>
        <button class="theme-toggle" type="button" id="theme-toggle" aria-label="切换明暗主题">主题</button>
      </nav>
    </div>
  </header>
  <main class="shell">
    <div class="hero">
      <div>
        <div class="eyebrow">MACD UNDERWATER CROSS</div>
        <h1>水下金叉历史验证</h1>
        <p>信号日收盘确认，下一交易日开盘买入。GoStock日线负责可交易验证，本地月度CSV只做方向性长样本检查。</p>
      </div>
      <div class="asof">
        <div class="asof-label">GoStock数据截止</div><div class="asof-value" id="data-end"></div>
        <div class="asof-note" id="generated-at"></div>
      </div>
    </div>
    <div class="notice">研究边界：结果是单笔信号事件研究，不是2万元账户的组合净值。最高价、最低价、历史ST和退市状态不完整，因此止损、一字板和幸存者偏差仍未解决。</div>
    <div class="verdict" role="note"><strong>核心判定</strong><span id="verdict-copy"></span></div>

    <section id="overview">
      <div class="toolbar">
        <p>切换持有期查看完整过滤策略。金额按每笔3000-5000元、100股整数倍和最低5元佣金计算。</p>
        <div class="tabs" role="tablist" aria-label="持有期"><button class="tab horizon-tab" data-horizon="5" role="tab" aria-selected="false">5日</button><button class="tab horizon-tab" data-horizon="10" role="tab" aria-selected="true">10日</button><button class="tab horizon-tab" data-horizon="20" role="tab" aria-selected="false">20日</button></div>
      </div>
      <div class="metric-grid">
        <article class="metric"><div class="metric-label">平均净收益</div><div class="metric-value" data-metric="avg_net"></div><div class="metric-note">含佣金、滑点、印花税和过户费</div></article>
        <article class="metric"><div class="metric-label">相对主板超额</div><div class="metric-value" data-metric="avg_gross_excess"></div><div class="metric-note">策略毛收益减同期等权主板毛收益</div></article>
        <article class="metric"><div class="metric-label">中位数净收益</div><div class="metric-value" data-metric="median_net"></div><div class="metric-note">降低极端涨幅对均值的影响</div></article>
        <article class="metric"><div class="metric-label">净收益胜率</div><div class="metric-value" data-metric="win_rate_net"></div><div class="metric-note">单笔净收益大于0</div></article>
        <article class="metric"><div class="metric-label">P10风险</div><div class="metric-value negative" data-metric="p10_net"></div><div class="metric-note">最差10%分位</div></article>
        <article class="metric"><div class="metric-label">P90收益</div><div class="metric-value positive" data-metric="p90_net"></div><div class="metric-note">最好10%分位起点</div></article>
        <article class="metric"><div class="metric-label">可评估交易</div><div class="metric-value" data-metric="trades"></div><div class="metric-note" data-metric-note="stocks"></div></article>
        <article class="metric"><div class="metric-label">信号日均候选</div><div class="metric-value" data-metric="avg_signals_per_day"></div><div class="metric-note" data-metric-note="signal_days"></div></article>
      </div>
    </section>

    <section id="performance">
      <div class="section-head"><h2>信号表现</h2><p>对比原始水下金叉与完整过滤。正收益不等于有效超额，必须同时观察同期主板基准。</p></div>
      <div class="split">
        <div class="panel"><p class="chart-hint">平均净收益、中位数净收益与相对等权主板毛超额。</p><div class="chart-wrap" id="return-chart" role="img" aria-label="不同持有期收益与超额"></div></div>
        <div class="panel"><p class="chart-hint">P10、中位数与P90揭示收益分布，不能只看平均值。</p><div class="chart-wrap" id="distribution-chart" role="img" aria-label="严格策略收益分位数"></div></div>
      </div>
      <div class="spacer"></div>
      <div class="table-wrap"><table><thead><tr><th>口径</th><th class="number">持有期</th><th class="number">交易数</th><th class="number">平均净收益</th><th class="number">中位数</th><th class="number">胜率</th><th class="number">相对主板超额</th></tr></thead><tbody id="metrics-table"></tbody></table></div>
    </section>

    <section id="filters">
      <div class="section-head"><h2>过滤漏斗</h2><p>过滤保留了72.43%的可评估水下金叉，但没有改善总体超额收益。</p></div>
      <div class="panel" id="funnel-list"></div>
    </section>

    <section id="stability">
      <div class="section-head"><h2>年度稳定性</h2><p>2022年仅10笔严格信号，不能与完整年度等权解读。2026年只统计到7月。</p></div>
      <div class="panel"><p class="chart-hint">柱形为严格策略平均净收益，折线为相对等权主板毛超额。标签显示交易数。</p><div class="chart-wrap chart-large" id="annual-chart" role="img" aria-label="严格策略年度稳定性"></div></div>
      <div class="spacer"></div>
      <div class="table-wrap"><table><thead><tr><th>年度</th><th class="number">交易数</th><th class="number">平均净收益</th><th class="number">中位数</th><th class="number">胜率</th><th class="number">相对主板超额</th></tr></thead><tbody id="annual-table"></tbody></table></div>
      <div class="spacer"></div>
      <div class="panel"><p class="chart-hint">同一天的信号先做横截面等权。点大小表示当日信号数，本图不是组合净值。</p><div class="chart-wrap chart-large" id="cohort-chart" role="img" aria-label="信号日横截面等权收益"></div></div>
    </section>

    <section id="long-sample">
      <div class="section-head"><h2>长样本方向性检查</h2><p>本地CSV覆盖2006-2026年，但只有月末截面，不是连续日线，不能与GoStock可交易结果合并。</p></div>
      <div class="panel boundary"><span class="boundary-badge">不可作为成交回测</span><p>月末DIF与DEA只说明相邻快照之间发生状态切换。未来收益列仅作结果标签，不参与筛选；没有次日开盘、逐日成交量和止损路径。</p><div class="chart-wrap" id="snapshot-chart" role="img" aria-label="月度CSV方向性收益"></div></div>
    </section>

    <section id="methodology">
      <div class="section-head"><h2>策略与数据口径</h2><p>所有口径均保留在页面与下载文件中，便于复算和识别偏差。</p></div>
      <dl class="methods">
        <div class="method"><dt>水下金叉</dt><dd>MACD(12,26,9)，SMA初始化。前一日DIF不高于DEA，当日DIF上穿DEA，且DIF和DEA均低于0。</dd></div>
        <div class="method"><dt>完整过滤</dt><dd>收盘价高于MA5；成交量不低于前5日均量80%；次日开盘涨幅不高于3%；3000-5000元可买100股整数倍。</dd></div>
        <div class="method"><dt>成交时点</dt><dd>信号日收盘确认，下一交易日开盘买入，第5、10或20个持有日收盘卖出。</dd></div>
        <div class="method"><dt>交易成本</dt><dd>买卖佣金各0.025%，最低5元；单边滑点0.05%；卖出印花税0.05%；买卖过户费各0.002%。</dd></div>
        <div class="method"><dt>同期基准</dt><dd>同一入场日期和持有期内，可用主板股票开盘至收盘收益的横截面等权平均。</dd></div>
        <div class="method"><dt>信号日聚类</dt><dd>同日信号先等权平均，再按日期统计，避免把市场共振下的几十只股票误当成独立样本。</dd></div>
      </dl>
    </section>

    <section id="quality">
      <div class="section-head"><h2>数据质量与限制</h2><p>页面不会用演示曲线、随机值或硬编码净值填补缺失数据。</p></div>
      <div class="risk-grid">
        <div class="panel"><div class="coverage-list" id="coverage-list"></div></div>
        <div class="panel"><ul class="risk-list" id="risk-list"></ul></div>
      </div>
      <details><summary>下载数据与查看来源</summary><div class="downloads" id="downloads"></div></details>
    </section>
  </main>
  <footer><div class="shell">MACD水下金叉量化研究报告。仅用于历史验证，不构成投资建议。</div></footer>
  <script>__ECHARTS_SOURCE__</script>
  <script id="report-data" type="application/json">__REPORT_DATA__</script>
  <script>
  (() => {
    const root = document.documentElement;
    const report = JSON.parse(document.getElementById("report-data").textContent);
    const themeButton = document.getElementById("theme-toggle");
    const chartIds = ["return-chart", "distribution-chart", "annual-chart", "cohort-chart", "snapshot-chart"];
    const charts = new Map();
    let horizon = 10;
    let saved = null;
    try { saved = localStorage.getItem("macd-underwater-theme"); } catch (_) {}
    if (saved === "dark" || saved === "light") root.dataset.theme = saved;
    const currentTheme = () => root.dataset.theme || (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
    themeButton.setAttribute("aria-pressed", String(currentTheme() === "dark"));
    themeButton.addEventListener("click", () => {
      const next = currentTheme() === "dark" ? "light" : "dark";
      root.dataset.theme = next;
      themeButton.setAttribute("aria-pressed", String(next === "dark"));
      try { localStorage.setItem("macd-underwater-theme", next); } catch (_) {}
      requestAnimationFrame(renderCharts);
    });

    const token = name => getComputedStyle(root).getPropertyValue(name).trim();
    const tones = () => ({ ink: token("--ink"), muted: token("--muted"), line: token("--line"), surface: token("--surface"), accent: token("--accent"), positive: token("--positive"), negative: token("--negative"), warning: token("--warning") });
    const pct = (value, digits = 2) => value == null || !Number.isFinite(Number(value)) ? "N/A" : `${Number(value) >= 0 ? "+" : ""}${(Number(value) * 100).toFixed(digits)}%`;
    const integer = value => new Intl.NumberFormat("zh-CN").format(Number(value || 0));
    const toneClass = value => Number(value) >= 0 ? "positive" : "negative";
    const strictMetric = value => report.metrics.find(row => row.variant === "strict" && Number(row.horizon) === Number(value));
    const cohortMetric = value => report.cohort_metrics.find(row => row.variant === "strict" && Number(row.horizon) === Number(value));
    const commonGrid = t => ({ left: 62, right: 30, top: 48, bottom: 55, containLabel: false, borderColor: t.line });
    const axis = t => ({ axisLine: { lineStyle: { color: t.line } }, axisTick: { show: false }, axisLabel: { color: t.muted }, splitLine: { lineStyle: { color: t.line, opacity: .6 } } });
    const tooltip = t => ({ trigger: "axis", backgroundColor: t.surface, borderColor: t.line, textStyle: { color: t.ink }, extraCssText: "box-shadow:0 8px 24px rgba(0,0,0,.12);border-radius:10px" });

    function setTone(element, value) { element.classList.remove("positive", "negative"); element.classList.add(toneClass(value)); }
    function renderMetrics() {
      const metric = strictMetric(horizon);
      const cohort = cohortMetric(horizon);
      const values = {
        avg_net: pct(metric.avg_net), avg_gross_excess: pct(metric.avg_gross_excess), median_net: pct(metric.median_net),
        win_rate_net: pct(metric.win_rate_net, 1), p10_net: pct(metric.p10_net), p90_net: pct(metric.p90_net),
        trades: integer(metric.trades), avg_signals_per_day: Number(cohort.avg_signals_per_day).toFixed(1)
      };
      document.querySelectorAll("[data-metric]").forEach(el => {
        const key = el.dataset.metric; el.textContent = values[key];
        if (["avg_net", "avg_gross_excess", "median_net"].includes(key)) setTone(el, metric[key]);
      });
      document.querySelector('[data-metric-note="stocks"]').textContent = `${integer(metric.stocks)}只股票`;
      document.querySelector('[data-metric-note="signal_days"]').textContent = `${integer(cohort.signal_days)}个信号日`;
    }

    function renderStatic() {
      document.getElementById("data-end").textContent = report.gostock.date_end;
      document.getElementById("generated-at").textContent = `报告生成 ${report.generated_at.replace("T", " ")}`;
      document.getElementById("verdict-copy").textContent = `${report.verdict} 这套信号可用于缩小候选范围，但不能单独构成买入理由。`;
      const table = document.getElementById("metrics-table");
      table.innerHTML = report.metrics.map(row => `<tr><td><strong>${row.variant === "strict" ? "完整过滤" : "原始金叉"}</strong></td><td class="number">${row.horizon}日</td><td class="number">${integer(row.trades)}</td><td class="number ${toneClass(row.avg_net)}">${pct(row.avg_net)}</td><td class="number ${toneClass(row.median_net)}">${pct(row.median_net)}</td><td class="number">${pct(row.win_rate_net, 1)}</td><td class="number ${toneClass(row.avg_gross_excess)}">${pct(row.avg_gross_excess)}</td></tr>`).join("");
      const funnel = document.getElementById("funnel-list");
      funnel.innerHTML = report.funnel.map(row => `<div class="funnel-row"><div class="funnel-label">${row.stage}</div><div class="funnel-track" aria-hidden="true"><div class="funnel-fill" style="transform:scaleX(${row.retention})"></div></div><div class="funnel-value">${integer(row.signals)} / ${(row.retention * 100).toFixed(1)}%</div></div>`).join("");
      const coverage = [
        ["输入文件", integer(report.gostock.input_files)], ["有效主板股票", integer(report.gostock.eligible_files)],
        ["核心信号", integer(report.gostock.unique_core_signals)], ["完整过滤信号", integer(report.gostock.unique_strict_signals)],
        ["GoStock行情起点", report.gostock.date_start], ["GoStock行情终点", report.gostock.date_end],
        ["CSV原始行", integer(report.snapshot_meta.raw_rows)], ["CSV日期范围", `${report.snapshot_meta.date_start} 至 ${report.snapshot_meta.date_end}`]
      ];
      document.getElementById("coverage-list").innerHTML = coverage.map(item => `<div class="coverage-item"><span>${item[0]}</span><strong>${item[1]}</strong></div>`).join("");
      const risks = [...report.gostock.limitations, ...report.snapshot_meta.limitations];
      document.getElementById("risk-list").innerHTML = risks.map(item => `<li>${item}</li>`).join("");
      document.getElementById("downloads").innerHTML = report.downloads.map(item => `<a class="download" href="${item[1]}" download><span>${item[0]}</span><strong>下载</strong></a>`).join("");
    }

    function renderAnnualTable() {
      const rows = report.annual.filter(row => row.variant === "strict" && Number(row.horizon) === horizon);
      document.getElementById("annual-table").innerHTML = rows.map(row => `<tr><td><strong>${row.year}</strong>${Number(row.trades) < 100 ? '<div class="negative">小样本</div>' : ""}</td><td class="number">${integer(row.trades)}</td><td class="number ${toneClass(row.avg_net)}">${pct(row.avg_net)}</td><td class="number ${toneClass(row.median_net)}">${pct(row.median_net)}</td><td class="number">${pct(row.win_rate_net, 1)}</td><td class="number ${toneClass(row.avg_gross_excess)}">${pct(row.avg_gross_excess)}</td></tr>`).join("");
    }

    function initChart(id) {
      const element = document.getElementById(id);
      if (!window.echarts) { element.innerHTML = '<div class="chart-fallback">图表组件未加载。请查看下方表格或CSV。</div>'; return null; }
      if (!charts.has(id)) charts.set(id, echarts.init(element, null, { renderer: "canvas" }));
      return charts.get(id);
    }

    function returnOption(t) {
      const horizons = [5, 10, 20];
      const strict = horizons.map(h => strictMetric(h));
      return { color: [t.accent, t.positive, t.negative], tooltip: tooltip(t), legend: { top: 4, textStyle: { color: t.muted } }, grid: commonGrid(t), xAxis: { type: "category", data: horizons.map(h => `${h}日`), ...axis(t) }, yAxis: { type: "value", axisLabel: { formatter: "{value}%", color: t.muted }, ...axis(t) }, series: [
        { name: "平均净收益", type: "bar", data: strict.map(r => +(r.avg_net * 100).toFixed(4)), barMaxWidth: 34 },
        { name: "中位数净收益", type: "bar", data: strict.map(r => +(r.median_net * 100).toFixed(4)), barMaxWidth: 34 },
        { name: "相对主板毛超额", type: "line", data: strict.map(r => +(r.avg_gross_excess * 100).toFixed(4)), symbolSize: 8, lineStyle: { width: 3 } }
      ]};
    }

    function distributionOption(t) {
      const horizons = [5, 10, 20]; const strict = horizons.map(h => strictMetric(h));
      return { color: [t.negative, t.accent, t.positive], tooltip: tooltip(t), legend: { top: 4, textStyle: { color: t.muted } }, grid: commonGrid(t), xAxis: { type: "category", data: horizons.map(h => `${h}日`), ...axis(t) }, yAxis: { type: "value", axisLabel: { formatter: "{value}%", color: t.muted }, ...axis(t) }, series: [
        { name: "P10", type: "bar", data: strict.map(r => +(r.p10_net * 100).toFixed(3)), barMaxWidth: 28 },
        { name: "中位数", type: "bar", data: strict.map(r => +(r.median_net * 100).toFixed(3)), barMaxWidth: 28 },
        { name: "P90", type: "bar", data: strict.map(r => +(r.p90_net * 100).toFixed(3)), barMaxWidth: 28 }
      ]};
    }

    function annualOption(t) {
      const rows = report.annual.filter(row => row.variant === "strict" && Number(row.horizon) === horizon);
      return { color: [t.accent, t.negative], tooltip: { ...tooltip(t), formatter: params => { const index = params[0].dataIndex; const row = rows[index]; return `<strong>${row.year}</strong><br>平均净收益：${pct(row.avg_net)}<br>相对主板超额：${pct(row.avg_gross_excess)}<br>交易数：${integer(row.trades)}`; } }, legend: { top: 4, textStyle: { color: t.muted } }, grid: commonGrid(t), xAxis: { type: "category", data: rows.map(r => String(r.year)), ...axis(t) }, yAxis: { type: "value", axisLabel: { formatter: "{value}%", color: t.muted }, ...axis(t) }, series: [
        { name: "平均净收益", type: "bar", barMaxWidth: 52, data: rows.map(r => ({ value: +(r.avg_net * 100).toFixed(3), label: { show: true, position: r.avg_net >= 0 ? "top" : "bottom", formatter: integer(r.trades), color: t.muted, fontSize: 10 } })) },
        { name: "相对主板毛超额", type: "line", data: rows.map(r => +(r.avg_gross_excess * 100).toFixed(3)), symbolSize: 8, lineStyle: { width: 3 } }
      ]};
    }

    function cohortOption(t) {
      const rows = report.cohorts.filter(row => Number(row.horizon) === horizon);
      const maxSignals = Math.max(...rows.map(r => Number(r.signals)), 1);
      return { color: [t.accent], tooltip: { ...tooltip(t), trigger: "item", formatter: p => `${p.value[0]}<br>横截面净收益：${p.value[1].toFixed(2)}%<br>信号数：${integer(p.value[2])}` }, grid: { ...commonGrid(t), bottom: 75 }, xAxis: { type: "time", ...axis(t) }, yAxis: { type: "value", axisLabel: { formatter: "{value}%", color: t.muted }, ...axis(t) }, dataZoom: [{ type: "inside" }, { type: "slider", bottom: 18, height: 20, borderColor: t.line, textStyle: { color: t.muted } }], series: [{ type: "scatter", data: rows.map(r => [r.signal_date, +(r.cohort_net * 100).toFixed(4), r.signals]), symbolSize: value => 5 + Math.sqrt(value[2] / maxSignals) * 17, itemStyle: { opacity: .58 } }] };
    }

    function snapshotOption(t) {
      const horizons = [5, 10, 20];
      const core = horizons.map(h => report.snapshot_metrics.find(r => r.variant === "core" && Number(r.horizon) === h));
      const proxy = horizons.map(h => report.snapshot_metrics.find(r => r.variant === "ma5_proxy" && Number(r.horizon) === h));
      return { color: [t.muted, t.warning], tooltip: tooltip(t), legend: { top: 4, textStyle: { color: t.muted } }, grid: commonGrid(t), xAxis: { type: "category", data: horizons.map(h => `${h}日`), ...axis(t) }, yAxis: { type: "value", axisLabel: { formatter: "{value}%", color: t.muted }, ...axis(t) }, series: [
        { name: "月末核心状态", type: "bar", data: core.map(r => +(r.avg_net * 100).toFixed(3)), barMaxWidth: 36 },
        { name: "MA5代理过滤", type: "bar", data: proxy.map(r => +(r.avg_net * 100).toFixed(3)), barMaxWidth: 36 }
      ]};
    }

    function renderCharts() {
      const t = tones();
      const options = { "return-chart": returnOption(t), "distribution-chart": distributionOption(t), "annual-chart": annualOption(t), "cohort-chart": cohortOption(t), "snapshot-chart": snapshotOption(t) };
      chartIds.forEach(id => { const chart = initChart(id); if (chart) chart.setOption(options[id], true); });
    }

    document.querySelectorAll(".horizon-tab").forEach(button => button.addEventListener("click", () => {
      horizon = Number(button.dataset.horizon);
      document.querySelectorAll(".horizon-tab").forEach(item => item.setAttribute("aria-selected", String(item === button)));
      renderMetrics(); renderAnnualTable(); renderCharts();
    }));
    renderStatic(); renderMetrics(); renderAnnualTable(); renderCharts();
    if (window.ResizeObserver) {
      const observer = new ResizeObserver(() => charts.forEach(chart => chart.resize()));
      chartIds.forEach(id => observer.observe(document.getElementById(id)));
    }
  })();
  </script>
</body>
</html>
'''


def visible_text(page: str) -> str:
    stripped = re.sub(r"<script\b[^>]*>.*?</script>", "", page, flags=re.DOTALL | re.IGNORECASE)
    stripped = re.sub(r"<style\b[^>]*>.*?</style>", "", stripped, flags=re.DOTALL | re.IGNORECASE)
    stripped = re.sub(r"<[^>]+>", " ", stripped)
    return html_module.unescape(stripped)


def build_dashboard(output_dir: Path, reference_html: Path) -> Path:
    missing = [name for name in REQUIRED_FILES if not (output_dir / name).is_file()]
    if missing:
        raise FileNotFoundError(f"缺少回测产物: {', '.join(missing)}")
    report = build_report_data(output_dir)
    echarts = extract_echarts(reference_html)
    report_json = json.dumps(report, ensure_ascii=False, separators=(",", ":")).replace("</", "<\\/")
    page = HTML_TEMPLATE.replace("__ECHARTS_SOURCE__", echarts).replace("__REPORT_DATA__", report_json)
    visible = visible_text(page)
    if "—" in visible or "–" in visible:
        raise ValueError("可见文案包含禁用的长破折号")
    for forbidden in ("Rank IC", "下月TOP10", "财报可得"):
        if forbidden in visible:
            raise ValueError(f"页面残留不属于MACD报告的文案: {forbidden}")
    for required_id in ("overview", "performance", "stability", "long-sample", "methodology", "return-chart"):
        if f'id="{required_id}"' not in page:
            raise ValueError(f"页面缺少区域: {required_id}")
    path = output_dir / "dashboard.html"
    path.write_text(page, encoding="utf-8")
    return path


def parse_args() -> argparse.Namespace:
    project_root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description="生成MACD水下金叉离线HTML仪表盘")
    parser.add_argument("--output-dir", type=Path, default=project_root / "outputs" / "macd_underwater_cross_backtest")
    parser.add_argument(
        "--reference-html",
        type=Path,
        default=project_root / "quant" / "earnings_growth" / "output" / "walk_forward_2010_2024q3" / "dashboard.html",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    path = build_dashboard(args.output_dir.resolve(), args.reference_html.resolve())
    size_mb = path.stat().st_size / (1024 * 1024)
    print(f"已生成: {path}")
    print(f"大小: {size_mb:.2f} MB")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
