"""Rolling-window backtest with strict next-day-open execution.

At each decision day ``t`` it calls ``weight_fn(panel, t)`` (which must use only
data with index <= t), then enters the resulting portfolio at ``t+1``'s OPEN and
holds for ``window_days``, scoring with the constant-mix metric. This mirrors the
game's lag: decide on the close of T, fill at the open of T+1.
"""

from __future__ import annotations

from typing import Callable, Mapping

from .scoring import daily_returns_from_prices, score_constant_mix

Panel = Mapping[str, Mapping]
WeightFn = Callable[[Panel, int], dict[str, float]]


def backtest_constant_mix(
    panel: Panel,
    weight_fn: WeightFn,
    window_days: int = 40,   # ~8 trading weeks
    step_days: int = 5,
    min_history: int = 60,
) -> list[dict]:
    """Walk decision points and score each held window. Panel must be aligned."""
    if not panel:
        return []
    any_series = next(iter(panel.values()))
    dates = any_series["dates"]
    n_days = len(dates)

    results: list[dict] = []
    t = min_history
    while t + 1 + window_days <= n_days:
        weights = weight_fn(panel, t)
        returns_by_ticker: dict[str, list[float]] = {}
        ok = True
        for tk in weights:
            opens_w = panel[tk]["opens"][t + 1 : t + 1 + window_days]
            closes_w = panel[tk]["closes"][t + 1 : t + 1 + window_days]
            if len(closes_w) < window_days:
                ok = False
                break
            returns_by_ticker[tk] = daily_returns_from_prices(
                opens_w, closes_w, entry_index=0
            )
        if ok:
            results.append(
                {
                    "decision_date": dates[t],
                    "entry_date": dates[t + 1],
                    "score": score_constant_mix(weights, returns_by_ticker),
                    "weights": weights,
                }
            )
        t += step_days
    return results


def summarize(results: list[dict]) -> dict:
    """Aggregate backtest windows into headline stats."""
    if not results:
        return {"windows": 0}
    scores = [r["score"] for r in results]
    wins = sum(1 for s in scores if s > 0)
    return {
        "windows": len(scores),
        "mean_return": sum(scores) / len(scores),
        "median_return": sorted(scores)[len(scores) // 2],
        "best": max(scores),
        "worst": min(scores),
        "hit_rate": wins / len(scores),
    }
