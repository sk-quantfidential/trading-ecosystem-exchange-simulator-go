"""Combine bottom-up signals with a top-down thesis into ranked candidates.

Pipeline (all as of a single decision date; execution is the next open):
  1. Compute each signal (momentum, earnings, catalyst) per stock.
  2. Cross-sectionally standardise each signal across the candidate set.
  3. Combine into one score: sum(signal_weight * z_score).
  4. Apply the thesis as an additive tilt in z-space (ln of its multiplier);
     a thesis multiplier of 0 drops the name.
  5. Rank; return the top N candidates for portfolio construction.

A candidate record is a plain dict so any data source can feed it:
    {
      "ticker": "AZN.L",
      "meta":   {"sector": "Health Care", "tags": ["ai_immune", "pricing_power"]},
      "closes": [...],   # daily closes truncated to the DECISION-day close
      "earnings": [{"date": date, "surprise": 0.08, "reaction": 0.05}],  # optional
      "events":   [{"date": date, "type": "buyback", "magnitude": 1.0}],  # optional
    }
"""

from __future__ import annotations

import math
from dataclasses import dataclass
from datetime import date
from typing import Mapping, Sequence

from .signals.base import standardize
from .signals.catalyst import catalyst_signal
from .signals.earnings import earnings_signal
from .signals.momentum import momentum_signal
from .signals.thesis import Thesis

DEFAULT_SIGNAL_WEIGHTS: dict[str, float] = {
    "momentum": 1.0,
    "earnings": 1.0,
    "catalyst": 1.0,
}


@dataclass
class CandidateScore:
    ticker: str
    final: float
    combined_z: float
    per_signal: dict[str, float]
    thesis_mult: float
    rationale: str


def select_candidates(
    records: Sequence[Mapping],
    as_of: date,
    signal_weights: Mapping[str, float] | None = None,
    thesis: Thesis | None = None,
    thesis_weight: float = 1.0,
    top_n: int = 20,
) -> list[CandidateScore]:
    weights = dict(DEFAULT_SIGNAL_WEIGHTS)
    if signal_weights:
        weights.update(signal_weights)

    # 1. raw per-signal scores
    raw: dict[str, dict[str, float]] = {"momentum": {}, "earnings": {}, "catalyst": {}}
    for r in records:
        t = r["ticker"]
        ms = momentum_signal(t, r.get("closes", []))
        if ms:
            raw["momentum"][t] = ms.score
        if r.get("earnings"):
            es = earnings_signal(t, r["earnings"], as_of)
            if es:
                raw["earnings"][t] = es.score
        if r.get("events"):
            cs = catalyst_signal(t, r["events"], as_of)
            if cs:
                raw["catalyst"][t] = cs.score

    # 2. standardise each signal across the cross-section
    z = {sig: standardize(vals) for sig, vals in raw.items()}

    # 3-4. combine + thesis tilt
    results: list[CandidateScore] = []
    for r in records:
        t = r["ticker"]
        combined = 0.0
        per_signal: dict[str, float] = {}
        for sig, w in weights.items():
            if t in z.get(sig, {}):
                per_signal[sig] = round(z[sig][t], 3)
                combined += w * z[sig][t]
        if not per_signal:
            continue  # no signal fired for this name

        mult, th_rationale = (1.0, "no thesis")
        if thesis is not None:
            mult, th_rationale = thesis.multiplier(r.get("meta", {}))
            if mult <= 0.0:
                continue  # excluded by thesis

        final = combined + (thesis_weight * math.log(mult) if thesis is not None else 0.0)
        rationale = (
            f"{t}: signals {per_signal} (combined z {combined:+.2f}); "
            f"thesis x{mult:.2f} [{th_rationale}] -> final {final:+.2f}"
        )
        results.append(CandidateScore(t, final, combined, per_signal, mult, rationale))

    results.sort(key=lambda c: c.final, reverse=True)
    return results[:top_n]
