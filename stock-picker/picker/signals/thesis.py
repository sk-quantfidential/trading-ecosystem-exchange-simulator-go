"""Macro-thesis filter: a top-down view expressed as tunable tilts.

A ``Thesis`` is loaded from JSON and turns a macro stance ("favour AI-immune
defensives, fade high-multiple AI-capex names, tilt to oil beta") into a single
multiplier applied to each candidate's bottom-up signal score:

    final_score = combined_signal_score * thesis_multiplier(stock_metadata)

A multiplier of 0.0 excludes the name entirely. This keeps *opinions* in version-
controlled config (testable, swappable) and out of the code — and lets you
backtest a thesis on the constant-mix harness before trusting it.

This is a hypothesis to test, never investment advice.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from pathlib import Path


@dataclass
class Thesis:
    name: str
    notes: str = ""
    sector_tilts: dict[str, float] = field(default_factory=dict)
    tag_tilts: dict[str, float] = field(default_factory=dict)
    exclude_sectors: list[str] = field(default_factory=list)
    include_only_sectors: list[str] | None = None

    @classmethod
    def from_json(cls, path: str | Path) -> "Thesis":
        data = json.loads(Path(path).read_text())
        return cls(
            name=data["name"],
            notes=data.get("notes", ""),
            sector_tilts=data.get("sector_tilts", {}),
            tag_tilts=data.get("tag_tilts", {}),
            exclude_sectors=data.get("exclude_sectors", []),
            include_only_sectors=data.get("include_only_sectors"),
        )

    def multiplier(self, meta: dict) -> tuple[float, str]:
        """Return (multiplier, rationale) for a stock's metadata.

        ``meta`` is ``{"sector": str, "tags": [str, ...]}``. Returns 0.0 to drop
        the name (excluded sector, or not in an include-only allowlist).
        """
        sector = meta.get("sector")
        tags = list(meta.get("tags", []))

        if self.include_only_sectors and sector not in self.include_only_sectors:
            return 0.0, f"excluded: sector {sector!r} not in include_only list"
        if sector in self.exclude_sectors:
            return 0.0, f"excluded: sector {sector!r} on exclude list"

        m = 1.0
        parts: list[str] = []
        if sector in self.sector_tilts:
            m *= self.sector_tilts[sector]
            parts.append(f"sector {sector} x{self.sector_tilts[sector]:.2f}")
        for tag in tags:
            if tag in self.tag_tilts:
                m *= self.tag_tilts[tag]
                parts.append(f"tag {tag} x{self.tag_tilts[tag]:.2f}")

        rationale = "; ".join(parts) if parts else "no thesis tilt (x1.00)"
        return m, rationale
