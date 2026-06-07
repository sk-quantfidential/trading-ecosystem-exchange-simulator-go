"""Tool schemas + dispatch for the research agent.

Four read-only, grounded tools. Each returns a JSON string sourced from the
``DataProvider`` — the model gets facts to reason over, never a free-text channel
to invent numbers. A missing data point comes back as an explicit ``"unknown"`` /
``error`` so the model can lower conviction rather than hallucinate.
"""

from __future__ import annotations

import json
from typing import Callable

from .provider import DataProvider

# Tool definitions in the Anthropic Messages API shape.
TOOL_SCHEMAS: list[dict] = [
    {
        "name": "get_fundamentals",
        "description": (
            "Fundamentals for a ticker (P/E, ROE, debt/equity, revenue growth, "
            "gross margin, market cap, currency). Call before asserting any "
            "fundamental figure — do not rely on prior knowledge."
        ),
        "input_schema": {
            "type": "object",
            "properties": {"ticker": {"type": "string"}},
            "required": ["ticker"],
        },
    },
    {
        "name": "get_price_history",
        "description": (
            "Price stats as of the decision date: last close, 5/20/60-day returns, "
            "20-day volatility, drawdown from high. Use for momentum/oversold reads."
        ),
        "input_schema": {
            "type": "object",
            "properties": {"ticker": {"type": "string"}},
            "required": ["ticker"],
        },
    },
    {
        "name": "get_factor_scores",
        "description": "Quant factor/momentum scores for a ticker (point-in-time).",
        "input_schema": {
            "type": "object",
            "properties": {"ticker": {"type": "string"}},
            "required": ["ticker"],
        },
    },
    {
        "name": "get_recent_news",
        "description": (
            "Recent news/catalysts and earnings events on or before the decision "
            "date (earnings surprises, guidance, buybacks, M&A, etc.)."
        ),
        "input_schema": {
            "type": "object",
            "properties": {"ticker": {"type": "string"}},
            "required": ["ticker"],
        },
    },
]


def _wrap(payload: object) -> str:
    return json.dumps(payload, default=str)


class ToolKit:
    """Binds the tool names to a provider and dispatches calls to JSON strings."""

    def __init__(self, provider: DataProvider) -> None:
        self.provider = provider
        self._handlers: dict[str, Callable[[dict], str]] = {
            "get_fundamentals": self._fundamentals,
            "get_price_history": self._price_history,
            "get_factor_scores": self._factor_scores,
            "get_recent_news": self._news,
        }
        self.call_log: list[tuple[str, dict]] = []

    def dispatch(self, name: str, tool_input: dict) -> str:
        self.call_log.append((name, dict(tool_input)))
        handler = self._handlers.get(name)
        if handler is None:
            return _wrap({"error": f"unknown tool {name!r}"})
        return handler(tool_input)

    # --- handlers ---------------------------------------------------------- #
    def _fundamentals(self, args: dict) -> str:
        data = self.provider.fundamentals(args["ticker"])
        return _wrap(data if data is not None else {"ticker": args["ticker"], "fundamentals": "unknown"})

    def _price_history(self, args: dict) -> str:
        data = self.provider.price_stats(args["ticker"])
        return _wrap(data if data is not None else {"ticker": args["ticker"], "price_stats": "unknown"})

    def _factor_scores(self, args: dict) -> str:
        data = self.provider.factor_scores(args["ticker"])
        return _wrap(data if data is not None else {"ticker": args["ticker"], "factor_scores": "unknown"})

    def _news(self, args: dict) -> str:
        return _wrap({"ticker": args["ticker"], "news": self.provider.news(args["ticker"])})
