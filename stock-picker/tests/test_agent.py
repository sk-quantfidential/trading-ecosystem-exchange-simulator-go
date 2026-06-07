"""Phase 2 research agent — exercised fully offline (no SDK, no network)."""

import json
from datetime import date

from picker.agent import (
    DictDataProvider,
    PanelDataProvider,
    ResearchAgent,
    ScriptedLLMClient,
    ToolKit,
    parse_result,
    picks_to_weights,
    validate_result,
)
from picker.portfolio import is_valid


def _provider():
    candidates = ["A", "B", "C", "D", "E", "F"]
    return DictDataProvider(
        candidates,
        _fundamentals={"A": {"pe": 12.0, "roe": 0.25, "currency": "USD"}},
        _price_stats={"A": {"last_close": 100.0, "ret_20d": 0.08, "drawdown_from_high": -0.15}},
        _factor_scores={"A": {"momentum_score": 0.08}},
        _news={"B": [{"date": "2026-05-20", "type": "buyback"}]},
    )


def test_toolkit_returns_grounded_facts_and_unknowns():
    kit = ToolKit(_provider())
    f = json.loads(kit.dispatch("get_fundamentals", {"ticker": "A"}))
    assert f["pe"] == 12.0
    # Missing data is explicit "unknown", never invented.
    missing = json.loads(kit.dispatch("get_fundamentals", {"ticker": "Z"}))
    assert missing["fundamentals"] == "unknown"
    news = json.loads(kit.dispatch("get_recent_news", {"ticker": "B"}))
    assert news["news"][0]["type"] == "buyback"


def _good_final():
    return {
        "picks": [
            {"ticker": t, "conviction": c, "horizon_weeks": 8,
             "thesis": f"{t} thesis", "key_risks": "macro", "evidence": ["ret_20d=0.08"]}
            for t, c in [("A", 5), ("B", 4), ("C", 4), ("D", 3), ("E", 2)]
        ],
        "discarded": [{"ticker": "F", "reason": "weak momentum"}],
    }


def test_agent_runs_tool_loop_then_returns_validated_picks():
    kit = ToolKit(_provider())
    client = ScriptedLLMClient(
        rounds=[
            [("get_price_history", {"ticker": "A"}), ("get_fundamentals", {"ticker": "A"})],
            [("get_recent_news", {"ticker": "B"})],
        ],
        final=_good_final(),
    )
    agent = ResearchAgent(client, kit)
    outcome = agent.run(["A", "B", "C", "D", "E", "F"], as_of=date(2026, 6, 5))

    # tools were actually dispatched and results fed back to the model
    assert ("get_price_history", {"ticker": "A"}) in outcome.tool_calls
    assert ("get_recent_news", {"ticker": "B"}) in outcome.tool_calls
    assert client.tool_results_seen >= 1

    assert outcome.errors == []
    assert len(outcome.result.picks) == 5
    weights = picks_to_weights(outcome.result, n=5, scheme="tilt")
    assert is_valid(weights)
    # higher conviction -> >= weight
    assert weights["A"] >= weights["E"]


def test_validate_result_flags_bad_output():
    bad = parse_result({
        "picks": [
            {"ticker": "A", "conviction": 9, "horizon_weeks": 8,
             "thesis": "x", "key_risks": "y", "evidence": []},  # conviction OOB + no evidence
            {"ticker": "ZZZ", "conviction": 3, "horizon_weeks": 8,
             "thesis": "x", "key_risks": "y", "evidence": ["fact"]},  # not eligible
        ],
        "discarded": [],
    })
    errors = validate_result(bad, eligible={"A", "B", "C", "D", "E"}, min_picks=5)
    assert any("conviction" in e for e in errors)
    assert any("evidence" in e for e in errors)
    assert any("not in candidate" in e for e in errors)
    assert any(">= 5 picks" in e for e in errors)


def test_panel_provider_respects_lag():
    closes = [100.0 + i for i in range(40)]
    panel = {"A": {"dates": [f"d{i}" for i in range(40)], "opens": closes, "closes": closes}}
    provider = PanelDataProvider(
        panel,
        as_of_index=20,
        as_of=date(2026, 6, 5),
        earnings={"A": [
            {"date": date(2026, 5, 1), "surprise": 0.1, "reaction": 0.05},
            {"date": date(2026, 6, 30), "surprise": 0.9, "reaction": 0.4},  # future -> hidden
        ]},
    )
    # price stats computed only from closes up to index 20
    stats = provider.price_stats("A")
    assert stats["last_close"] == 100.0 + 20
    # future earnings event is not visible
    news = provider.news("A")
    assert all(n["date"] <= "2026-06-05" for n in news)
    assert len(news) == 1
