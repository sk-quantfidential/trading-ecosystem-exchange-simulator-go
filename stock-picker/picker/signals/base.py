"""Shared types and helpers for signals."""

from __future__ import annotations

import statistics as st
from dataclasses import dataclass, field
from typing import Mapping


@dataclass
class SignalScore:
    """One signal's read on one stock, as of a decision date.

    ``score`` is a raw, signal-native number (e.g. a return). The selector
    cross-sectionally standardises scores across the candidate set before
    combining signals, so raw scales need not match across signals.
    """

    ticker: str
    name: str
    score: float
    components: dict[str, float] = field(default_factory=dict)
    rationale: str = ""


def standardize(values: Mapping[str, float]) -> dict[str, float]:
    """Cross-sectional z-scores. Constant or singleton input -> all zeros."""
    xs = list(values.values())
    if len(xs) < 2:
        return {k: 0.0 for k in values}
    mu = st.mean(xs)
    sd = st.pstdev(xs)
    if sd == 0:
        return {k: 0.0 for k in values}
    return {k: (v - mu) / sd for k, v in values.items()}
