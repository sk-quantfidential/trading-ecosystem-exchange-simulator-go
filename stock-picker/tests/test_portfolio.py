"""Portfolio constraint validation and weight construction."""

from picker.portfolio import (
    MAX_WEIGHT,
    MIN_WEIGHT,
    build_weights,
    is_valid,
    validate_weights,
)


def test_equal_weight_five_names_is_valid():
    w = {"A": 0.2, "B": 0.2, "C": 0.2, "D": 0.2, "E": 0.2}
    assert is_valid(w)


def test_too_few_holdings_is_invalid():
    assert validate_weights({"A": 0.5, "B": 0.5})


def test_weight_outside_box_is_invalid():
    w = {"A": 0.30, "B": 0.20, "C": 0.20, "D": 0.20, "E": 0.10}  # A > 25%
    assert any("outside" in e for e in validate_weights(w))


def test_weights_must_sum_to_one():
    w = {"A": 0.2, "B": 0.2, "C": 0.2, "D": 0.2, "E": 0.1}  # 0.9
    assert any("sum" in e for e in validate_weights(w))


def test_eligible_universe_enforced():
    w = {"A": 0.2, "B": 0.2, "C": 0.2, "D": 0.2, "E": 0.2}
    errs = validate_weights(w, eligible={"A", "B", "C", "D"})
    assert any("eligible" in e for e in errs)


def test_build_weights_equal_default():
    ranked = [("A", 3.0), ("B", 2.0), ("C", 1.0), ("D", 0.5), ("E", 0.1), ("F", -1.0)]
    w = build_weights(ranked, n=5, scheme="equal")
    assert len(w) == 5
    assert is_valid(w)
    assert all(abs(v - 0.2) < 1e-9 for v in w.values())


def test_build_weights_tilt_is_rules_valid():
    ranked = [("A", 5.0), ("B", 2.0), ("C", 1.0), ("D", 0.0), ("E", -1.0), ("F", -3.0)]
    w = build_weights(ranked, n=6, scheme="tilt")
    assert is_valid(w), validate_weights(w)
    assert abs(sum(w.values()) - 1.0) < 1e-6
    assert all(MIN_WEIGHT - 1e-9 <= v <= MAX_WEIGHT + 1e-9 for v in w.values())
    # Higher-scored names get >= weight than lower-scored ones.
    assert w["A"] >= w["F"]


def test_infeasible_holding_count_raises():
    ranked = [(f"T{i}", float(i)) for i in range(30)]
    raised = False
    try:
        build_weights(ranked, n=21, scheme="equal")
    except ValueError:
        raised = True
    assert raised
