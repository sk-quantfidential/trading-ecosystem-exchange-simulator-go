"""Phase 1 baseline picker: signals + (optional) thesis -> rules-valid portfolio."""

from __future__ import annotations

from datetime import date
from typing import Mapping, Sequence

from .portfolio import build_weights
from .selector import CandidateScore, select_candidates
from .signals.thesis import Thesis


def baseline_pick(
    records: Sequence[Mapping],
    as_of: date,
    thesis: Thesis | None = None,
    n: int = 5,
    scheme: str = "equal",
    signal_weights: Mapping[str, float] | None = None,
    top_n: int = 20,
) -> tuple[dict[str, float], list[CandidateScore]]:
    """Return (rules-valid weights, ranked candidates).

    Trades implied by these weights execute at the NEXT day's open.
    """
    candidates = select_candidates(
        records, as_of, signal_weights=signal_weights, thesis=thesis, top_n=top_n
    )
    if len(candidates) < n:
        raise ValueError(f"only {len(candidates)} candidates survived; need >= {n}")
    ranked = [(c.ticker, c.final) for c in candidates]
    weights = build_weights(ranked, n=n, scheme=scheme)
    return weights, candidates


def records_from_panel(
    panel: Mapping[str, Mapping],
    as_of_index: int,
    meta_by_ticker: Mapping[str, dict] | None = None,
    earnings_by_ticker: Mapping[str, list] | None = None,
    events_by_ticker: Mapping[str, list] | None = None,
) -> list[dict]:
    """Build selector records from a price panel, truncating closes to the decision day.

    Truncation at ``as_of_index`` is the lag guard — signals never see prices after
    the decision close.
    """
    meta = meta_by_ticker or {}
    earn = earnings_by_ticker or {}
    evts = events_by_ticker or {}
    records: list[dict] = []
    for t, s in panel.items():
        records.append(
            {
                "ticker": t,
                "meta": meta.get(t, {"sector": "", "tags": []}),
                "closes": list(s["closes"][: as_of_index + 1]),
                "earnings": earn.get(t, []),
                "events": evts.get(t, []),
            }
        )
    return records
