"""Cross-sectional factor enrichment (Phase 1, optional — needs pandas).

The stdlib momentum signal covers the 8-week horizon's main driver. This module
adds richer cross-sectional factors when pandas is available. pandas is imported
lazily so the rest of the package works without it.
"""

from __future__ import annotations

from typing import Mapping


def _pd():
    try:
        import pandas as pd  # noqa: WPS433
        return pd
    except ImportError as exc:  # pragma: no cover - environment dependent
        raise RuntimeError("pandas not installed. Run: pip install '.[data]'") from exc


def closes_frame(panel: Mapping[str, Mapping]):
    """Build a dates x tickers DataFrame of closes from an aligned panel."""
    pd = _pd()
    data = {t: s["closes"] for t, s in panel.items()}
    index = next(iter(panel.values()))["dates"]
    return pd.DataFrame(data, index=index)


def factor_frame(panel: Mapping[str, Mapping], as_of_index: int | None = None):
    """Return a tickers x factors DataFrame (momentum, low-vol) as of a row.

    Factors are point-in-time: computed only from rows up to ``as_of_index``.
    """
    pd = _pd()
    closes = closes_frame(panel)
    if as_of_index is not None:
        closes = closes.iloc[: as_of_index + 1]
    rets = closes.pct_change()

    factors = pd.DataFrame(index=closes.columns)
    factors["mom_20d"] = closes.iloc[-1] / closes.iloc[-21] - 1.0 if len(closes) > 21 else float("nan")
    factors["mom_60d"] = closes.iloc[-1] / closes.iloc[-61] - 1.0 if len(closes) > 61 else float("nan")
    factors["vol_20d"] = rets.tail(20).std()
    # low-vol is attractive -> invert; z-score each factor cross-sectionally
    z = (factors - factors.mean()) / factors.std(ddof=0)
    z["vol_20d"] = -z["vol_20d"]
    factors["composite"] = z.mean(axis=1)
    return factors.sort_values("composite", ascending=False)
