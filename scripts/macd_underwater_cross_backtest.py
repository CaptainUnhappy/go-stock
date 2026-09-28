#!/usr/bin/env python3
"""MACD 水下金叉历史验证。

两套数据保持独立：
1. GoStock 严格缓存日线：精确到交易日，信号日收盘确认、次日开盘成交；
2. 本地月度截面 CSV：只验证月末指标与后续收益，不冒充可成交回测。
"""

from __future__ import annotations

import argparse
import ast
import json
import math
from collections import Counter, defaultdict
from datetime import datetime
from pathlib import Path
from typing import Any, Iterable

import numpy as np
import pandas as pd


HORIZONS = (5, 10, 20)
MAINBOARD_PREFIXES = ("000", "001", "002", "003", "600", "601", "603", "605")
BUY_COMMISSION = 0.00025
SELL_COMMISSION = 0.00025
SLIPPAGE_ONE_WAY = 0.0005
STAMP_DUTY_SELL = 0.0005
TRANSFER_ONE_WAY = 0.00002
MIN_COMMISSION = 5.0


def ema_sma_seed(values: np.ndarray, period: int) -> np.ndarray:
    """与 GoStock K 线信号口径一致：SMA 初始化，随后标准 EMA。"""
    result = np.full(len(values), np.nan, dtype=float)
    finite = np.flatnonzero(np.isfinite(values))
    if len(finite) < period:
        return result
    start = finite[0]
    window = values[start : start + period]
    if len(window) < period or not np.isfinite(window).all():
        return result
    seed_index = start + period - 1
    result[seed_index] = float(window.mean())
    alpha = 2.0 / (period + 1.0)
    for index in range(seed_index + 1, len(values)):
        if np.isfinite(values[index]):
            result[index] = alpha * values[index] + (1.0 - alpha) * result[index - 1]
    return result


def macd_sma_seed(close: np.ndarray) -> tuple[np.ndarray, np.ndarray, np.ndarray]:
    dif = ema_sma_seed(close, 12) - ema_sma_seed(close, 26)
    dea = ema_sma_seed(dif, 9)
    histogram = 2.0 * (dif - dea)
    return dif, dea, histogram


def normalize_code(value: Any) -> str:
    digits = "".join(character for character in str(value) if character.isdigit())
    return digits[-6:].zfill(6) if digits else ""


def is_mainboard(code: str) -> bool:
    return len(code) == 6 and code.startswith(MAINBOARD_PREFIXES)


def load_stock_basic(path: Path) -> dict[str, dict[str, Any]]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    data = payload.get("data", {})
    fields = data.get("fields", [])
    records: dict[str, dict[str, Any]] = {}
    for item in data.get("items", []):
        row = dict(zip(fields, item))
        code = normalize_code(row.get("symbol"))
        if code:
            records[code] = row
    return records


def listed_for_120_business_days(list_date: Any, signal_date: pd.Timestamp) -> bool:
    parsed = pd.to_datetime(str(list_date), format="%Y%m%d", errors="coerce")
    if pd.isna(parsed) or parsed >= signal_date:
        return False
    return int(np.busday_count(parsed.date(), signal_date.date())) >= 120


def choose_shares(entry_open: float, low_cash: float = 3000.0, high_cash: float = 5000.0) -> int:
    if not np.isfinite(entry_open) or entry_open <= 0:
        return 0
    min_lots = max(1, math.ceil(low_cash / (entry_open * 100.0)))
    max_lots = math.floor(high_cash / (entry_open * 100.0))
    if max_lots < min_lots:
        return 0
    target_lots = round(4000.0 / (entry_open * 100.0))
    return int(min(max(target_lots, min_lots), max_lots) * 100)


def trade_returns(entry_open: float, exit_close: float, shares: int) -> tuple[float, float, float]:
    gross_return = exit_close / entry_open - 1.0
    buy_price = entry_open * (1.0 + SLIPPAGE_ONE_WAY)
    sell_price = exit_close * (1.0 - SLIPPAGE_ONE_WAY)
    buy_notional = buy_price * shares
    sell_notional = sell_price * shares
    buy_fees = max(MIN_COMMISSION, buy_notional * BUY_COMMISSION) + buy_notional * TRANSFER_ONE_WAY
    sell_fees = (
        max(MIN_COMMISSION, sell_notional * SELL_COMMISSION)
        + sell_notional * (STAMP_DUTY_SELL + TRANSFER_ONE_WAY)
    )
    cash_out = buy_notional + buy_fees
    net_return = (sell_notional - sell_fees - cash_out) / cash_out
    return gross_return, net_return, buy_notional


def eligible_current_stock(code: str, basic: dict[str, Any]) -> bool:
    name = str(basic.get("name", "")).upper()
    return (
        is_mainboard(code)
        and basic.get("list_status") == "L"
        and str(basic.get("market", "")) == "主板"
        and "ST" not in name
    )


def process_gostock(
    input_dir: Path,
    stock_basic_path: Path,
    max_files: int | None = None,
) -> tuple[pd.DataFrame, pd.DataFrame, dict[str, Any]]:
    basics = load_stock_basic(stock_basic_path)
    paths = sorted(input_dir.glob("*.json"))
    if max_files:
        paths = paths[:max_files]

    trades: list[dict[str, Any]] = []
    benchmark: dict[tuple[str, int], list[float]] = defaultdict(lambda: [0.0, 0.0])
    sources: Counter[str] = Counter()
    rejection_counts: Counter[str] = Counter()
    accepted_files = 0
    min_date: pd.Timestamp | None = None
    max_date: pd.Timestamp | None = None

    for path in paths:
        payload = json.loads(path.read_text(encoding="utf-8"))
        code = normalize_code(payload.get("code", path.stem))
        basic = basics.get(code, {})
        if not eligible_current_stock(code, basic):
            rejection_counts["非当前非ST主板"] += 1
            continue
        daily = pd.DataFrame(payload.get("daily", []))
        if daily.empty or not {"date", "open", "close", "volume"}.issubset(daily.columns):
            rejection_counts["日线为空或字段缺失"] += 1
            continue
        daily["date"] = pd.to_datetime(daily["date"], errors="coerce")
        for column in ("open", "close", "volume"):
            daily[column] = pd.to_numeric(daily[column], errors="coerce")
        daily = daily.dropna(subset=["date", "open", "close", "volume"]).drop_duplicates("date").sort_values("date")
        if len(daily) < 40:
            rejection_counts["日线不足40条"] += 1
            continue
        accepted_files += 1
        min_date = daily["date"].min() if min_date is None else min(min_date, daily["date"].min())
        max_date = daily["date"].max() if max_date is None else max(max_date, daily["date"].max())
        sources[str(payload.get("sources", {}).get("kline", "GoStock缓存未记录源标签"))] += 1

        close = daily["close"].to_numpy(dtype=float)
        open_ = daily["open"].to_numpy(dtype=float)
        volume = daily["volume"].to_numpy(dtype=float)
        dates = daily["date"].to_numpy()
        dif, dea, histogram = macd_sma_seed(close)
        ma5 = pd.Series(close).rolling(5, min_periods=5).mean().to_numpy()
        prev5_volume = pd.Series(volume).shift(1).rolling(5, min_periods=5).mean().to_numpy()

        # 同期等权主板作为机会成本基准；只按日期聚合，不使用未来数据生成信号。
        for entry_index in range(len(daily)):
            entry_date = pd.Timestamp(dates[entry_index]).strftime("%Y-%m-%d")
            for horizon in HORIZONS:
                exit_index = entry_index + horizon - 1
                if exit_index < len(daily) and open_[entry_index] > 0:
                    benchmark[(entry_date, horizon)][0] += close[exit_index] / open_[entry_index] - 1.0
                    benchmark[(entry_date, horizon)][1] += 1.0

        for signal_index in range(1, len(daily) - 1):
            if not (
                np.isfinite(dif[signal_index - 1])
                and np.isfinite(dea[signal_index - 1])
                and np.isfinite(dif[signal_index])
                and np.isfinite(dea[signal_index])
                and dif[signal_index - 1] <= dea[signal_index - 1]
                and dif[signal_index] > dea[signal_index]
                and dif[signal_index] < 0.0
                and dea[signal_index] < 0.0
            ):
                continue
            signal_date = pd.Timestamp(dates[signal_index])
            if not listed_for_120_business_days(basic.get("list_date"), signal_date):
                rejection_counts["上市不足120工作日"] += 1
                continue
            entry_index = signal_index + 1
            entry_open = open_[entry_index]
            shares = choose_shares(entry_open)
            volume_ratio = volume[signal_index] / prev5_volume[signal_index] if prev5_volume[signal_index] > 0 else np.nan
            entry_gap = entry_open / close[signal_index] - 1.0
            passes_ma5 = bool(np.isfinite(ma5[signal_index]) and close[signal_index] > ma5[signal_index])
            passes_volume = bool(np.isfinite(volume_ratio) and volume_ratio >= 0.80)
            passes_gap = bool(np.isfinite(entry_gap) and entry_gap <= 0.03 and entry_gap < 0.095)
            passes_budget = shares > 0
            passes_strict = passes_ma5 and passes_volume and passes_gap and passes_budget
            common = {
                "code": code,
                "name": basic.get("name", ""),
                "signal_date": signal_date.strftime("%Y-%m-%d"),
                "entry_date": pd.Timestamp(dates[entry_index]).strftime("%Y-%m-%d"),
                "signal_close": close[signal_index],
                "entry_open": entry_open,
                "dif": dif[signal_index],
                "dea": dea[signal_index],
                "histogram": histogram[signal_index],
                "ma5": ma5[signal_index],
                "volume_ratio_prev5": volume_ratio,
                "entry_gap": entry_gap,
                "shares": shares,
                "passes_ma5": passes_ma5,
                "passes_volume": passes_volume,
                "passes_gap": passes_gap,
                "passes_budget": passes_budget,
                "passes_strict": passes_strict,
            }
            for horizon in HORIZONS:
                exit_index = entry_index + horizon - 1
                if exit_index >= len(daily):
                    continue
                for variant in ("core", "strict"):
                    if variant == "strict" and not passes_strict:
                        continue
                    variant_shares = shares if shares else max(100, round(4000.0 / (entry_open * 100.0)) * 100)
                    gross_return, net_return, buy_notional = trade_returns(
                        entry_open, close[exit_index], variant_shares
                    )
                    trades.append(
                        {
                            **common,
                            "variant": variant,
                            "horizon": horizon,
                            "exit_date": pd.Timestamp(dates[exit_index]).strftime("%Y-%m-%d"),
                            "exit_close": close[exit_index],
                            "buy_notional": buy_notional,
                            "gross_return": gross_return,
                            "net_return": net_return,
                        }
                    )

    trade_frame = pd.DataFrame(trades)
    if not trade_frame.empty:
        benchmark_lookup = {
            key: total / count
            for key, (total, count) in benchmark.items()
            if count
        }
        trade_frame["benchmark_gross_return"] = [
            benchmark_lookup.get((entry_date, int(horizon)), np.nan)
            for entry_date, horizon in zip(trade_frame["entry_date"], trade_frame["horizon"])
        ]
        trade_frame["gross_excess_vs_equal_weight"] = (
            trade_frame["gross_return"] - trade_frame["benchmark_gross_return"]
        )
    metrics = summarize_trades(trade_frame, group_columns=["variant", "horizon"])
    metadata = {
        "input_files": len(paths),
        "eligible_files": accepted_files,
        "trade_rows": len(trade_frame),
        "unique_core_signals": int(
            trade_frame.loc[trade_frame.get("variant", pd.Series(dtype=str)).eq("core"), ["code", "signal_date"]]
            .drop_duplicates()
            .shape[0]
        ) if not trade_frame.empty else 0,
        "unique_strict_signals": int(
            trade_frame.loc[trade_frame.get("variant", pd.Series(dtype=str)).eq("strict"), ["code", "signal_date"]]
            .drop_duplicates()
            .shape[0]
        ) if not trade_frame.empty else 0,
        "date_start": min_date.strftime("%Y-%m-%d") if min_date is not None else None,
        "date_end": max_date.strftime("%Y-%m-%d") if max_date is not None else None,
        "source_labels": dict(sources),
        "file_rejections": dict(rejection_counts),
        "adjustment": "qfq（由原始 GoStock 严格缓存生成代码确认）",
        "limitations": [
            "缓存只有开盘、收盘、成交量，没有最高/最低价；不能验证信号日低点/5%止损和一字板。",
            "股票名称与上市状态使用当前 stock_basic，存在幸存者偏差和历史 ST 状态偏差。",
            "结果是单笔信号事件研究；20,000 元组合的并发持仓与信号排序尚未定义，因此不伪造组合净值。",
        ],
    }
    return trade_frame, metrics, metadata


def parse_forward_returns(raw: Any) -> list[float]:
    if raw is None or (isinstance(raw, float) and np.isnan(raw)):
        return []
    try:
        values = ast.literal_eval(str(raw))
    except (ValueError, SyntaxError):
        return []
    if not isinstance(values, list):
        return []
    return [float(value) for value in values if value is not None and np.isfinite(float(value))]


def estimated_roundtrip_cost_rate(notional: float = 4000.0) -> float:
    buy_fees = max(MIN_COMMISSION, notional * BUY_COMMISSION) + notional * TRANSFER_ONE_WAY
    sell_fees = max(MIN_COMMISSION, notional * SELL_COMMISSION) + notional * (STAMP_DUTY_SELL + TRANSFER_ONE_WAY)
    return 2.0 * SLIPPAGE_ONE_WAY + (buy_fees + sell_fees) / notional


def process_snapshot_csv(path: Path) -> tuple[pd.DataFrame, pd.DataFrame, dict[str, Any]]:
    columns = [
        "交易日期", "股票代码", "股票名称", "是否交易", "收盘价", "上市至今交易天数",
        "DIF", "DEA", "bias_5", "新版申万一级行业名称", "下周期每天涨跌幅",
    ]
    frame = pd.read_csv(path, encoding="gb18030", usecols=columns, low_memory=False)
    raw_rows = len(frame)
    frame["code"] = frame["股票代码"].map(normalize_code)
    frame["date"] = pd.to_datetime(frame["交易日期"], errors="coerce")
    for column in ("是否交易", "收盘价", "上市至今交易天数", "DIF", "DEA", "bias_5"):
        frame[column] = pd.to_numeric(frame[column], errors="coerce")
    frame = frame.loc[
        frame["code"].map(is_mainboard)
        & frame["是否交易"].eq(1)
        & frame["上市至今交易天数"].ge(120)
        & ~frame["股票名称"].astype(str).str.upper().str.contains("ST", regex=False)
    ].copy()
    frame = frame.dropna(subset=["date", "DIF", "DEA"]).drop_duplicates(["code", "date"]).sort_values(["code", "date"])
    frame["prev_dif"] = frame.groupby("code", sort=False)["DIF"].shift(1)
    frame["prev_dea"] = frame.groupby("code", sort=False)["DEA"].shift(1)
    core_mask = (
        frame["prev_dif"].le(frame["prev_dea"])
        & frame["DIF"].gt(frame["DEA"])
        & frame["DIF"].lt(0)
        & frame["DEA"].lt(0)
    )
    signals = frame.loc[core_mask].copy()
    signals["passes_ma5_proxy"] = signals["bias_5"].gt(0)
    cost_rate = estimated_roundtrip_cost_rate()
    rows: list[dict[str, Any]] = []
    for signal in signals.itertuples(index=False):
        forward = parse_forward_returns(getattr(signal, "下周期每天涨跌幅"))
        common = {
            "code": signal.code,
            "name": getattr(signal, "股票名称"),
            "signal_date": signal.date.strftime("%Y-%m-%d"),
            "industry": getattr(signal, "新版申万一级行业名称"),
            "dif": signal.DIF,
            "dea": signal.DEA,
            "bias_5": signal.bias_5,
            "passes_ma5_proxy": signal.passes_ma5_proxy,
        }
        for horizon in HORIZONS:
            if len(forward) < horizon:
                continue
            gross_return = float(np.prod(1.0 + np.asarray(forward[:horizon], dtype=float)) - 1.0)
            for variant in ("core", "ma5_proxy"):
                if variant == "ma5_proxy" and not signal.passes_ma5_proxy:
                    continue
                rows.append(
                    {
                        **common,
                        "variant": variant,
                        "horizon": horizon,
                        "gross_return": gross_return,
                        "net_return_estimate": gross_return - cost_rate,
                    }
                )
    result = pd.DataFrame(rows)
    metrics = summarize_trades(
        result.rename(columns={"net_return_estimate": "net_return"}),
        group_columns=["variant", "horizon"],
    )
    metadata = {
        "raw_rows": int(raw_rows),
        "eligible_rows": int(len(frame)),
        "unique_core_signals": int(signals[["code", "date"]].drop_duplicates().shape[0]),
        "unique_ma5_proxy_signals": int(signals.loc[signals["passes_ma5_proxy"], ["code", "date"]].drop_duplicates().shape[0]),
        "date_start": frame["date"].min().strftime("%Y-%m-%d") if not frame.empty else None,
        "date_end": frame["date"].max().strftime("%Y-%m-%d") if not frame.empty else None,
        "estimated_roundtrip_cost_rate": cost_rate,
        "source_label": f"local_csv:{path.as_posix()}",
        "limitations": [
            "CSV 是月度截面，不是连续日线；金叉只能表示相邻月末快照之间发生了状态切换。",
            "下周期每天涨跌幅仅作为信号产生后的结果标签，未参与筛选；其口径不是次日开盘可成交价。",
            "CSV 缺少可用的逐日成交量序列，无法验证量能过滤、开盘跳空和止损。",
            "历史 ST/停牌标记覆盖有限；本层只作为长样本方向性稳健性检查。",
        ],
    }
    return result, metrics, metadata


def summarize_trades(frame: pd.DataFrame, group_columns: list[str]) -> pd.DataFrame:
    columns = group_columns + [
        "trades", "stocks", "start", "end", "avg_gross", "avg_net", "median_net",
        "win_rate_net", "p10_net", "p90_net", "avg_benchmark_gross", "avg_gross_excess",
    ]
    if frame.empty:
        return pd.DataFrame(columns=columns)
    rows: list[dict[str, Any]] = []
    for keys, group in frame.groupby(group_columns, dropna=False):
        if not isinstance(keys, tuple):
            keys = (keys,)
        row = dict(zip(group_columns, keys))
        dates = pd.to_datetime(group["signal_date"], errors="coerce")
        row.update(
            {
                "trades": int(len(group)),
                "stocks": int(group["code"].nunique()),
                "start": dates.min().strftime("%Y-%m-%d"),
                "end": dates.max().strftime("%Y-%m-%d"),
                "avg_gross": float(group["gross_return"].mean()),
                "avg_net": float(group["net_return"].mean()),
                "median_net": float(group["net_return"].median()),
                "win_rate_net": float(group["net_return"].gt(0).mean()),
                "p10_net": float(group["net_return"].quantile(0.10)),
                "p90_net": float(group["net_return"].quantile(0.90)),
                "avg_benchmark_gross": float(group["benchmark_gross_return"].mean())
                if "benchmark_gross_return" in group else np.nan,
                "avg_gross_excess": float(group["gross_excess_vs_equal_weight"].mean())
                if "gross_excess_vs_equal_weight" in group else np.nan,
            }
        )
        rows.append(row)
    return pd.DataFrame(rows).sort_values(group_columns).reset_index(drop=True)


def summarize_by_year(frame: pd.DataFrame) -> pd.DataFrame:
    if frame.empty:
        return pd.DataFrame()
    yearly = frame.copy()
    yearly["year"] = pd.to_datetime(yearly["signal_date"], errors="coerce").dt.year
    return summarize_trades(yearly, group_columns=["variant", "horizon", "year"])


def summarize_signal_date_cohorts(frame: pd.DataFrame) -> tuple[pd.DataFrame, pd.DataFrame]:
    if frame.empty:
        return pd.DataFrame(), pd.DataFrame()
    aggregation: dict[str, tuple[str, str]] = {
        "signals": ("code", "size"),
        "cohort_net": ("net_return", "mean"),
    }
    if "gross_excess_vs_equal_weight" in frame:
        aggregation["cohort_gross_excess"] = ("gross_excess_vs_equal_weight", "mean")
    daily = (
        frame.groupby(["variant", "horizon", "signal_date"], as_index=False)
        .agg(**aggregation)
        .sort_values(["variant", "horizon", "signal_date"])
    )
    rows: list[dict[str, Any]] = []
    for (variant, horizon), group in daily.groupby(["variant", "horizon"]):
        rows.append(
            {
                "variant": variant,
                "horizon": horizon,
                "signal_days": len(group),
                "avg_signals_per_day": float(group["signals"].mean()),
                "avg_cohort_net": float(group["cohort_net"].mean()),
                "median_cohort_net": float(group["cohort_net"].median()),
                "positive_cohort_days": float(group["cohort_net"].gt(0).mean()),
                "avg_cohort_gross_excess": float(group["cohort_gross_excess"].mean())
                if "cohort_gross_excess" in group else np.nan,
            }
        )
    return daily, pd.DataFrame(rows).sort_values(["variant", "horizon"]).reset_index(drop=True)


def build_filter_funnel(frame: pd.DataFrame) -> pd.DataFrame:
    if frame.empty:
        return pd.DataFrame(columns=["stage", "signals", "retention"])
    signals = frame.loc[frame["variant"].eq("core")].drop_duplicates(["code", "signal_date"])
    masks: list[tuple[str, pd.Series]] = [
        ("水下金叉", pd.Series(True, index=signals.index)),
        ("+ 收盘价>MA5", signals["passes_ma5"]),
        ("+ 量能>=前5日均量80%", signals["passes_ma5"] & signals["passes_volume"]),
        (
            "+ 次日开盘涨幅<=3%",
            signals["passes_ma5"] & signals["passes_volume"] & signals["passes_gap"],
        ),
        ("+ 3000-5000元整手可买", signals["passes_strict"]),
    ]
    total = max(len(signals), 1)
    return pd.DataFrame(
        [{"stage": stage, "signals": int(mask.sum()), "retention": float(mask.sum() / total)} for stage, mask in masks]
    )


def markdown_metrics(frame: pd.DataFrame) -> str:
    if frame.empty:
        return "无有效记录。"
    display = frame.copy()
    percent_columns = [
        "avg_gross", "avg_net", "median_net", "win_rate_net", "p10_net", "p90_net",
        "avg_benchmark_gross", "avg_gross_excess",
    ]
    for column in percent_columns:
        if column in display:
            display[column] = display[column].map(lambda value: "" if pd.isna(value) else f"{value:.2%}")
    return display.to_markdown(index=False)


def markdown_cohorts(frame: pd.DataFrame) -> str:
    if frame.empty:
        return "无有效记录。"
    display = frame.copy()
    for column in ("avg_cohort_net", "median_cohort_net", "positive_cohort_days", "avg_cohort_gross_excess"):
        display[column] = display[column].map(lambda value: "" if pd.isna(value) else f"{value:.2%}")
    display["avg_signals_per_day"] = display["avg_signals_per_day"].map(lambda value: f"{value:.1f}")
    return display.to_markdown(index=False)


def write_report(
    output_dir: Path,
    gostock_metrics: pd.DataFrame,
    gostock_meta: dict[str, Any],
    gostock_cohort_metrics: pd.DataFrame,
    filter_funnel: pd.DataFrame,
    csv_metrics: pd.DataFrame,
    csv_meta: dict[str, Any],
) -> None:
    strict_excess = gostock_metrics.loc[gostock_metrics["variant"].eq("strict"), "avg_gross_excess"]
    verdict = (
        "近期 GoStock 日线没有证明这套信号具有独立超额收益：完整过滤在所有持有期均落后同期等权主板。"
        if not strict_excess.empty and strict_excess.le(0).all()
        else "近期 GoStock 日线出现部分正超额，但仍需检查年度稳定性与组合容量。"
    )
    report = f"""# MACD 水下金叉历史验证

生成时间：{datetime.now().astimezone().isoformat(timespec='seconds')}

## 核心判定

{verdict} 长历史 CSV 的正收益只能视为方向性证据，因为它不是连续日线，也不是次日开盘成交口径。

## 结论读取顺序

先看 GoStock 日线的 `strict`：这是最接近可交易口径的结果。`core` 仅含水下金叉，用来检验附加过滤是否真的改善结果。本地 CSV 的 `ma5_proxy` 只做长样本方向性验证，不能当成精确成交回测。

## GoStock 日线事件研究

- 数据范围：{gostock_meta.get('date_start')} 至 {gostock_meta.get('date_end')}
- 输入文件：{gostock_meta.get('input_files')}；当前非 ST 主板有效文件：{gostock_meta.get('eligible_files')}
- 核心信号：{gostock_meta.get('unique_core_signals')}；完整过滤信号：{gostock_meta.get('unique_strict_signals')}
- 成交口径：信号日收盘确认，下一交易日开盘买入，第 5/10/20 个持有日收盘卖出
- 资金口径：每笔 3,000–5,000 元、100 股整数倍、目标约 4,000 元；佣金最低 5 元

{markdown_metrics(gostock_metrics)}

### 按信号日聚类后的复核

同一天出现的几十只金叉高度相关，因此把每个信号日先做等权平均，再统计信号日，避免把横截面拥挤误当成独立样本。

{markdown_cohorts(gostock_cohort_metrics)}

### 过滤漏斗

{filter_funnel.to_markdown(index=False)}

## 本地 CSV 月度稳健性验证

- 数据范围：{csv_meta.get('date_start')} 至 {csv_meta.get('date_end')}
- 原始行：{csv_meta.get('raw_rows')}；过滤后有效行：{csv_meta.get('eligible_rows')}
- 核心月末信号：{csv_meta.get('unique_core_signals')}；MA5 代理过滤：{csv_meta.get('unique_ma5_proxy_signals')}
- 估算单笔往返成本：{csv_meta.get('estimated_roundtrip_cost_rate', float('nan')):.2%}

{markdown_metrics(csv_metrics)}

## 不能跨越的边界

### GoStock

{chr(10).join('- ' + item for item in gostock_meta.get('limitations', []))}

### 本地 CSV

{chr(10).join('- ' + item for item in csv_meta.get('limitations', []))}

## 数据源

- GoStock K 线源标签：`{json.dumps(gostock_meta.get('source_labels', {}), ensure_ascii=False)}`
- CSV：`{csv_meta.get('source_label')}`
"""
    (output_dir / "report.md").write_text(report, encoding="utf-8")


def discover_gostock_dir(project_root: Path) -> Path:
    candidates = list((project_root / "quant").glob("REVS*/data/strict/inputs"))
    if not candidates:
        raise FileNotFoundError("未找到 quant/REVS*/data/strict/inputs")
    return candidates[0]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="GoStock + 本地 CSV 的 MACD 水下金叉历史验证")
    parser.add_argument("--project-root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--gostock-dir", type=Path)
    parser.add_argument("--stock-basic", type=Path)
    parser.add_argument("--snapshot-csv", type=Path)
    parser.add_argument("--output-dir", type=Path)
    parser.add_argument("--max-files", type=int, help="仅用于快速冒烟验证")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = args.project_root.resolve()
    gostock_dir = args.gostock_dir or discover_gostock_dir(root)
    stock_basic = args.stock_basic or root / "backend" / "data" / "stock_basic.json"
    snapshot_csv = args.snapshot_csv or root / "quant" / "data" / "供选股数据.csv"
    output_dir = args.output_dir or root / "outputs" / "macd_underwater_cross_backtest"
    output_dir.mkdir(parents=True, exist_ok=True)

    print(f"[1/2] GoStock 日线：{gostock_dir}", flush=True)
    gostock_trades, gostock_metrics, gostock_meta = process_gostock(
        gostock_dir, stock_basic, max_files=args.max_files
    )
    gostock_year_metrics = summarize_by_year(gostock_trades)
    gostock_cohorts, gostock_cohort_metrics = summarize_signal_date_cohorts(gostock_trades)
    filter_funnel = build_filter_funnel(gostock_trades)
    print(f"[2/2] 本地月度 CSV：{snapshot_csv}", flush=True)
    csv_trades, csv_metrics, csv_meta = process_snapshot_csv(snapshot_csv)

    gostock_trades.to_csv(output_dir / "gostock_trades.csv", index=False, encoding="utf-8-sig")
    gostock_metrics.to_csv(output_dir / "gostock_metrics.csv", index=False, encoding="utf-8-sig")
    gostock_year_metrics.to_csv(output_dir / "gostock_metrics_by_year.csv", index=False, encoding="utf-8-sig")
    gostock_cohorts.to_csv(output_dir / "gostock_signal_date_cohorts.csv", index=False, encoding="utf-8-sig")
    gostock_cohort_metrics.to_csv(output_dir / "gostock_cohort_metrics.csv", index=False, encoding="utf-8-sig")
    filter_funnel.to_csv(output_dir / "gostock_filter_funnel.csv", index=False, encoding="utf-8-sig")
    csv_trades.to_csv(output_dir / "csv_snapshot_signals.csv", index=False, encoding="utf-8-sig")
    csv_metrics.to_csv(output_dir / "csv_snapshot_metrics.csv", index=False, encoding="utf-8-sig")
    summary = {
        "generated_at": datetime.now().astimezone().isoformat(timespec="seconds"),
        "method": {
            "signal": "MACD(12,26,9) SMA seed; DIF crosses above DEA; DIF<0 and DEA<0",
            "strict_filters": "close>MA5; signal volume>=80% of prior-5 mean; next-open gap<=3%; 3000-5000 yuan round lot",
            "horizons": list(HORIZONS),
        },
        "gostock": gostock_meta,
        "snapshot_csv": csv_meta,
    }
    (output_dir / "summary.json").write_text(
        json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    write_report(
        output_dir, gostock_metrics, gostock_meta, gostock_cohort_metrics, filter_funnel, csv_metrics, csv_meta
    )
    print("\nGoStock：")
    print(gostock_metrics.to_string(index=False))
    print("\n本地 CSV：")
    print(csv_metrics.to_string(index=False))
    print(f"\n结果目录：{output_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
