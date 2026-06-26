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
OUTPUT_DIR = ROOT / "outputs" / "low_absorb_burst"


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

    universe.sort(key=lambda x: (x["ts_code"]))
    return universe[:max_stocks] if max_stocks else universe


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
                        "amplitude": float(p[7]),
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
            time.sleep(0.4 + attempt * 0.6)
    return None


def build_panel(universe: list[dict], limit: int, workers: int) -> pd.DataFrame:
    frames: list[pd.DataFrame] = []
    total = len(universe)
    with futures.ThreadPoolExecutor(max_workers=workers) as pool:
        jobs = {pool.submit(fetch_kline, stock, limit): stock for stock in universe}
        for i, job in enumerate(futures.as_completed(jobs), 1):
            df = job.result()
            if df is not None and len(df) >= 120:
                frames.append(df)
            if i % 50 == 0 or i == total:
                print(f"fetched {i}/{total}, usable={len(frames)}")
    if not frames:
        raise RuntimeError("No usable kline data fetched.")
    return pd.concat(frames, ignore_index=True).sort_values(["ts_code", "date"])


def add_factors(panel: pd.DataFrame, hold_days: int) -> pd.DataFrame:
    def per_stock(g: pd.DataFrame) -> pd.DataFrame:
        g = g.copy()
        close = g["close"]
        amount = g["amount"]
        ret = close.pct_change()
        g["ret"] = ret
        g["ma5"] = close.rolling(5).mean()
        g["ma10"] = close.rolling(10).mean()
        g["ma20"] = close.rolling(20).mean()
        g["ma60"] = close.rolling(60).mean()
        g["high20"] = close.rolling(20).max()
        g["low20"] = close.rolling(20).min()
        g["drawdown20"] = close / g["high20"] - 1
        g["range20"] = g["high20"] / g["low20"] - 1
        g["dist_ma20"] = close / g["ma20"] - 1
        g["ret3"] = close / close.shift(3) - 1
        g["ret5"] = close / close.shift(5) - 1
        g["vol5"] = ret.rolling(5).std()
        g["vol20"] = ret.rolling(20).std()
        g["vol_contract"] = g["vol5"] / g["vol20"]
        g["amount_ma20"] = amount.rolling(20).mean()
        g["amount_ratio"] = amount / g["amount_ma20"]
        g["burst20"] = g["amount_ratio"].rolling(20).max()
        g["next_date"] = g["date"].shift(-1)
        g["exit_date"] = g["date"].shift(-hold_days)
        g["next_open"] = g["open"].shift(-1)
        g["exit_close"] = g["close"].shift(-hold_days)
        return g

    panel = pd.concat([per_stock(g) for _, g in panel.groupby("ts_code", sort=False)], ignore_index=True)
    panel["liquid"] = panel["amount_ma20"] >= 80_000_000

    signal = (
        panel["liquid"]
        & (panel["ma20"] > panel["ma60"] * 0.98)
        & (panel["close"] > panel["ma60"] * 0.92)
        & panel["drawdown20"].between(-0.18, -0.03)
        & panel["dist_ma20"].between(-0.08, 0.035)
        & (panel["ret5"] > -0.09)
        & (panel["vol_contract"] < 0.95)
        & (panel["burst20"] > 1.7)
        & (panel["amount_ratio"] > 0.85)
        & (panel["turnover"] > 0.006)
    )
    panel["signal"] = signal.fillna(False)

    pullback_score = 1 - (panel["drawdown20"].abs() - 0.08).abs() / 0.12
    contraction_score = 1 - panel["vol_contract"].clip(0, 1.5) / 1.5
    recovery_score = (panel["ret3"].fillna(0) * 8) + (panel["dist_ma20"].fillna(0) * -2)
    burst_score = (panel["burst20"].clip(1, 4) - 1) / 3
    liquidity_score = (panel["amount_ma20"].clip(80_000_000, 800_000_000) / 800_000_000)
    panel["score"] = (
        0.28 * pullback_score
        + 0.24 * burst_score
        + 0.20 * contraction_score
        + 0.18 * recovery_score
        + 0.10 * liquidity_score
    )
    return panel


def max_drawdown(nav: pd.Series) -> float:
    dd = nav / nav.cummax() - 1
    return float(dd.min())


def latest_complete_signal_date(panel: pd.DataFrame, today: pd.Timestamp | None = None) -> pd.Timestamp:
    dates = sorted(panel["date"].drop_duplicates())
    if not dates:
        raise RuntimeError("No dates available.")
    today = today or pd.Timestamp.today().normalize()
    latest = pd.Timestamp(dates[-1]).normalize()
    if latest >= today and len(dates) >= 2:
        return pd.Timestamp(dates[-2])
    return pd.Timestamp(dates[-1])


def net_trade_return(
    entry_open: pd.Series,
    exit_close: pd.Series,
    commission_bps: float,
    slippage_bps: float,
    stamp_tax_bps: float,
    transfer_fee_bps: float,
) -> pd.Series:
    commission = commission_bps / 10000.0
    slippage = slippage_bps / 10000.0
    stamp_tax = stamp_tax_bps / 10000.0
    transfer_fee = transfer_fee_bps / 10000.0

    buy_price = entry_open * (1 + slippage)
    sell_price = exit_close * (1 - slippage)
    buy_cost_multiplier = 1 + commission + transfer_fee
    sell_proceeds_multiplier = 1 - commission - stamp_tax - transfer_fee
    return (sell_price * sell_proceeds_multiplier) / (buy_price * buy_cost_multiplier) - 1


def run_backtest(
    panel: pd.DataFrame,
    top_n: int,
    hold_days: int,
    step_days: int,
    commission_bps: float,
    slippage_bps: float,
    stamp_tax_bps: float,
    transfer_fee_bps: float,
    complete_signal_date: pd.Timestamp,
) -> tuple[pd.DataFrame, pd.DataFrame]:
    dates = [d for d in sorted(panel["date"].drop_duplicates()) if d <= complete_signal_date]
    trades = []
    for idx in range(80, len(dates), step_days):
        d = dates[idx]
        day = panel[(panel["date"] == d) & (panel["signal"])].copy()
        day = day.dropna(subset=["score", "next_date", "exit_date", "next_open", "exit_close"])
        if day.empty:
            trades.append({"signal_date": d, "portfolio_ret": 0.0, "names": "cash", "count": 0})
            continue
        pick = day.sort_values("score", ascending=False).head(top_n)
        pick = pick.copy()
        pick["net_ret"] = net_trade_return(
            pick["next_open"],
            pick["exit_close"],
            commission_bps,
            slippage_bps,
            stamp_tax_bps,
            transfer_fee_bps,
        )
        net = pick["net_ret"].mean()
        trades.append(
            {
                "signal_date": d,
                "entry_date": pick["next_date"].iloc[0],
                "exit_date": pick["exit_date"].iloc[0],
                "portfolio_ret": float(net),
                "names": " ".join(pick["name"].tolist()),
                "codes": " ".join(pick["ts_code"].tolist()),
                "avg_entry_open": float(pick["next_open"].mean()),
                "avg_exit_close": float(pick["exit_close"].mean()),
                "count": len(pick),
            }
        )

    trade_df = pd.DataFrame(trades)
    if trade_df.empty:
        raise RuntimeError("No rebalance periods available for backtest.")
    trade_df["nav"] = (1 + trade_df["portfolio_ret"]).cumprod()

    by_date = panel.groupby("date")["ret"].mean().dropna()
    benchmark = by_date.loc[
        (by_date.index >= trade_df["signal_date"].min()) & (by_date.index <= trade_df["signal_date"].max())
    ]
    bench_nav = (1 + benchmark).cumprod()
    equity = trade_df[["signal_date", "nav"]].copy()
    equity = equity.rename(columns={"signal_date": "date"})
    equity = equity.merge(bench_nav.rename("benchmark_nav"), left_on="date", right_index=True, how="left")
    equity["benchmark_nav"] = equity["benchmark_nav"].ffill()
    return trade_df, equity


def metrics(trades: pd.DataFrame, equity: pd.DataFrame) -> dict:
    periods = len(trades)
    years = max(periods * 5 / 252, 1 / 252)
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
    }


def save_chart(equity: pd.DataFrame, out: Path) -> None:
    plt.figure(figsize=(12, 6))
    plt.plot(equity["date"], equity["nav"], label="Low absorb burst")
    if "benchmark_nav" in equity:
        plt.plot(equity["date"], equity["benchmark_nav"], label="Equal-weight scanned universe", alpha=0.7)
    plt.title("Low Absorb + Burst Backtest")
    plt.xlabel("Date")
    plt.ylabel("Net Asset Value")
    plt.grid(True, alpha=0.25)
    plt.legend()
    plt.tight_layout()
    plt.savefig(out, dpi=160)
    plt.close()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--max-stocks", type=int, default=600)
    parser.add_argument("--limit", type=int, default=460)
    parser.add_argument("--workers", type=int, default=12)
    parser.add_argument("--top-n", type=int, default=5)
    parser.add_argument("--hold-days", type=int, default=5)
    parser.add_argument("--step-days", type=int, default=5)
    parser.add_argument("--commission-bps", type=float, default=2.5)
    parser.add_argument("--slippage-bps", type=float, default=5.0)
    parser.add_argument("--stamp-tax-bps", type=float, default=5.0)
    parser.add_argument("--transfer-fee-bps", type=float, default=0.2)
    args = parser.parse_args()

    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    universe = load_universe(args.max_stocks)
    print(f"universe={len(universe)}")

    panel = build_panel(universe, args.limit, args.workers)
    panel = add_factors(panel, args.hold_days)

    complete_signal_date = latest_complete_signal_date(panel)
    candidates = (
        panel[(panel["date"] == complete_signal_date) & panel["signal"]]
        .sort_values("score", ascending=False)
        .head(30)
        [
            [
                "date",
                "ts_code",
                "name",
                "industry",
                "close",
                "score",
                "drawdown20",
                "dist_ma20",
                "ret3",
                "ret5",
                "amount_ratio",
                "burst20",
                "vol_contract",
                "turnover",
            ]
        ]
    )

    trades, equity = run_backtest(
        panel,
        args.top_n,
        args.hold_days,
        args.step_days,
        args.commission_bps,
        args.slippage_bps,
        args.stamp_tax_bps,
        args.transfer_fee_bps,
        complete_signal_date,
    )
    stats = metrics(trades, equity)
    stats.update(
        {
            "complete_signal_date": str(complete_signal_date.date()),
            "cost_model": {
                "commission_bps_each_side": args.commission_bps,
                "slippage_bps_each_side": args.slippage_bps,
                "stamp_tax_bps_sell_side": args.stamp_tax_bps,
                "transfer_fee_bps_each_side": args.transfer_fee_bps,
            },
        }
    )

    panel_path = OUTPUT_DIR / "factor_panel_sample.csv"
    candidates_path = OUTPUT_DIR / "latest_candidates.csv"
    trades_path = OUTPUT_DIR / "backtest_trades.csv"
    equity_path = OUTPUT_DIR / "backtest_equity.csv"
    stats_path = OUTPUT_DIR / "backtest_stats.json"
    chart_path = OUTPUT_DIR / "backtest_nav.png"

    panel.to_csv(panel_path, index=False, encoding="utf-8-sig")
    candidates.to_csv(candidates_path, index=False, encoding="utf-8-sig")
    trades.to_csv(trades_path, index=False, encoding="utf-8-sig")
    equity.to_csv(equity_path, index=False, encoding="utf-8-sig")
    stats_path.write_text(json.dumps(stats, ensure_ascii=False, indent=2), encoding="utf-8")
    save_chart(equity, chart_path)

    print(json.dumps({"complete_signal_date": str(complete_signal_date.date()), "stats": stats}, ensure_ascii=False, indent=2))
    print(f"candidates={candidates_path}")
    print(f"trades={trades_path}")
    print(f"equity={equity_path}")
    print(f"chart={chart_path}")


if __name__ == "__main__":
    main()
