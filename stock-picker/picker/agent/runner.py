"""Agent loop + LLM client abstraction.

``ResearchAgent`` runs a manual tool-use loop (research) then a structured-output
call (extract). It talks to an ``LLMClient`` so the orchestration can be tested
offline with ``ScriptedLLMClient`` and run live with ``AnthropicClient`` (which
lazily imports the SDK, so this module imports without ``anthropic`` installed).

Messages use a small normalized form both clients understand:
  {"role": "user", "kind": "text", "text": str}
  {"role": "assistant", "kind": "assistant", "text": str, "tool_calls": [ToolCall]}
  {"role": "user", "kind": "tool_results", "results": [{"id","content","is_error"}]}
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from datetime import date
from typing import Protocol, Sequence

from .prompts import (
    EXTRACT_INSTRUCTION,
    SYSTEM_EXTRACT,
    SYSTEM_RESEARCH,
    build_user_prompt,
)
from .schema import PICK_SCHEMA, ResearchResult, parse_result, validate_result
from .tools import TOOL_SCHEMAS, ToolKit


@dataclass
class ToolCall:
    id: str
    name: str
    input: dict


@dataclass
class LLMResponse:
    text: str = ""
    tool_calls: list[ToolCall] = field(default_factory=list)
    stop_reason: str = "end_turn"
    parsed: dict | None = None


# Normalized message constructors -------------------------------------------- #
def user_text(text: str) -> dict:
    return {"role": "user", "kind": "text", "text": text}


def assistant_turn(text: str, tool_calls: Sequence[ToolCall]) -> dict:
    return {"role": "assistant", "kind": "assistant", "text": text, "tool_calls": list(tool_calls)}


def tool_results_msg(results: Sequence[dict]) -> dict:
    return {"role": "user", "kind": "tool_results", "results": list(results)}


class LLMClient(Protocol):
    def complete(
        self,
        *,
        system: str,
        messages: list[dict],
        tools: list[dict] | None,
        output_schema: dict | None = None,
    ) -> LLMResponse: ...


# --------------------------------------------------------------------------- #
# Live client (lazy SDK import)
# --------------------------------------------------------------------------- #
class AnthropicClient:
    """Adapter to the Anthropic Python SDK. Verified offline via ScriptedLLMClient.

    Uses Opus 4.8 defaults from the API guidance: adaptive thinking, no sampling
    params, effort via output_config.
    """

    def __init__(
        self,
        model: str = "claude-opus-4-8",
        effort: str = "high",
        max_tokens: int = 12000,
    ) -> None:
        try:
            import anthropic  # noqa: WPS433 (lazy on purpose)
        except ImportError as exc:  # pragma: no cover - environment dependent
            raise RuntimeError("anthropic not installed. Run: pip install '.[llm]'") from exc
        self._client = anthropic.Anthropic()
        self.model = model
        self.effort = effort
        self.max_tokens = max_tokens

    def _to_sdk(self, messages: list[dict]) -> list[dict]:
        sdk: list[dict] = []
        for m in messages:
            if m["kind"] == "text":
                sdk.append({"role": m["role"], "content": [{"type": "text", "text": m["text"]}]})
            elif m["kind"] == "assistant":
                content: list[dict] = []
                if m.get("text"):
                    content.append({"type": "text", "text": m["text"]})
                for tc in m.get("tool_calls", []):
                    content.append(
                        {"type": "tool_use", "id": tc.id, "name": tc.name, "input": tc.input}
                    )
                sdk.append({"role": "assistant", "content": content})
            elif m["kind"] == "tool_results":
                sdk.append(
                    {
                        "role": "user",
                        "content": [
                            {
                                "type": "tool_result",
                                "tool_use_id": r["id"],
                                "content": r["content"],
                                "is_error": r.get("is_error", False),
                            }
                            for r in m["results"]
                        ],
                    }
                )
        return sdk

    def complete(
        self,
        *,
        system: str,
        messages: list[dict],
        tools: list[dict] | None,
        output_schema: dict | None = None,
    ) -> LLMResponse:
        output_config: dict = {"effort": self.effort}
        if output_schema is not None:
            output_config["format"] = {"type": "json_schema", "schema": output_schema}

        params: dict = {
            "model": self.model,
            "max_tokens": self.max_tokens,
            "system": system,
            "messages": self._to_sdk(messages),
            "thinking": {"type": "adaptive"},
            "output_config": output_config,
        }
        if tools:
            params["tools"] = tools

        resp = self._client.messages.create(**params)

        text_parts: list[str] = []
        tool_calls: list[ToolCall] = []
        for block in resp.content:
            btype = getattr(block, "type", None)
            if btype == "text":
                text_parts.append(block.text)
            elif btype == "tool_use":
                tool_calls.append(ToolCall(id=block.id, name=block.name, input=dict(block.input)))
        text = "".join(text_parts)
        parsed = None
        if output_schema is not None and text:
            parsed = json.loads(text)
        return LLMResponse(
            text=text, tool_calls=tool_calls, stop_reason=resp.stop_reason, parsed=parsed
        )


# --------------------------------------------------------------------------- #
# Scripted client for offline tests/dev
# --------------------------------------------------------------------------- #
@dataclass
class ScriptedLLMClient:
    """Follows a fixed script: rounds of tool calls, then a final JSON result.

    ``rounds`` is a list of rounds; each round is a list of (tool_name, input)
    the "model" requests. ``final`` is the structured result returned on the
    output_schema call. Records what it saw for assertions.
    """

    rounds: list[list[tuple[str, dict]]]
    final: dict
    tool_results_seen: int = 0
    _round: int = 0

    def complete(
        self,
        *,
        system: str,
        messages: list[dict],
        tools: list[dict] | None,
        output_schema: dict | None = None,
    ) -> LLMResponse:
        self.tool_results_seen = sum(1 for m in messages if m.get("kind") == "tool_results")
        if output_schema is not None:
            return LLMResponse(
                text=json.dumps(self.final), stop_reason="end_turn", parsed=self.final
            )
        if self._round < len(self.rounds):
            rnd = self.rounds[self._round]
            self._round += 1
            calls = [
                ToolCall(id=f"toolu_{self._round}_{j}", name=name, input=inp)
                for j, (name, inp) in enumerate(rnd)
            ]
            return LLMResponse(text="(researching)", tool_calls=calls, stop_reason="tool_use")
        return LLMResponse(text="research complete", stop_reason="end_turn")


# --------------------------------------------------------------------------- #
# The agent
# --------------------------------------------------------------------------- #
@dataclass
class ResearchOutcome:
    result: ResearchResult
    errors: list[str]
    transcript: list[dict]
    tool_calls: list[tuple[str, dict]]


class ResearchAgent:
    def __init__(self, client: LLMClient, toolkit: ToolKit, max_rounds: int = 8) -> None:
        self.client = client
        self.toolkit = toolkit
        self.max_rounds = max_rounds

    def run(
        self,
        candidates: Sequence[str],
        as_of: date,
        target_picks: int = 8,
        min_picks: int = 5,
    ) -> ResearchOutcome:
        messages: list[dict] = [user_text(build_user_prompt(candidates, as_of, target_picks))]

        # 1. research loop
        for _ in range(self.max_rounds):
            resp = self.client.complete(
                system=SYSTEM_RESEARCH, messages=messages, tools=TOOL_SCHEMAS, output_schema=None
            )
            messages.append(assistant_turn(resp.text, resp.tool_calls))
            if resp.stop_reason == "tool_use" and resp.tool_calls:
                results = []
                for tc in resp.tool_calls:
                    try:
                        content = self.toolkit.dispatch(tc.name, tc.input)
                        is_error = False
                    except Exception as exc:  # noqa: BLE001 - surface tool errors to the model
                        content = json.dumps({"error": str(exc)})
                        is_error = True
                    results.append({"id": tc.id, "content": content, "is_error": is_error})
                messages.append(tool_results_msg(results))
            else:
                break

        # 2. structured extraction
        messages.append(user_text(EXTRACT_INSTRUCTION))
        resp = self.client.complete(
            system=SYSTEM_EXTRACT, messages=messages, tools=None, output_schema=PICK_SCHEMA
        )
        result = parse_result(resp.parsed or {"picks": [], "discarded": []})
        errors = validate_result(result, eligible=set(candidates), min_picks=min_picks)
        return ResearchOutcome(
            result=result,
            errors=errors,
            transcript=messages,
            tool_calls=list(self.toolkit.call_log),
        )


def default_live_client() -> AnthropicClient:
    """Construct the live client; raises a clear error if unusable."""
    if not os.environ.get("ANTHROPIC_API_KEY"):
        raise RuntimeError("ANTHROPIC_API_KEY is not set")
    return AnthropicClient()
