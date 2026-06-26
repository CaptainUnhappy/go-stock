import json
import time
from pathlib import Path

import pandas as pd
import requests


ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "outputs" / "low_absorb_burst"
CANDIDATES = OUT / "latest_candidates.csv"
PANEL = OUT / "factor_panel_sample.csv"


def code_to_stock_list_code(ts_code: str) -> str:
    return ts_code.split(".")[0]


def fetch_announcements(codes: list[str]) -> dict[str, list[dict]]:
    stock_list = ",".join(code_to_stock_list_code(c) for c in codes)
    url = (
        "https://np-anotice-stock.eastmoney.com/api/security/ann"
        "?page_size=80&page_index=1&ann_type=SHA%2CCYB%2CSZA%2CBJA%2CINV"
        "&client_source=web&f_node=0&stock_list="
        + stock_list
    )
    headers = {
        "User-Agent": "Mozilla/5.0",
        "Referer": "https://data.eastmoney.com/notices/hsa/5.html",
    }
    result = {c: [] for c in codes}
    try:
        data = requests.get(url, headers=headers, timeout=15).json()
        for item in ((data.get("data") or {}).get("list") or []):
            codes_in_item = [s.get("stock_code") for s in item.get("codes", []) if isinstance(s, dict)]
            for full_code in codes:
                raw = code_to_stock_list_code(full_code)
                if raw in codes_in_item:
                    columns = item.get("columns") or []
                    result[full_code].append(
                        {
                            "date": item.get("notice_date"),
                            "title": item.get("title"),
                            "type": columns[0].get("column_name") if columns else "",
                        }
                    )
    except Exception as exc:
        for c in codes:
            result[c].append({"error": str(exc)})
    return result


def fetch_reports(code: str) -> list[dict]:
    raw = code_to_stock_list_code(code)
    url = "https://reportapi.eastmoney.com/report/list2"
    body = {
        "beginTime": "2025-06-18",
        "endTime": "2026-06-18",
        "industryCode": "*",
        "ratingChange": "",
        "rating": "",
        "orgCode": None,
        "code": raw,
        "rcode": "",
        "pageSize": 20,
        "pageNo": 1,
        "p": 1,
        "pageNum": 1,
        "pageNumber": 1,
    }
    headers = {
        "User-Agent": "Mozilla/5.0",
        "Origin": "https://data.eastmoney.com",
        "Referer": "https://data.eastmoney.com/report/stock.jshtml",
        "Content-Type": "application/json",
    }
    try:
        data = requests.post(url, json=body, headers=headers, timeout=15).json()
        rows = data.get("data") or []
        out = []
        for r in rows[:10]:
            out.append(
                {
                    "date": r.get("publishDate") or r.get("publishTime"),
                    "title": r.get("title"),
                    "org": r.get("orgSName") or r.get("orgName"),
                    "rating": r.get("emRatingName") or r.get("rating"),
                    "rating_change": r.get("ratingChange"),
                }
            )
        return out
    except Exception as exc:
        return [{"error": str(exc)}]


def kline_summary(panel: pd.DataFrame, code: str) -> dict:
    g = panel[panel["ts_code"] == code].copy().sort_values("date")
    g["date"] = pd.to_datetime(g["date"])
    last = g.iloc[-1]
    recent20 = g.tail(20)
    recent60 = g.tail(60)
    return {
        "last_date": str(last["date"].date()),
        "close": round(float(last["close"]), 3),
        "ma20": round(float(last["ma20"]), 3),
        "ma60": round(float(last["ma60"]), 3),
        "drawdown20": round(float(last["drawdown20"]), 4),
        "dist_ma20": round(float(last["dist_ma20"]), 4),
        "ret3": round(float(last["ret3"]), 4),
        "ret5": round(float(last["ret5"]), 4),
        "ret20": round(float(last["close"] / recent20.iloc[0]["close"] - 1), 4),
        "ret60": round(float(last["close"] / recent60.iloc[0]["close"] - 1), 4),
        "amount_ratio": round(float(last["amount_ratio"]), 3),
        "burst20": round(float(last["burst20"]), 3),
        "turnover": round(float(last["turnover"]), 4),
        "vol_contract": round(float(last["vol_contract"]), 3),
        "recent_10": [
            {
                "date": str(r["date"].date()),
                "close": round(float(r["close"]), 3),
                "pct_chg": round(float(r["pct_chg"]), 4),
                "amount_ratio": round(float(r["amount_ratio"]), 3),
                "turnover": round(float(r["turnover"]), 4),
            }
            for _, r in g.tail(10).iterrows()
        ],
    }


def main() -> None:
    candidates = pd.read_csv(CANDIDATES).head(5)
    panel = pd.read_csv(PANEL)
    codes = candidates["ts_code"].tolist()
    anns = fetch_announcements(codes)
    result = []
    for _, row in candidates.iterrows():
        code = row["ts_code"]
        result.append(
            {
                "code": code,
                "name": row["name"],
                "industry": row["industry"],
                "strategy_score": float(row["score"]),
                "kline": kline_summary(panel, code),
                "announcements": anns.get(code, [])[:8],
                "reports": fetch_reports(code),
            }
        )
        time.sleep(0.2)
    path = OUT / "candidate_deep_dive.json"
    path.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
    print(path)


if __name__ == "__main__":
    main()
