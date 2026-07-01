import argparse
import concurrent.futures as futures
import json
import math
import time
from pathlib import Path

import matplotlib.pyplot as plt
import pandas as pd
import requests


ROOT = Path(__file__).resolve().parents[1]
STOCK_BASIC = ROOT / "build" / "stock_basic.json"
OUTPUT_DIR = ROOT / "outputs" / "kline_signal_summary"


def normalize_code(ts_code: str) -> str:
    code, market = ts_code.split(".")
    if market == "SH":
        return f"1.{code}"
    if market == "SZ":
        return f"0.{code}"
    return ""


def load_universe(max_stocks: int | None) -> list[dict]:
    raw = json.loads(STOCK_BASIC.read_text(encoding="utf-8"))
    fields = raw["data"]["fields"]
    rows = raw["data"]["items"]
    items = [dict(zip(fields, row)) for row in rows]

    prefixes = ("000", "001", "002", "003", "300", "301", "600", "601", "603", "605", "688")
    universe = []
    for item in items:
        ts_code = item.get("ts_code") or ""
        symbol = item.get("symbol") or ""
        name = item.get("name") or ""
        if item.get("list_status") != "L":
            continue
        if not symbol.startswith(prefixes):
            continue
        if "ST" in name.upper() or "退" in name:
            continue
        secid = normalize_code(ts_code)
        if not secid:
            continue
        universe.append(
            {
                "ts_code": ts_code,
                "symbol": symbol,
                "name": name,
                "industry": item.get("industry") or "",
                "list_date": item.get("list_date") or "",
                "secid": secid,
            }
        )

    universe.sort(key=lambda x: x["ts_code"])
    if max_stocks and max_stocks > 0 and max_stocks < len(universe):
        step = len(universe) / max_stocks
        return [universe[min(int(i * step), len(universe) - 1)] for i in range(max_stocks)]
    return universe


def fetch_kline(stock: dict, limit: int, retries: int = 2) -> pd.DataFrame | None:
    url = "https://push2his.eastmoney.com/api/qt/stock/kline/get"
    params = {
        "secid": stock["secid"],
        "klt": "101",
        "fqt": "1",
        "end": "20500101",
        "lmt": str(limit),
        "fields1": "f1,f2,f3,f4,f5,f6",
        "fields2": "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61,f116",
        "_": str(int(time.time() * 1000)),
    }
    headers = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "Referer": "https://quote.eastmoney.com",
    }
    for attempt in range(retries + 1):
        try:
            resp = requests.get(url, params=params, headers=headers, timeout=12)
            resp.raise_for_status()
            payload = resp.json()
            klines = (payload.get("data") or {}).get("klines") or []
            if not klines:
                return None
            rows = []
            for line in klines:
                p = line.split(",")
                if len(p) < 11:
                    continue
                rows.append(
                    {
                        "date": p[0],
                        "open": float(p[1]),
                        "close": float(p[2]),
                        "high": float(p[3]),
                        "low": float(p[4]),
                        "volume": float(p[5]),
                        "amount": float(p[6]),
                        "amplitude": float(p[7]) / 100.0,
                        "pct_chg": float(p[8]) / 100.0,
                        "change": float(p[9]),
                        "turnover": float(p[10]) / 100.0,
                        "ts_code": stock["ts_code"],
                        "symbol": stock["symbol"],
                        "name": stock["name"],
                        "industry": stock["industry"],
                    }
                )
            if not rows:
                return None
            df = pd.DataFrame(rows)
            df["date"] = pd.to_datetime(df["date"])
            return df.sort_values("date")
        except Exception:
            if attempt >= retries:
                return None
            time.sleep(0.35 + attempt * 0.65)
    return None


def build_panel(universe: list[dict], limit: int, workers: int) -> pd.DataFrame:
    frames: list[pd.DataFrame] = []
    total = len(universe)
    with futures.ThreadPoolExecutor(max_workers=workers) as pool:
        jobs = {pool.submit(fetch_kline, stock, limit): stock for stock in universe}
        for i, job in enumerate(futures.as_completed(jobs), 1):
            df = job.result()
            if df is not None and len(df) >= 180:
                frames.append(df)
            if i % 100 == 0 or i == total:
                print(f"fetched {i}/{total}, usable={len(frames)}")
    if not frames:
        raise RuntimeError("No usable kline data fetched.")
    return pd.concat(frames, ignore_index=True).sort_values(["ts_code", "date"])


def load_panel_csv(path: Path) -> pd.DataFrame:
    df = pd.read_csv(path, encoding="utf-8-sig")
    numeric_cols = ["open", "close", "high", "low", "volume", "amount", "pct_chg", "change", "amplitude", "turnover"]
    for col in numeric_cols:
        df[col] = pd.to_numeric(df[col], errors="coerce")
    df["date"] = pd.to_datetime(df["date"])
    df = df.dropna(subset=["date", "open", "close", "high", "low", "volume"])
    if "amount" not in df or df["amount"].fillna(0).le(0).all():
        df["amount"] = df["close"] * df["volume"] * 1_000_000
    else:
        estimated_amount = df["close"] * df["volume"] * 1_000_000
        df["amount"] = df["amount"].where(df["amount"].fillna(0).gt(0), estimated_amount)
    df["pct_chg"] = df["pct_chg"] / 100.0
    df["amplitude"] = df["amplitude"] / 100.0
    df["turnover"] = df["turnover"] / 100.0
    return df.sort_values(["ts_code", "date"])


def ema(s: pd.Series, span: int) -> pd.Series:
    return s.ewm(span=span, adjust=False, min_periods=span).mean()


def rsi(close: pd.Series, period: int = 14) -> pd.Series:
    diff = close.diff()
    gain = diff.clip(lower=0).ewm(alpha=1 / period, adjust=False, min_periods=period).mean()
    loss = (-diff.clip(upper=0)).ewm(alpha=1 / period, adjust=False, min_periods=period).mean()
    rs = gain / loss.replace(0, float("nan"))
    return 100 - (100 / (1 + rs))


def add_signal_counts(g: pd.DataFrame) -> pd.DataFrame:
    g = g.copy()
    close = g["close"]
    high = g["high"]
    low = g["low"]
    vol = g["volume"]
    typical = (high + low + close) / 3

    g["ret"] = close.pct_change()
    g["ma5"] = close.rolling(5).mean()
    g["ma10"] = close.rolling(10).mean()
    g["ma20"] = close.rolling(20).mean()
    g["ma60"] = close.rolling(60).mean()
    g["amount_ma20"] = g["amount"].rolling(20).mean()
    g["volume_ma5_prev"] = vol.shift(1).rolling(5).mean()
    g["volume_ratio_prev5"] = vol / g["volume_ma5_prev"]
    g["ret5"] = close / close.shift(5) - 1
    g["ret20"] = close / close.shift(20) - 1
    g["dist_ma20"] = close / g["ma20"] - 1
    g["dist_ma60"] = close / g["ma60"] - 1
    g["high20"] = high.rolling(20).max()
    g["low20"] = low.rolling(20).min()
    g["drawdown20"] = close / g["high20"] - 1

    counts = {key: pd.Series(0, index=g.index, dtype="int16") for key in ["bullish", "bearish", "neutral", "oscillating", "total"]}

    def mark(bullish, bearish, oscillating=None, neutral=None):
        bullish = bullish.fillna(False)
        bearish = bearish.fillna(False)
        oscillating = pd.Series(False, index=g.index) if oscillating is None else oscillating.fillna(False)
        neutral = pd.Series(False, index=g.index) if neutral is None else neutral.fillna(False)
        valid = bullish | bearish | oscillating | neutral
        counts["bullish"][:] = counts["bullish"] + bullish.astype("int16")
        counts["bearish"][:] = counts["bearish"] + bearish.astype("int16")
        counts["oscillating"][:] = counts["oscillating"] + oscillating.astype("int16")
        counts["neutral"][:] = counts["neutral"] + neutral.astype("int16")
        counts["total"][:] = counts["total"] + valid.astype("int16")

    # Trend indicators aligned with the front-end indicator signal summary.
    mark(g["ma5"].gt(g["ma10"]) & g["ma10"].gt(g["ma20"]) & g["ma20"].gt(g["ma60"]), g["ma5"].lt(g["ma10"]) & g["ma10"].lt(g["ma20"]) & g["ma20"].lt(g["ma60"]), ((g["ma5"].gt(g["ma20"]) & g["ma10"].lt(g["ma60"])) | (g["ma5"].lt(g["ma20"]) & g["ma10"].gt(g["ma60"]))))
    e12, e21 = ema(close, 12), ema(close, 21)
    mark(e12.gt(e21), e12.lt(e21), neutral=e12.eq(e21))
    mid = close.rolling(20).mean()
    std = close.rolling(20).std()
    upper, lower = mid + 2 * std, mid - 2 * std
    mark(close.gt(upper), close.lt(lower), close.gt(mid) & close.le(upper), close.le(mid) & close.ge(lower))
    vwap20 = (typical * vol).rolling(20).sum() / vol.rolling(20).sum()
    mark(close.gt(vwap20), close.lt(vwap20), neutral=close.eq(vwap20))
    e1 = ema(close, 21)
    dema = 2 * e1 - ema(e1, 21)
    tema = 3 * e1 - 3 * ema(e1, 21) + ema(ema(e1, 21), 21)
    mark(close.gt(dema), close.lt(dema), neutral=close.eq(dema))
    mark(close.gt(tema), close.lt(tema), neutral=close.eq(tema))
    tr = pd.concat([(high - low), (high - close.shift()).abs(), (low - close.shift()).abs()], axis=1).max(axis=1)
    atr14 = tr.rolling(14).mean()
    kelt_mid = ema(close, 20)
    mark(close.gt(kelt_mid + 1.5 * atr14), close.lt(kelt_mid - 1.5 * atr14), close.between(kelt_mid - 1.5 * atr14, kelt_mid + 1.5 * atr14))
    don_upper, don_lower = high.rolling(20).max(), low.rolling(20).min()
    mark(close.ge(don_upper), close.le(don_lower), close.gt(don_lower) & close.lt(don_upper))

    # Momentum and oscillator indicators.
    ema12, ema26 = ema(close, 12), ema(close, 26)
    dif = ema12 - ema26
    dea = ema(dif, 9)
    hist = dif - dea
    mark(dif.gt(dea) & hist.gt(0), dif.lt(dea) & hist.lt(0), ((dif.gt(0) & hist.lt(0)) | (dif.lt(0) & hist.gt(0))), neutral=hist.eq(0))
    rsi14 = rsi(close, 14)
    mark(rsi14.gt(50) & rsi14.le(70), rsi14.lt(50) & rsi14.ge(30), rsi14.gt(70) | rsi14.lt(30), neutral=rsi14.eq(50))
    ll9, hh9 = low.rolling(9).min(), high.rolling(9).max()
    rsv = (close - ll9) / (hh9 - ll9).replace(0, float("nan")) * 100
    k = rsv.ewm(alpha=1 / 3, adjust=False, min_periods=3).mean()
    d = k.ewm(alpha=1 / 3, adjust=False, min_periods=3).mean()
    j = 3 * k - 2 * d
    mark((j.gt(k) & k.gt(d) & k.lt(80)) | k.lt(20), (j.lt(k) & k.lt(d) & k.gt(20)) | k.gt(80), neutral=k.between(20, 80) & ~(j.gt(k) & k.gt(d)) & ~(j.lt(k) & k.lt(d)))
    cci = (typical - typical.rolling(20).mean()) / (0.015 * typical.rolling(20).apply(lambda x: (x - x.mean()).abs().mean(), raw=False))
    mark(cci.gt(100), cci.lt(-100), cci.between(-100, 100))
    wr = (high.rolling(14).max() - close) / (high.rolling(14).max() - low.rolling(14).min()).replace(0, float("nan")) * -100
    mark(wr.lt(-80), wr.gt(-20), wr.between(-80, -20))
    rsi_min = rsi14.rolling(14).min()
    rsi_max = rsi14.rolling(14).max()
    stoch_rsi = (rsi14 - rsi_min) / (rsi_max - rsi_min).replace(0, float("nan")) * 100
    stoch_k = stoch_rsi.rolling(3).mean()
    stoch_d = stoch_k.rolling(3).mean()
    mark(stoch_k.lt(20) & stoch_d.lt(20) & stoch_k.gt(stoch_d), stoch_k.gt(80) & stoch_d.gt(80) & stoch_k.lt(stoch_d), neutral=stoch_k.notna() & stoch_d.notna() & ~((stoch_k.lt(20) & stoch_d.lt(20) & stoch_k.gt(stoch_d)) | (stoch_k.gt(80) & stoch_d.gt(80) & stoch_k.lt(stoch_d))))
    plus_dm = (high.diff()).where((high.diff() > -low.diff()) & (high.diff() > 0), 0)
    minus_dm = (-low.diff()).where((-low.diff() > high.diff()) & (-low.diff() > 0), 0)
    tr14 = tr.rolling(14).sum()
    plus_di = 100 * plus_dm.rolling(14).sum() / tr14.replace(0, float("nan"))
    minus_di = 100 * minus_dm.rolling(14).sum() / tr14.replace(0, float("nan"))
    dx = (plus_di - minus_di).abs() / (plus_di + minus_di).replace(0, float("nan")) * 100
    adx = dx.rolling(14).mean()
    mark(adx.gt(25) & plus_di.gt(minus_di), adx.gt(25) & plus_di.lt(minus_di), adx.le(25))
    roc = close / close.shift(12) - 1
    mark(roc.gt(0), roc.lt(0), neutral=roc.eq(0))

    # Volume indicators.
    obv = (vol * close.diff().apply(lambda x: 1 if x > 0 else (-1 if x < 0 else 0))).cumsum()
    mark(obv.gt(obv.shift()), obv.lt(obv.shift()), neutral=obv.eq(obv.shift()))
    raw_money_flow = typical * vol
    pos_flow = raw_money_flow.where(typical.gt(typical.shift()), 0).rolling(14).sum()
    neg_flow = raw_money_flow.where(typical.lt(typical.shift()), 0).rolling(14).sum()
    mfi = 100 - 100 / (1 + pos_flow / neg_flow.replace(0, float("nan")))
    mark(mfi.gt(50) & mfi.le(80), mfi.lt(50) & mfi.ge(20), mfi.gt(80) | mfi.lt(20), neutral=mfi.eq(50))
    mf_multiplier = ((close - low) - (high - close)) / (high - low).replace(0, float("nan"))
    cmf = (mf_multiplier * vol).rolling(20).sum() / vol.rolling(20).sum()
    mark(cmf.gt(0.05), cmf.lt(-0.05), cmf.between(-0.05, 0.05))
    force_index = close.diff() * vol
    mark(force_index.gt(0), force_index.lt(0), neutral=force_index.eq(0))

    # Volatility/risk indicators.
    mark(atr14.gt(atr14.shift()), pd.Series(False, index=g.index), neutral=atr14.le(atr14.shift()))
    chop = 100 * ((tr.rolling(14).sum() / (high.rolling(14).max() - low.rolling(14).min()).replace(0, float("nan"))).apply(lambda x: math.log10(x) if x and x > 0 else float("nan")) / math.log10(14))
    mark(pd.Series(False, index=g.index), pd.Series(False, index=g.index), chop.gt(61.8), neutral=chop.le(61.8))
    max_close14 = close.rolling(14).max()
    ulcer = (((close / max_close14 - 1) * 100).pow(2).rolling(14).mean()).pow(0.5)
    mark(ulcer.lt(5), ulcer.gt(15), neutral=ulcer.between(5, 15))

    for key, series in counts.items():
        g[f"signal_{key}"] = series
    g["bullish_pct"] = g["signal_bullish"] / g["signal_total"].replace(0, float("nan"))
    g["bearish_pct"] = g["signal_bearish"] / g["signal_total"].replace(0, float("nan"))
    g["oscillating_pct"] = g["signal_oscillating"] / g["signal_total"].replace(0, float("nan"))
    g["neutral_pct"] = g["signal_neutral"] / g["signal_total"].replace(0, float("nan"))
    g["signal_spread"] = g["bullish_pct"] - g["bearish_pct"]
    return g


def add_factors(panel: pd.DataFrame) -> pd.DataFrame:
    panel = pd.concat([add_signal_counts(g) for _, g in panel.groupby("ts_code", sort=False)], ignore_index=True)
    panel["liquid"] = panel["amount_ma20"] >= 60_000_000
    panel["not_overheated"] = panel["ret5"].between(-0.12, 0.015) & panel["dist_ma20"].between(-0.10, 0.015)
    panel["volume_confirm"] = panel["volume_ratio_prev5"].between(0.75, 1.8)
    panel["trend_floor"] = panel["close"].gt(panel["ma60"] * 0.85)
    panel["summary_signal"] = (
        panel["liquid"]
        & panel["not_overheated"]
        & panel["volume_confirm"]
        & panel["trend_floor"]
        & panel["signal_total"].ge(18)
        & panel["bullish_pct"].ge(0.28)
        & panel["bearish_pct"].le(0.36)
        & panel["signal_spread"].ge(-0.04)
        & panel["oscillating_pct"].ge(0.22)
        & panel["turnover"].between(0.006, 0.16)
    ).fillna(False)

    ret_repair = 1 - ((panel["ret5"] + 0.055).abs() / 0.12).clip(0, 1)
    low_absorb = 1 - ((panel["dist_ma20"] + 0.04).abs() / 0.10).clip(0, 1)
    volume_balance = 1 - ((panel["volume_ratio_prev5"] - 1.05).abs() / 1.0).clip(0, 1)
    panel["score"] = (
        0.25 * ret_repair.fillna(0)
        + 0.24 * low_absorb.fillna(0)
        + 0.18 * panel["oscillating_pct"].fillna(0)
        + 0.17 * panel["bullish_pct"].fillna(0)
        + 0.10 * volume_balance.fillna(0)
        + 0.06 * panel["signal_spread"].fillna(0)
    )
    return panel


def latest_complete_signal_date(panel: pd.DataFrame, today: pd.Timestamp | None = None) -> pd.Timestamp:
    dates = sorted(panel["date"].drop_duplicates())
    if not dates:
        raise RuntimeError("No dates available.")
    today = today or pd.Timestamp.today().normalize()
    latest = pd.Timestamp(dates[-1]).normalize()
    if latest >= today and len(dates) >= 2:
        return pd.Timestamp(dates[-2])
    return pd.Timestamp(dates[-1])


def net_exit_return(
    entry_open: float,
    exit_price: float,
    commission_bps: float,
    slippage_bps: float,
    stamp_tax_bps: float,
    transfer_fee_bps: float,
) -> float:
    commission = commission_bps / 10000.0
    slippage = slippage_bps / 10000.0
    stamp_tax = stamp_tax_bps / 10000.0
    transfer_fee = transfer_fee_bps / 10000.0
    buy_price = entry_open * (1 + slippage)
    sell_price = exit_price * (1 - slippage)
    buy_cost_multiplier = 1 + commission + transfer_fee
    sell_proceeds_multiplier = 1 - commission - stamp_tax - transfer_fee
    return (sell_price * sell_proceeds_multiplier) / (buy_price * buy_cost_multiplier) - 1


def simulate_trade(
    g: pd.DataFrame,
    signal_date: pd.Timestamp,
    hold_days: int,
    stop_loss: float,
    take_profit: float,
    commission_bps: float,
    slippage_bps: float,
    stamp_tax_bps: float,
    transfer_fee_bps: float,
) -> dict | None:
    g = g.sort_values("date").reset_index(drop=True)
    idxs = g.index[g["date"].eq(signal_date)].tolist()
    if not idxs:
        return None
    idx = idxs[0]
    entry_idx = idx + 1
    if entry_idx >= len(g):
        return None
    entry = g.loc[entry_idx]
    entry_open = float(entry["open"])
    if not math.isfinite(entry_open) or entry_open <= 0:
        return None

    last_exit_idx = min(entry_idx + hold_days, len(g) - 1)
    if last_exit_idx <= entry_idx:
        return None

    stop_price = entry_open * (1 + stop_loss)
    take_price = entry_open * (1 + take_profit)
    exit_idx = last_exit_idx
    exit_price = float(g.loc[last_exit_idx, "close"])
    exit_reason = "timeout"

    # A-share T+1: do not sell on the entry day. If stop and take-profit both hit
    # on the same later daily bar, assume the stop is hit first.
    for j in range(entry_idx + 1, last_exit_idx + 1):
        row = g.loc[j]
        if float(row["low"]) <= stop_price:
            exit_idx = j
            exit_price = stop_price
            exit_reason = "stop"
            break
        if float(row["high"]) >= take_price:
            exit_idx = j
            exit_price = take_price
            exit_reason = "take_profit"
            break

    return {
        "entry_date": g.loc[entry_idx, "date"],
        "exit_date": g.loc[exit_idx, "date"],
        "entry_open": entry_open,
        "exit_price": exit_price,
        "exit_reason": exit_reason,
        "net_ret": net_exit_return(entry_open, exit_price, commission_bps, slippage_bps, stamp_tax_bps, transfer_fee_bps),
    }


def run_backtest(
    panel: pd.DataFrame,
    top_n: int,
    hold_days: int,
    step_days: int,
    stop_loss: float,
    take_profit: float,
    commission_bps: float,
    slippage_bps: float,
    stamp_tax_bps: float,
    transfer_fee_bps: float,
    complete_signal_date: pd.Timestamp,
) -> tuple[pd.DataFrame, pd.DataFrame]:
    dates = [pd.Timestamp(d) for d in sorted(panel["date"].drop_duplicates()) if d <= complete_signal_date]
    by_code = {code: g for code, g in panel.groupby("ts_code", sort=False)}
    rows = []
    for idx in range(160, len(dates), step_days):
        d = dates[idx]
        day = panel[(panel["date"].eq(d)) & (panel["summary_signal"])].copy()
        day = day.dropna(subset=["score"])
        if day.empty:
            rows.append({"signal_date": d, "portfolio_ret": 0.0, "names": "cash", "codes": "", "count": 0})
            continue
        pick = day.sort_values("score", ascending=False).head(top_n)
        trade_parts = []
        for _, row in pick.iterrows():
            sim = simulate_trade(
                by_code[row["ts_code"]],
                d,
                hold_days,
                stop_loss,
                take_profit,
                commission_bps,
                slippage_bps,
                stamp_tax_bps,
                transfer_fee_bps,
            )
            if sim is None:
                continue
            sim.update({"ts_code": row["ts_code"], "name": row["name"]})
            trade_parts.append(sim)
        if not trade_parts:
            rows.append({"signal_date": d, "portfolio_ret": 0.0, "names": "cash", "codes": "", "count": 0})
            continue
        rets = [x["net_ret"] for x in trade_parts]
        rows.append(
            {
                "signal_date": d,
                "entry_date": trade_parts[0]["entry_date"],
                "exit_date": trade_parts[0]["exit_date"],
                "portfolio_ret": float(sum(rets) / len(rets)),
                "names": " ".join(x["name"] for x in trade_parts),
                "codes": " ".join(x["ts_code"] for x in trade_parts),
                "exit_reasons": " ".join(x["exit_reason"] for x in trade_parts),
                "count": len(trade_parts),
            }
        )

    trades = pd.DataFrame(rows)
    if trades.empty:
        raise RuntimeError("No rebalance periods available.")
    trades["nav"] = (1 + trades["portfolio_ret"]).cumprod()

    benchmark = panel.groupby("date")["ret"].mean().dropna()
    benchmark = benchmark.loc[(benchmark.index >= trades["signal_date"].min()) & (benchmark.index <= trades["signal_date"].max())]
    bench_nav = (1 + benchmark).cumprod()
    equity = trades[["signal_date", "nav"]].rename(columns={"signal_date": "date"})
    equity = equity.merge(bench_nav.rename("benchmark_nav"), left_on="date", right_index=True, how="left")
    equity["benchmark_nav"] = equity["benchmark_nav"].ffill()
    return trades, equity


def max_drawdown(nav: pd.Series) -> float:
    dd = nav / nav.cummax() - 1
    return float(dd.min())


def metrics(trades: pd.DataFrame, equity: pd.DataFrame, step_days: int) -> dict:
    periods = len(trades)
    years = max(periods * step_days / 252, 1 / 252)
    final_nav = float(equity["nav"].iloc[-1])
    ann = final_nav ** (1 / years) - 1
    win_rate = float((trades["portfolio_ret"] > 0).mean())
    return {
        "periods": periods,
        "final_nav": final_nav,
        "annual_return": ann,
        "max_drawdown": max_drawdown(equity["nav"]),
        "win_rate": win_rate,
        "avg_period_return": float(trades["portfolio_ret"].mean()),
        "benchmark_final_nav": float(equity["benchmark_nav"].iloc[-1]) if not math.isnan(equity["benchmark_nav"].iloc[-1]) else None,
        "benchmark_max_drawdown": max_drawdown(equity["benchmark_nav"].dropna()) if equity["benchmark_nav"].notna().any() else None,
    }


def save_chart(equity: pd.DataFrame, out: Path) -> None:
    plt.figure(figsize=(12, 6))
    plt.plot(equity["date"], equity["nav"], label="K-line signal summary strategy")
    if "benchmark_nav" in equity:
        plt.plot(equity["date"], equity["benchmark_nav"], label="Equal-weight scanned universe", alpha=0.75)
    plt.title("K-line Indicator Signal Summary Backtest")
    plt.xlabel("Date")
    plt.ylabel("Net Asset Value")
    plt.grid(True, alpha=0.25)
    plt.legend()
    plt.tight_layout()
    plt.savefig(out, dpi=160)
    plt.close()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--max-stocks", type=int, default=0, help="0 means all eligible A shares in build/stock_basic.json")
    parser.add_argument("--limit", type=int, default=520)
    parser.add_argument("--workers", type=int, default=20)
    parser.add_argument("--top-n", type=int, default=5)
    parser.add_argument("--hold-days", type=int, default=5)
    parser.add_argument("--step-days", type=int, default=5)
    parser.add_argument("--stop-loss", type=float, default=-0.055)
    parser.add_argument("--take-profit", type=float, default=0.095)
    parser.add_argument("--commission-bps", type=float, default=2.5)
    parser.add_argument("--slippage-bps", type=float, default=5.0)
    parser.add_argument("--stamp-tax-bps", type=float, default=5.0)
    parser.add_argument("--transfer-fee-bps", type=float, default=0.2)
    parser.add_argument("--input-csv", type=Path, default=None)
    args = parser.parse_args()

    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    if args.input_csv:
        panel = load_panel_csv(args.input_csv)
        universe_size = int(panel["ts_code"].nunique())
        print(f"loaded panel={args.input_csv}, stocks={universe_size}, rows={len(panel)}")
    else:
        universe = load_universe(args.max_stocks)
        universe_size = len(universe)
        print(f"universe={len(universe)}")
        panel = build_panel(universe, args.limit, args.workers)
    panel = add_factors(panel)
    complete_signal_date = latest_complete_signal_date(panel)

    latest = panel[(panel["date"].eq(complete_signal_date)) & panel["summary_signal"]].copy()
    latest = latest.sort_values("score", ascending=False).head(50)
    latest_cols = [
        "date",
        "ts_code",
        "name",
        "industry",
        "close",
        "score",
        "signal_total",
        "signal_bullish",
        "signal_bearish",
        "signal_oscillating",
        "signal_neutral",
        "bullish_pct",
        "bearish_pct",
        "signal_spread",
        "ret5",
        "ret20",
        "dist_ma20",
        "dist_ma60",
        "volume_ratio_prev5",
        "turnover",
        "amount_ma20",
    ]

    trades, equity = run_backtest(
        panel,
        args.top_n,
        args.hold_days,
        args.step_days,
        args.stop_loss,
        args.take_profit,
        args.commission_bps,
        args.slippage_bps,
        args.stamp_tax_bps,
        args.transfer_fee_bps,
        complete_signal_date,
    )
    stats = metrics(trades, equity, args.step_days)
    stats.update(
        {
            "complete_signal_date": str(complete_signal_date.date()),
            "universe_size": universe_size,
            "usable_stocks": int(panel["ts_code"].nunique()),
            "strategy": {
                "top_n": args.top_n,
                "hold_days": args.hold_days,
                "rebalance_step_days": args.step_days,
                "stop_loss": args.stop_loss,
                "take_profit": args.take_profit,
                "signal_rule": "oversold-turn: signal_total>=18, bullish_pct>=28%, bearish_pct<=36%, oscillating_pct>=22%, spread>=-4%, low-position/volume/liquidity filters",
            },
            "cost_model": {
                "commission_bps_each_side": args.commission_bps,
                "slippage_bps_each_side": args.slippage_bps,
                "stamp_tax_bps_sell_side": args.stamp_tax_bps,
                "transfer_fee_bps_each_side": args.transfer_fee_bps,
            },
        }
    )

    candidates_path = OUTPUT_DIR / "latest_candidates.csv"
    trades_path = OUTPUT_DIR / "backtest_trades.csv"
    equity_path = OUTPUT_DIR / "backtest_equity.csv"
    stats_path = OUTPUT_DIR / "backtest_stats.json"
    chart_path = OUTPUT_DIR / "backtest_nav.png"
    sample_path = OUTPUT_DIR / "signal_panel_tail.csv"

    latest[latest_cols].to_csv(candidates_path, index=False, encoding="utf-8-sig")
    trades.to_csv(trades_path, index=False, encoding="utf-8-sig")
    equity.to_csv(equity_path, index=False, encoding="utf-8-sig")
    stats_path.write_text(json.dumps(stats, ensure_ascii=False, indent=2), encoding="utf-8")
    save_chart(equity, chart_path)
    panel[panel["date"].ge(complete_signal_date - pd.Timedelta(days=20))][latest_cols + ["summary_signal"]].to_csv(sample_path, index=False, encoding="utf-8-sig")

    print(json.dumps({"complete_signal_date": str(complete_signal_date.date()), "stats": stats}, ensure_ascii=False, indent=2))
    print(f"candidates={candidates_path}")
    print(f"trades={trades_path}")
    print(f"equity={equity_path}")
    print(f"chart={chart_path}")


if __name__ == "__main__":
    main()
