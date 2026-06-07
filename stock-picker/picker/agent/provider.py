"""Data providers behind the agent's tools.

The agent never touches raw data — it calls tools, and tools read a ``DataProvider``.
This decouples the agent from the data source: use ``DictDataProvider`` /
``PanelDataProvider`` with synthetic data now, swap a live provider in later.

Every provider is responsible for the next-day-open LAG: anything it returns must
be knowable as of the decision date (it should pre-filter news/prices to ``as_of``).
"""

from __future__ import annotations

import statistics as st
from dataclasses import dataclass
from datetime import date
from typing import Mapping, Protocol, Sequence


class DataProvider(Protocol):
    def candidates(self) -> list[str]: ...
    def fundamentals(self, ticker: str) -> dict | None: ...
    def price_stats(self, ticker: str) -> dict | None: ...
    def factor_scores(self, ticker: str) -> dict | None: ...
    def news(self, ticker: str) -> list[dict]: ...


def _price_stats_from_closes(closes: Sequence[float]) -> dict | None:
    if len(closes) < 21:
        return None

    def ret(k: int) -> float | None:
        if len(closes) <= k or closes[-1 - k] == 0:
            return None
        return round(closes[-1] / closes[-1 - k] - 1.0, 4)

    daily = [closes[i] / closes[i - 1] - 1.0 for i in range(1, len(closes)) if closes[i - 1]]
    vol = round(st.pstdev(daily[-20:]), 4) if len(daily) >= 20 else None
    high = max(closes)
    drawdown = round(closes[-1] / high - 1.0, 4) if high else None
    return {
        "last_close": round(closes[-1], 4),
        "ret_5d": ret(5),
        "ret_20d": ret(20),
        "ret_60d": ret(60),
        "vol_20d": vol,
        "drawdown_from_high": drawdown,
    }


@dataclass
class DictDataProvider:
    """In-memory provider — handy for tests and offline development."""

    _candidates: list[str]
    _fundamentals: Mapping[str, dict] = None  # type: ignore[assignment]
    _price_stats: Mapping[str, dict] = None  # type: ignore[assignment]
    _factor_scores: Mapping[str, dict] = None  # type: ignore[assignment]
    _news: Mapping[str, list[dict]] = None  # type: ignore[assignment]

    def candidates(self) -> list[str]:
        return list(self._candidates)

    def fundamentals(self, ticker: str) -> dict | None:
        return (self._fundamentals or {}).get(ticker)

    def price_stats(self, ticker: str) -> dict | None:
        return (self._price_stats or {}).get(ticker)

    def factor_scores(self, ticker: str) -> dict | None:
        return (self._factor_scores or {}).get(ticker)

    def news(self, ticker: str) -> list[dict]:
        return list((self._news or {}).get(ticker, []))


class PanelDataProvider:
    """Provider backed by a Phase-0 price panel (+ optional fundamentals/news).

    Closes are truncated to ``as_of_index`` and news/events to ``as_of`` so tools
    can't leak post-decision information.
    """

    def __init__(
        self,
        panel: Mapping[str, Mapping],
        as_of_index: int,
        as_of: date,
        fundamentals: Mapping[str, dict] | None = None,
        earnings: Mapping[str, list] | None = None,
        events: Mapping[str, list] | None = None,
    ) -> None:
        self.panel = panel
        self.as_of_index = as_of_index
        self.as_of = as_of
        self._fundamentals = fundamentals or {}
        self._earnings = earnings or {}
        self._events = events or {}

    def _closes(self, ticker: str) -> list[float]:
        return list(self.panel[ticker]["closes"][: self.as_of_index + 1])

    def candidates(self) -> list[str]:
        return list(self.panel.keys())

    def fundamentals(self, ticker: str) -> dict | None:
        return self._fundamentals.get(ticker)

    def price_stats(self, ticker: str) -> dict | None:
        if ticker not in self.panel:
            return None
        return _price_stats_from_closes(self._closes(ticker))

    def factor_scores(self, ticker: str) -> dict | None:
        from ..signals.momentum import momentum_signal

        if ticker not in self.panel:
            return None
        sig = momentum_signal(ticker, self._closes(ticker))
        if sig is None:
            return None
        return {"momentum_score": round(sig.score, 4), **{k: round(v, 4) for k, v in sig.components.items()}}

    def news(self, ticker: str) -> list[dict]:
        out: list[dict] = []
        for e in self._earnings.get(ticker, []):
            if e["date"] <= self.as_of:
                out.append({"date": e["date"].isoformat(), "type": "earnings", **{k: v for k, v in e.items() if k != "date"}})
        for e in self._events.get(ticker, []):
            if e["date"] <= self.as_of:
                out.append({"date": e["date"].isoformat(), "type": e.get("type"), "magnitude": e.get("magnitude", 1.0)})
        return sorted(out, key=lambda x: x["date"], reverse=True)
