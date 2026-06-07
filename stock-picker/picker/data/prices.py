"""Price data: live fetch (yfinance) + sqlite cache + stdlib CSV panel I/O.

A "panel" is the in-memory shape the scorer/backtester consume:
    {ticker: {"dates": [str, ...], "opens": [float, ...], "closes": [float, ...]}}

Live fetch needs ``pip install '.[data]'`` (pandas + yfinance). CSV panel
load/save and ``align_panel`` are stdlib so backtests run with no extra deps.
"""

from __future__ import annotations

import csv
import sqlite3
from pathlib import Path
from typing import Iterable

Panel = dict[str, dict[str, list]]


# --------------------------------------------------------------------------- #
# stdlib: CSV panel I/O + alignment
# --------------------------------------------------------------------------- #
def load_panel_from_csv(path: str | Path) -> Panel:
    """Load a long-format CSV (date,ticker,open,close) into a panel dict."""
    panel: Panel = {}
    with open(path, newline="", encoding="utf-8") as fh:
        for r in csv.DictReader(fh):
            t = r["ticker"]
            slot = panel.setdefault(t, {"dates": [], "opens": [], "closes": []})
            slot["dates"].append(r["date"])
            slot["opens"].append(float(r["open"]))
            slot["closes"].append(float(r["close"]))
    return panel


def save_panel_to_csv(panel: Panel, path: str | Path) -> None:
    with open(path, "w", newline="", encoding="utf-8") as fh:
        w = csv.writer(fh)
        w.writerow(["date", "ticker", "open", "close"])
        for t, s in panel.items():
            for d, o, c in zip(s["dates"], s["opens"], s["closes"]):
                w.writerow([d, t, o, c])


def align_panel(panel: Panel) -> tuple[list[str], Panel]:
    """Restrict all tickers to their common dates (so day indexes line up).

    Returns (common_dates, aligned_panel). Tickers without full coverage of the
    common dates are kept but only on the intersecting dates.
    """
    if not panel:
        return [], {}
    common = set.intersection(*(set(s["dates"]) for s in panel.values()))
    dates = sorted(common)
    aligned: Panel = {}
    for t, s in panel.items():
        idx = {d: i for i, d in enumerate(s["dates"])}
        aligned[t] = {
            "dates": dates,
            "opens": [s["opens"][idx[d]] for d in dates],
            "closes": [s["closes"][idx[d]] for d in dates],
        }
    return dates, aligned


# --------------------------------------------------------------------------- #
# sqlite cache
# --------------------------------------------------------------------------- #
_SCHEMA = """
CREATE TABLE IF NOT EXISTS prices (
    ticker TEXT, date TEXT, open REAL, close REAL,
    PRIMARY KEY (ticker, date)
);
"""


def _connect(cache_path: str | Path) -> sqlite3.Connection:
    conn = sqlite3.connect(str(cache_path))
    conn.execute(_SCHEMA)
    return conn


def cache_panel(panel: Panel, cache_path: str | Path = "prices.sqlite") -> None:
    conn = _connect(cache_path)
    with conn:
        for t, s in panel.items():
            conn.executemany(
                "INSERT OR REPLACE INTO prices(ticker,date,open,close) VALUES (?,?,?,?)",
                [(t, d, o, c) for d, o, c in zip(s["dates"], s["opens"], s["closes"])],
            )
    conn.close()


def load_cached(tickers: Iterable[str], cache_path: str | Path = "prices.sqlite") -> Panel:
    conn = _connect(cache_path)
    panel: Panel = {}
    for t in tickers:
        rows = conn.execute(
            "SELECT date,open,close FROM prices WHERE ticker=? ORDER BY date", (t,)
        ).fetchall()
        if rows:
            panel[t] = {
                "dates": [r[0] for r in rows],
                "opens": [r[1] for r in rows],
                "closes": [r[2] for r in rows],
            }
    conn.close()
    return panel


# --------------------------------------------------------------------------- #
# live fetch (lazy yfinance import)
# --------------------------------------------------------------------------- #
def fetch_prices(tickers: list[str], start: str, end: str) -> Panel:
    """Fetch daily open/close from yfinance. Requires ``pip install '.[data]'``."""
    try:
        import yfinance as yf  # noqa: WPS433 (lazy on purpose)
    except ImportError as exc:  # pragma: no cover - environment dependent
        raise RuntimeError(
            "yfinance not installed. Run: pip install '.[data]'"
        ) from exc

    panel: Panel = {}
    raw = yf.download(
        tickers, start=start, end=end, group_by="ticker",
        auto_adjust=False, progress=False,
    )
    for t in tickers:
        try:
            sub = raw[t] if len(tickers) > 1 else raw
            sub = sub.dropna(subset=["Open", "Close"])
            panel[t] = {
                "dates": [d.strftime("%Y-%m-%d") for d in sub.index],
                "opens": [float(x) for x in sub["Open"].tolist()],
                "closes": [float(x) for x in sub["Close"].tolist()],
            }
        except (KeyError, TypeError):  # pragma: no cover - data dependent
            continue
    return panel
