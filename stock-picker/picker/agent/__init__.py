"""Phase 2: the research agent.

A single Claude agent that researches candidate stocks through grounded tools and
returns structured picks. Designed to run offline during development:

  * ``LLMClient`` is an interface — ``ScriptedLLMClient`` (tests/dev) vs
    ``AnthropicClient`` (live; lazily imports the SDK so this package imports
    without ``anthropic`` installed).
  * Tools sit behind a ``DataProvider`` — synthetic/panel-backed now, live later.

The cardinal rule (enforced by the system prompt and the tool design): the model
states no number it didn't get from a tool.
"""

from .provider import DataProvider, DictDataProvider, PanelDataProvider
from .runner import (
    AnthropicClient,
    LLMClient,
    LLMResponse,
    ResearchAgent,
    ResearchOutcome,
    ScriptedLLMClient,
    ToolCall,
)
from .schema import (
    PICK_SCHEMA,
    Pick,
    ResearchResult,
    parse_result,
    picks_to_weights,
    validate_result,
)
from .tools import TOOL_SCHEMAS, ToolKit

__all__ = [
    "DataProvider",
    "DictDataProvider",
    "PanelDataProvider",
    "LLMClient",
    "LLMResponse",
    "ToolCall",
    "AnthropicClient",
    "ScriptedLLMClient",
    "ResearchAgent",
    "ResearchOutcome",
    "ToolKit",
    "TOOL_SCHEMAS",
    "PICK_SCHEMA",
    "Pick",
    "ResearchResult",
    "parse_result",
    "validate_result",
    "picks_to_weights",
]
