import json

from kline_signal_summary_backtest import (
    OUTPUT_DIR,
    add_factors,
    latest_complete_signal_date,
    load_panel_csv,
    metrics,
    run_backtest,
)


def base_args():
    return {
        "top_n": 5,
        "hold_days": 5,
        "step_days": 5,
        "stop_loss": -0.055,
        "take_profit": 0.095,
        "commission_bps": 2.5,
        "slippage_bps": 5.0,
        "stamp_tax_bps": 5.0,
        "transfer_fee_bps": 0.2,
    }


def apply_variant(panel, name):
    p = panel.copy()
    if name == "pullback_confirm":
        p["summary_signal"] = (
            p["liquid"]
            & p["signal_total"].ge(18)
            & p["bullish_pct"].between(0.34, 0.58)
            & p["bearish_pct"].le(0.32)
            & p["signal_spread"].ge(0.08)
            & p["dist_ma20"].between(-0.065, 0.035)
            & p["ret5"].between(-0.085, 0.035)
            & p["ret20"].between(-0.12, 0.18)
            & p["volume_ratio_prev5"].between(0.9, 2.4)
            & p["turnover"].between(0.006, 0.18)
            & p["close"].gt(p["ma60"] * 0.93)
        ).fillna(False)
        p["score"] = (
            0.28 * (1 - ((p["dist_ma20"] + 0.015).abs() / 0.065).clip(0, 1)).fillna(0)
            + 0.22 * (1 - ((p["ret5"] + 0.025).abs() / 0.085).clip(0, 1)).fillna(0)
            + 0.18 * p["signal_spread"].fillna(0)
            + 0.14 * p["bullish_pct"].fillna(0)
            + 0.12 * ((p["volume_ratio_prev5"].clip(0.9, 2.4) - 0.9) / 1.5).fillna(0)
            + 0.06 * (1 - (p["ret20"].abs() / 0.18).clip(0, 1)).fillna(0)
        )
    elif name == "breakout_mild":
        p["summary_signal"] = (
            p["liquid"]
            & p["signal_total"].ge(18)
            & p["bullish_pct"].ge(0.50)
            & p["bearish_pct"].le(0.20)
            & p["signal_spread"].ge(0.28)
            & p["dist_ma20"].between(0.0, 0.065)
            & p["ret5"].between(-0.015, 0.065)
            & p["ret20"].between(0.0, 0.22)
            & p["volume_ratio_prev5"].between(1.05, 2.6)
            & p["turnover"].between(0.006, 0.18)
        ).fillna(False)
        p["score"] = (
            0.34 * p["signal_spread"].fillna(0)
            + 0.20 * p["bullish_pct"].fillna(0)
            + 0.18 * ((p["volume_ratio_prev5"].clip(1.05, 2.6) - 1.05) / 1.55).fillna(0)
            + 0.16 * (1 - ((p["dist_ma20"] - 0.025).abs() / 0.065).clip(0, 1)).fillna(0)
            + 0.12 * (1 - ((p["ret5"] - 0.025).abs() / 0.065).clip(0, 1)).fillna(0)
        )
    elif name == "oversold_turn":
        p["summary_signal"] = (
            p["liquid"]
            & p["signal_total"].ge(18)
            & p["bullish_pct"].ge(0.28)
            & p["bearish_pct"].le(0.36)
            & p["signal_spread"].ge(-0.04)
            & p["oscillating_pct"].ge(0.22)
            & p["dist_ma20"].between(-0.10, 0.015)
            & p["ret5"].between(-0.12, 0.015)
            & p["volume_ratio_prev5"].between(0.75, 1.8)
            & p["turnover"].between(0.006, 0.18)
        ).fillna(False)
        p["score"] = (
            0.25 * (1 - ((p["ret5"] + 0.055).abs() / 0.12).clip(0, 1)).fillna(0)
            + 0.24 * (1 - ((p["dist_ma20"] + 0.04).abs() / 0.10).clip(0, 1)).fillna(0)
            + 0.18 * p["oscillating_pct"].fillna(0)
            + 0.17 * p["bullish_pct"].fillna(0)
            + 0.10 * (1 - ((p["volume_ratio_prev5"] - 1.05).abs() / 1.0).clip(0, 1)).fillna(0)
            + 0.06 * p["signal_spread"].fillna(0)
        )
    else:
        raise ValueError(name)
    return p


def main():
    input_csv = OUTPUT_DIR / "kline_panel_1200.csv"
    panel = add_factors(load_panel_csv(input_csv))
    complete_signal_date = latest_complete_signal_date(panel)
    args = base_args()
    results = []
    for name in ["pullback_confirm", "breakout_mild", "oversold_turn"]:
        p = apply_variant(panel, name)
        trades, equity = run_backtest(p, complete_signal_date=complete_signal_date, **args)
        stats = metrics(trades, equity, args["step_days"])
        stats["variant"] = name
        stats["latest_candidates"] = int(p[(p["date"].eq(complete_signal_date)) & p["summary_signal"]].shape[0])
        results.append(stats)
        print(json.dumps(stats, ensure_ascii=False))
    out = OUTPUT_DIR / "variant_sweep.json"
    out.write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"wrote {out}")


if __name__ == "__main__":
    main()
