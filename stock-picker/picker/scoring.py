"""Constant-mix, daily-rebalanced scoring — the FT Stock Picking Game mechanic.

The game is NOT buy-and-hold. Each day the portfolio's percentage weights reset
to your targets (auto-rebalanced at the official close). So:

    portfolio_daily_return = sum_i( target_weight_i * daily_return_i )
    season_return          = product_d( 1 + portfolio_daily_return_d ) - 1

Per-holding base-price rule for daily_return_i:
  * Held from the previous day -> base = previous CLOSE  (close-to-close).
  * Added / increased via reweighting on day d -> base = that day's OPEN
    (the trade fills at the next open), so day d uses open-to-close.

Worked example from the rules: a 25% holding that rises 40% contributes
0.25 * 0.40 = +10% to the portfolio that day, then its weight snaps back to 25%.

Everything here is stdlib-only and currency-agnostic: returns are native-currency
percentages (the game appears to neutralise FX — confirm before relying on it).
"""

from __future__ import annotations

from typing import Mapping, Sequence


def daily_returns_from_prices(
    opens: Sequence[float],
    closes: Sequence[float],
    entry_index: int = 0,
) -> list[float]:
    """Per-day return series for ONE holding under the game's base-price rules.

    Args:
        opens, closes: aligned daily open/close prices for the holding.
        entry_index: the day the position is bought or reweighted in. On that day
            the base is the OPEN; afterwards the base is the previous CLOSE. Days
            before ``entry_index`` return 0.0 (not held yet).
    """
    if len(opens) != len(closes):
        raise ValueError("opens and closes must be the same length")
    n = len(closes)
    rets = [0.0] * n
    for d in range(n):
        if d < entry_index:
            rets[d] = 0.0
        elif d == entry_index:
            if opens[d] == 0:
                raise ValueError(f"zero open price at entry day {d}")
            rets[d] = closes[d] / opens[d] - 1.0
        else:
            if closes[d - 1] == 0:
                raise ValueError(f"zero prev-close at day {d}")
            rets[d] = closes[d] / closes[d - 1] - 1.0
    return rets


def portfolio_daily_returns(
    weights: Mapping[str, float],
    returns_by_ticker: Mapping[str, Sequence[float]],
) -> list[float]:
    """Constant-mix portfolio daily returns: sum_i(weight_i * return_i) each day.

    Weights are applied fresh every day (the game resets them at close), so a
    holding's percentage never drifts with its price.
    """
    if not weights:
        raise ValueError("weights is empty")
    missing = [t for t in weights if t not in returns_by_ticker]
    if missing:
        raise ValueError(f"no return series for: {missing}")
    lengths = {len(returns_by_ticker[t]) for t in weights}
    if len(lengths) != 1:
        raise ValueError("all return series must be the same length")
    n = lengths.pop()
    out = [0.0] * n
    for d in range(n):
        out[d] = sum(weights[t] * returns_by_ticker[t][d] for t in weights)
    return out


def season_return(daily_returns: Sequence[float]) -> float:
    """Compound a series of daily returns into a total (season) return."""
    acc = 1.0
    for r in daily_returns:
        acc *= 1.0 + r
    return acc - 1.0


def score_constant_mix(
    weights: Mapping[str, float],
    returns_by_ticker: Mapping[str, Sequence[float]],
) -> float:
    """Total season return for a fixed-target-weight, daily-rebalanced portfolio."""
    return season_return(portfolio_daily_returns(weights, returns_by_ticker))
