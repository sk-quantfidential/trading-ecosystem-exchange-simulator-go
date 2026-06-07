"""Eligible-stock universe (LSE / Nasdaq / NYSE / main European exchanges).

The game restricts you to a curated list. Maintain it as a CSV and load it here.
CSV columns: ticker,name,exchange,currency,sector,tags
``tags`` is a ``;``-separated list (e.g. "ai_immune;pricing_power").
"""

from __future__ import annotations

import csv
from dataclasses import dataclass, field
from pathlib import Path


@dataclass
class Security:
    ticker: str
    name: str
    exchange: str
    currency: str
    sector: str
    tags: list[str] = field(default_factory=list)

    @property
    def meta(self) -> dict:
        """Metadata shape consumed by the thesis filter / selector."""
        return {"sector": self.sector, "tags": self.tags}


def load_universe(path: str | Path) -> list[Security]:
    rows: list[Security] = []
    with open(path, newline="", encoding="utf-8") as fh:
        for r in csv.DictReader(fh):
            tags = [t.strip() for t in (r.get("tags") or "").split(";") if t.strip()]
            rows.append(
                Security(
                    ticker=r["ticker"].strip(),
                    name=r.get("name", "").strip(),
                    exchange=r.get("exchange", "").strip(),
                    currency=r.get("currency", "").strip(),
                    sector=r.get("sector", "").strip(),
                    tags=tags,
                )
            )
    return rows


def eligible_tickers(universe: list[Security]) -> set[str]:
    return {s.ticker for s in universe}


def meta_by_ticker(universe: list[Security]) -> dict[str, dict]:
    return {s.ticker: s.meta for s in universe}
