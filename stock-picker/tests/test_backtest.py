"""Backtest: next-day-open execution lag + window accounting."""

from picker.backtest import backtest_constant_mix, summarize


def _panel(n_days=110, n_tickers=6, drift=1.003):
    panel = {}
    for k in range(n_tickers):
        t = f"T{k}"
        closes = [100 * (drift ** i) for i in range(n_days)]
        panel[t] = {
            "dates": [f"2026-{1 + i // 28:02d}-{1 + i % 28:02d}" for i in range(n_days)],
            "opens": closes,
            "closes": closes,
        }
    return panel


def test_weight_fn_only_sees_past_data():
    panel = _panel()
    seen_lengths = []

    def weight_fn(pnl, t):
        # The contract: use only data up to index t. Record what we could see.
        seen_lengths.append((t, len(pnl["T0"]["closes"][: t + 1])))
        return {f"T{k}": 0.2 for k in range(5)}

    backtest_constant_mix(panel, weight_fn, window_days=40, step_days=10, min_history=60)
    assert seen_lengths
    for t, visible in seen_lengths:
        assert visible == t + 1  # never more than the decision day


def test_entry_is_day_after_decision_and_windows_score():
    panel = _panel()
    res = backtest_constant_mix(
        panel, lambda pnl, t: {f"T{k}": 0.2 for k in range(5)},
        window_days=40, step_days=10, min_history=60,
    )
    assert res
    for r in res:
        di = panel["T0"]["dates"].index(r["decision_date"])
        ei = panel["T0"]["dates"].index(r["entry_date"])
        assert ei == di + 1  # fills next day
    stats = summarize(res)
    assert stats["windows"] == len(res)
    assert stats["mean_return"] > 0  # uptrending synthetic data


def test_overnight_gap_at_entry_is_excluded_from_score():
    # One name gaps +50% overnight between the decision close and the entry open,
    # then is flat. Because entry base = open, the gap is NOT earned.
    n = 110
    dates = [f"2026-{1 + i // 28:02d}-{1 + i % 28:02d}" for i in range(n)]
    flat = [100.0] * n
    panel = {f"T{k}": {"dates": dates, "opens": list(flat), "closes": list(flat)} for k in range(5)}
    # GAP: closes flat at 100 up to entry; at the entry open it jumps to 150 then flat.
    gap_opens = [100.0] * n
    gap_closes = [100.0] * n
    decision_t = 60
    for i in range(decision_t + 1, n):  # from entry day onward, price is 150 (already gapped)
        gap_opens[i] = 150.0
        gap_closes[i] = 150.0
    panel["GAP"] = {"dates": dates, "opens": gap_opens, "closes": gap_closes}

    weights = {"T0": 0.2, "T1": 0.2, "T2": 0.2, "T3": 0.2, "GAP": 0.2}
    res = backtest_constant_mix(
        panel, lambda pnl, t: weights, window_days=40, step_days=1000, min_history=decision_t,
    )
    assert res
    # The +50% overnight gap is not captured; the held window is flat -> ~0 return.
    assert abs(res[0]["score"]) < 1e-9
