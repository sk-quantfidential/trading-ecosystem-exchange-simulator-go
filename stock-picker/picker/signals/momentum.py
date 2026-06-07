"""Momentum / severe-move signal.

Short-to-medium horizon momentum (the dominant driver over an 8-week season)
plus a "severity" z-score that flags an unusually large latest move relative to
the stock's own recent daily-return distribution.

IMPORTANT (lag): pass ``closes`` already truncated to the decision day — the last
element must be the close on the day you decide. The trade fills next open.
"""

from __future__ import annotations

import statistics as st
from typing import Sequence

from .base import SignalScore


def _ret(closes: Sequence[float], k: int) -> float | None:
    if len(closes) <= k or closes[-1 - k] == 0:
        return None
    return closes[-1] / closes[-1 - k] - 1.0


def momentum_signal(
    ticker: str,
    closes: Sequence[float],
    lookbacks: tuple[int, ...] = (5, 10, 20),
    severity_window: int = 60,
) -> SignalScore | None:
    """Average lookback return as the score; latest-move z-score as ``severity_z``.

    Returns None if there isn't enough history for any lookback.
    """
    components: dict[str, float] = {}
    rs: list[float] = []
    for k in lookbacks:
        r = _ret(closes, k)
        if r is not None:
            components[f"ret_{k}d"] = r
            rs.append(r)
    if not rs:
        return None

    momentum = sum(rs) / len(rs)

    daily = [
        closes[i] / closes[i - 1] - 1.0
        for i in range(1, len(closes))
        if closes[i - 1]
    ]
    severity = 0.0
    tail = daily[-severity_window:]
    if len(tail) >= 10 and st.pstdev(tail) > 0:
        severity = (daily[-1] - st.mean(tail)) / st.pstdev(tail)
    components["severity_z"] = severity

    rationale = (
        f"{ticker}: avg momentum {momentum:+.1%} over {lookbacks} days; "
        f"latest-move severity z={severity:+.1f}"
    )
    return SignalScore(ticker, "momentum", momentum, components, rationale)
