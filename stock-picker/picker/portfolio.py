"""Portfolio constraints and weight construction for the FT Stock Picking Game.

Hard rules (from stockpickinggame.ft.com/how-to-play):
  * at least MIN_HOLDINGS (5) stocks,
  * every weight in [MIN_WEIGHT, MAX_WEIGHT] = [5%, 25%],
  * weights sum to 100% (fully invested),
  * long-only,
  * tickers drawn from the eligible universe.

Never trust the LLM to satisfy these by arithmetic — build weights in code and
validate them, reject-and-retry on failure.
"""

from __future__ import annotations

from typing import Iterable, Mapping, Sequence

MIN_HOLDINGS = 5
MIN_WEIGHT = 0.05
MAX_WEIGHT = 0.25
SUM_TOL = 1e-6

# With a 5% floor / 25% cap, the feasible holding count is 5..20.
MIN_FEASIBLE_N = MIN_HOLDINGS                       # 1 / 0.20 -> 5 @ 20% each
MAX_FEASIBLE_N = int(round(1.0 / MIN_WEIGHT))       # 20 @ 5% each


def validate_weights(
    weights: Mapping[str, float],
    eligible: Iterable[str] | None = None,
) -> list[str]:
    """Return a list of rule violations (empty list == valid)."""
    errors: list[str] = []
    if len(weights) < MIN_HOLDINGS:
        errors.append(f"need >= {MIN_HOLDINGS} holdings, got {len(weights)}")
    for t, w in weights.items():
        if w < MIN_WEIGHT - SUM_TOL or w > MAX_WEIGHT + SUM_TOL:
            errors.append(f"{t} weight {w:.4f} outside [{MIN_WEIGHT}, {MAX_WEIGHT}]")
    total = sum(weights.values())
    if abs(total - 1.0) > SUM_TOL:
        errors.append(f"weights sum to {total:.6f}, must be 1.0")
    if any(w < 0 for w in weights.values()):
        errors.append("long-only: negative weights are not allowed")
    if eligible is not None:
        elig = set(eligible)
        for t in weights:
            if t not in elig:
                errors.append(f"{t} not in eligible universe")
    return errors


def is_valid(weights: Mapping[str, float], eligible: Iterable[str] | None = None) -> bool:
    return not validate_weights(weights, eligible)


def build_weights(
    ranked: Sequence[tuple[str, float]],
    n: int = MIN_HOLDINGS,
    scheme: str = "equal",
) -> dict[str, float]:
    """Build a rules-valid weight set from a score-ranked list of (ticker, score).

    scheme="equal": equal-weight the top ``n`` (e.g. 5 -> 20% each).
    scheme="tilt":  weight the top ``n`` proportionally to score, then project
                    onto the [5%, 25%] box that sums to 1.0 (water-filling).
    """
    if not (MIN_FEASIBLE_N <= n <= MAX_FEASIBLE_N):
        raise ValueError(f"n must be in [{MIN_FEASIBLE_N}, {MAX_FEASIBLE_N}], got {n}")
    if len(ranked) < n:
        raise ValueError(f"need >= {n} candidates, got {len(ranked)}")

    selected = list(ranked[:n])
    tickers = [t for t, _ in selected]

    if scheme == "equal":
        w = 1.0 / n
        return {t: round(w, 6) for t in tickers}

    if scheme != "tilt":
        raise ValueError(f"unknown scheme {scheme!r}")

    # Proportional to score, shifted so the worst selected name still gets weight.
    scores = {t: s for t, s in selected}
    lo = min(scores.values())
    base = {t: (scores[t] - lo) + 1e-6 for t in tickers}
    total = sum(base.values())
    weights = {t: base[t] / total for t in tickers}

    # Project onto the box [MIN_WEIGHT, MAX_WEIGHT] with sum == 1 (iterative).
    for _ in range(200):
        clipped = {t: min(max(weights[t], MIN_WEIGHT), MAX_WEIGHT) for t in tickers}
        s = sum(clipped.values())
        if abs(s - 1.0) <= 1e-12:
            weights = clipped
            break
        free = [t for t in tickers if MIN_WEIGHT < clipped[t] < MAX_WEIGHT]
        if not free:
            weights = {t: clipped[t] / s for t in tickers}  # last-resort renormalise
            break
        adj = (s - 1.0) / len(free)
        weights = dict(clipped)
        for t in free:
            weights[t] = clipped[t] - adj

    out = {t: round(weights[t], 6) for t in tickers}
    # Fix any tiny rounding residual on the largest in-bounds holding.
    residual = round(1.0 - sum(out.values()), 6)
    if residual:
        for t in sorted(out, key=lambda x: out[x], reverse=True):
            if MIN_WEIGHT <= out[t] + residual <= MAX_WEIGHT:
                out[t] = round(out[t] + residual, 6)
                break
    return out
