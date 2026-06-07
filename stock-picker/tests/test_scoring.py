"""Constant-mix scoring — pinned to the FT rules' worked example."""

from picker.scoring import (
    daily_returns_from_prices,
    portfolio_daily_returns,
    score_constant_mix,
    season_return,
)

APPROX = 1e-9


def test_rules_worked_example_25pct_up_40pct_gives_10pct():
    # "portfolio A holds 25% of share X. When X rises 40%, the portfolio gains 10%."
    weights = {"X": 0.25, "A": 0.25, "B": 0.25, "C": 0.20, "D": 0.05}
    returns = {"X": [0.40], "A": [0.0], "B": [0.0], "C": [0.0], "D": [0.0]}
    assert abs(score_constant_mix(weights, returns) - 0.10) < APPROX


def test_constant_mix_resets_weights_no_drift():
    # X: +40% then +10%. Constant-mix compounds 0.25*0.4 then 0.25*0.1.
    weights = {"X": 0.25, "A": 0.25, "B": 0.25, "C": 0.20, "D": 0.05}
    flat = [0.0, 0.0]
    returns = {"X": [0.40, 0.10], "A": flat, "B": flat, "C": flat, "D": flat}
    daily = portfolio_daily_returns(weights, returns)
    assert abs(daily[0] - 0.10) < APPROX
    assert abs(daily[1] - 0.025) < APPROX  # weight reset to 25%, NOT drifted up
    expected = (1.10) * (1.025) - 1.0
    assert abs(season_return(daily) - expected) < APPROX


def test_base_price_entry_day_uses_open_else_prev_close():
    opens = [10.0, 11.0]
    closes = [10.5, 12.0]
    # Entry on day 0: day0 = close/open, day1 = close/prev_close.
    r0 = daily_returns_from_prices(opens, closes, entry_index=0)
    assert abs(r0[0] - (10.5 / 10.0 - 1)) < APPROX
    assert abs(r0[1] - (12.0 / 10.5 - 1)) < APPROX
    # Entry on day 1: day0 not held (0), day1 = close/open (open base).
    r1 = daily_returns_from_prices(opens, closes, entry_index=1)
    assert r1[0] == 0.0
    assert abs(r1[1] - (12.0 / 11.0 - 1)) < APPROX


def test_overnight_gap_before_entry_is_not_captured():
    # Decision uses close=10; you buy at next open=13 (gap up). The +30% gap is
    # NOT earned — entry base is the open, so a flat day scores ~0.
    opens = [13.0]
    closes = [13.0]
    r = daily_returns_from_prices(opens, closes, entry_index=0)
    assert abs(r[0]) < APPROX
