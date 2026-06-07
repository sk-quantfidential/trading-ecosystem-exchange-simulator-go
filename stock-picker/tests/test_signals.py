"""Signal engine: momentum, earnings lag, catalyst, thesis, selector."""

from datetime import date

from picker.baseline_pick import baseline_pick, records_from_panel
from picker.selector import select_candidates
from picker.signals.catalyst import catalyst_signal
from picker.signals.earnings import earnings_signal
from picker.signals.momentum import momentum_signal
from picker.signals.thesis import Thesis


def test_momentum_ranks_stronger_trend_higher():
    up = [100 * (1.01 ** i) for i in range(40)]      # steady uptrend
    flat = [100.0] * 40
    s_up = momentum_signal("UP", up)
    s_flat = momentum_signal("FLAT", flat)
    assert s_up.score > s_flat.score


def test_earnings_signal_ignores_events_after_decision_date():
    as_of = date(2026, 6, 5)
    future = [{"date": date(2026, 6, 10), "surprise": 0.2, "reaction": 0.1}]
    assert earnings_signal("X", future, as_of) is None  # lag guard

    past = [{"date": date(2026, 5, 25), "surprise": 0.2, "reaction": 0.1}]
    s = earnings_signal("X", past, as_of)
    assert s is not None and s.score > 0


def test_catalyst_positive_event_scores_positive_and_decays():
    as_of = date(2026, 6, 5)
    recent = catalyst_signal("X", [{"date": date(2026, 6, 4), "type": "guidance_raise"}], as_of)
    older = catalyst_signal("X", [{"date": date(2026, 5, 20), "type": "guidance_raise"}], as_of)
    assert recent.score > 0
    assert recent.score > older.score  # recency decay


def test_thesis_excludes_and_tilts():
    thesis = Thesis(
        name="t",
        sector_tilts={"Consumer Staples": 1.25},
        tag_tilts={"ai_immune": 1.2, "high_multiple_ai_capex": 0.6},
        exclude_sectors=["Real Estate"],
    )
    excl, _ = thesis.multiplier({"sector": "Real Estate", "tags": []})
    assert excl == 0.0
    boost, _ = thesis.multiplier({"sector": "Consumer Staples", "tags": ["ai_immune"]})
    assert boost > 1.0
    fade, _ = thesis.multiplier({"sector": "Information Technology", "tags": ["high_multiple_ai_capex"]})
    assert fade < 1.0


def test_records_from_panel_truncates_to_decision_day():
    panel = {"A": {"dates": ["d0", "d1", "d2", "d3"],
                   "opens": [1, 2, 3, 4], "closes": [1, 2, 3, 4]}}
    recs = records_from_panel(panel, as_of_index=1)
    assert recs[0]["closes"] == [1, 2]  # only up to and including index 1


def _trending_panel():
    # 6 names; A trends up hardest, F flat. 40 days.
    panel = {}
    rates = {"A": 1.02, "B": 1.015, "C": 1.01, "D": 1.005, "E": 1.0, "F": 1.0}
    for t, r in rates.items():
        closes = [100 * (r ** i) for i in range(40)]
        panel[t] = {"dates": [f"d{i}" for i in range(40)],
                    "opens": closes, "closes": closes}
    return panel


def test_selector_then_baseline_produces_valid_portfolio():
    panel = _trending_panel()
    recs = records_from_panel(panel, as_of_index=39)
    weights, candidates = baseline_pick(recs, date(2026, 6, 5), n=5, scheme="equal")
    assert len(weights) == 5
    assert abs(sum(weights.values()) - 1.0) < 1e-6
    # strongest trender should be selected
    assert "A" in weights
    assert candidates[0].ticker == "A"
