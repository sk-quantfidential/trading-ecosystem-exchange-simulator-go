"""Structural / event catalyst signal.

Discrete corporate events (guidance changes, buybacks, M&A, contract wins,
regulatory decisions, broker actions) carry directional information that can play
out over days-to-weeks. We weight each event by type and decay it by recency.

Event records are plain dicts:
    {"date": date, "type": "guidance_raise", "magnitude": 1.0}
``magnitude`` (default 1.0) lets you scale an event's size/credibility.
"""

from __future__ import annotations

from datetime import date
from typing import Sequence

from .base import SignalScore

# Directional weights: positive = bullish catalyst, negative = bearish.
EVENT_WEIGHTS: dict[str, float] = {
    "mna_target": 1.2,          # company is a takeover target
    "guidance_raise": 1.0,
    "regulatory_approval": 1.0,
    "contract_win": 0.7,
    "buyback": 0.6,
    "dividend_raise": 0.4,
    "broker_upgrade": 0.4,
    "insider_buy": 0.3,
    "mna_acquirer": -0.3,        # acquirers often de-rate near-term
    "broker_downgrade": -0.5,
    "regulatory_setback": -1.0,
    "guidance_cut": -1.0,
    "profit_warning": -1.2,
}


def catalyst_signal(
    ticker: str,
    events: Sequence[dict],
    as_of: date,
    half_life_days: int = 10,
    horizon_days: int = 30,
) -> SignalScore | None:
    """Recency-decayed sum of weighted events on/before the decision date."""
    eligible = [
        e for e in events
        if e["date"] <= as_of and (as_of - e["date"]).days <= horizon_days
    ]
    if not eligible:
        return None

    score = 0.0
    contributions: dict[str, float] = {}
    for e in eligible:
        etype = e["type"]
        weight = EVENT_WEIGHTS.get(etype, 0.0)
        if weight == 0.0:
            continue
        magnitude = float(e.get("magnitude", 1.0))
        days = (as_of - e["date"]).days
        decay = 0.5 ** (days / half_life_days)
        contribution = weight * magnitude * decay
        score += contribution
        contributions[f"{etype}@{days}d"] = round(contribution, 4)

    if not contributions:
        return None

    rationale = f"{ticker}: catalysts {contributions} -> {score:+.2f}"
    return SignalScore(ticker, "catalyst", score, contributions, rationale)
