"""Structured-output schema for picks + parsing/validation helpers.

Structured outputs guarantee the model returns parseable JSON in this shape.
Note: the JSON-schema subset for structured outputs does NOT support numeric
min/max, so ``conviction`` bounds (1-5) are enforced in ``validate_result``.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Mapping, Sequence

from ..portfolio import build_weights

# JSON schema handed to the API via output_config.format.
PICK_SCHEMA: dict = {
    "type": "object",
    "additionalProperties": False,
    "properties": {
        "picks": {
            "type": "array",
            "items": {
                "type": "object",
                "additionalProperties": False,
                "properties": {
                    "ticker": {"type": "string"},
                    "conviction": {"type": "integer"},          # 1-5 (checked in code)
                    "horizon_weeks": {"type": "integer"},
                    "thesis": {"type": "string"},
                    "key_risks": {"type": "string"},
                    "evidence": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "Tool-sourced facts supporting the thesis.",
                    },
                },
                "required": [
                    "ticker", "conviction", "horizon_weeks",
                    "thesis", "key_risks", "evidence",
                ],
            },
        },
        "discarded": {
            "type": "array",
            "items": {
                "type": "object",
                "additionalProperties": False,
                "properties": {
                    "ticker": {"type": "string"},
                    "reason": {"type": "string"},
                },
                "required": ["ticker", "reason"],
            },
        },
    },
    "required": ["picks", "discarded"],
}

CONVICTION_MIN = 1
CONVICTION_MAX = 5


@dataclass
class Pick:
    ticker: str
    conviction: int
    horizon_weeks: int
    thesis: str
    key_risks: str
    evidence: list[str] = field(default_factory=list)


@dataclass
class ResearchResult:
    picks: list[Pick]
    discarded: list[dict]


def parse_result(data: Mapping) -> ResearchResult:
    picks = [
        Pick(
            ticker=p["ticker"],
            conviction=int(p["conviction"]),
            horizon_weeks=int(p["horizon_weeks"]),
            thesis=p["thesis"],
            key_risks=p["key_risks"],
            evidence=list(p.get("evidence", [])),
        )
        for p in data.get("picks", [])
    ]
    return ResearchResult(picks=picks, discarded=list(data.get("discarded", [])))


def validate_result(
    result: ResearchResult,
    eligible: Sequence[str] | None = None,
    min_picks: int = 5,
) -> list[str]:
    """Return a list of problems with the model's output (empty == clean)."""
    errors: list[str] = []
    if len(result.picks) < min_picks:
        errors.append(f"need >= {min_picks} picks, got {len(result.picks)}")

    seen: set[str] = set()
    elig = set(eligible) if eligible is not None else None
    for p in result.picks:
        if not (CONVICTION_MIN <= p.conviction <= CONVICTION_MAX):
            errors.append(f"{p.ticker}: conviction {p.conviction} outside [1, 5]")
        if p.ticker in seen:
            errors.append(f"{p.ticker}: duplicate pick")
        seen.add(p.ticker)
        if elig is not None and p.ticker not in elig:
            errors.append(f"{p.ticker}: not in candidate/eligible set")
        if not p.evidence:
            errors.append(f"{p.ticker}: no tool-sourced evidence cited")
    return errors


def picks_to_weights(
    result: ResearchResult,
    n: int = 5,
    scheme: str = "tilt",
) -> dict[str, float]:
    """Convert the top picks (by conviction) into a rules-valid weight set.

    Conviction is the ranking score. Ties keep the model's order. The result is
    validated by ``portfolio.build_weights`` (>=5 names, 5-25% box, sum 1.0).
    """
    if len(result.picks) < n:
        raise ValueError(f"need >= {n} picks, got {len(result.picks)}")
    ranked = [(p.ticker, float(p.conviction)) for p in result.picks]
    return build_weights(ranked, n=n, scheme=scheme)
