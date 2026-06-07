"""Earnings-anomaly signal: post-earnings-announcement drift (PEAD).

Stocks that beat expectations tend to keep drifting in the direction of the
surprise for several weeks. We score the most recent earnings event that falls
within the drift window and on/before the decision date, decayed by how much of
the window has elapsed.

Event records are plain dicts so the signal stays data-source-agnostic:
    {"date": date, "surprise": 0.08, "reaction": 0.05}
    surprise = (actual - estimate) / |estimate|      (0.08 == an 8% beat)
    reaction = the 1-day price move on the announcement (confirmation)
"""

from __future__ import annotations

from datetime import date
from typing import Sequence

from .base import SignalScore


def earnings_signal(
    ticker: str,
    events: Sequence[dict],
    as_of: date,
    drift_days: int = 45,
) -> SignalScore | None:
    """Score PEAD from the latest qualifying earnings event, else None.

    Only events with ``date <= as_of`` within ``drift_days`` are eligible — this
    is the lag guard (no peeking at announcements after the decision date).
    """
    eligible = [
        e for e in events
        if e["date"] <= as_of and (as_of - e["date"]).days <= drift_days
    ]
    if not eligible:
        return None

    e = max(eligible, key=lambda x: x["date"])
    days_since = (as_of - e["date"]).days
    decay = max(0.0, 1.0 - days_since / drift_days)

    surprise = float(e.get("surprise", 0.0))
    reaction = float(e.get("reaction", 0.0))
    # Drift is strongest when the fundamental surprise and the market's reaction
    # agree; average them and decay over the remaining drift window.
    score = 0.5 * (surprise + reaction) * decay

    components = {
        "surprise": surprise,
        "reaction": reaction,
        "days_since": float(days_since),
        "decay": decay,
    }
    rationale = (
        f"{ticker}: earnings {days_since}d ago, surprise {surprise:+.1%}, "
        f"reaction {reaction:+.1%}, drift decay {decay:.2f}"
    )
    return SignalScore(ticker, "earnings", score, components, rationale)
