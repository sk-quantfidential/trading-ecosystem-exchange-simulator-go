"""Zero-dependency test runner (works without pytest).

    python3 tests/run.py        # from the stock-picker/ directory
    python3 -m tests.run

Also pytest-compatible: `pip install '.[dev]' && pytest` runs the same files.
"""

from __future__ import annotations

import importlib
import sys
import traceback
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

TEST_MODULES = [
    "tests.test_scoring",
    "tests.test_portfolio",
    "tests.test_signals",
    "tests.test_backtest",
    "tests.test_agent",
]


def main() -> int:
    passed = 0
    failures: list[tuple[str, str]] = []
    for modname in TEST_MODULES:
        mod = importlib.import_module(modname)
        for name in sorted(vars(mod)):
            if not name.startswith("test_"):
                continue
            fn = getattr(mod, name)
            if not callable(fn):
                continue
            label = f"{modname}.{name}"
            try:
                fn()
                passed += 1
                print(f"PASS {label}")
            except Exception as exc:  # noqa: BLE001 - test runner
                failures.append((label, traceback.format_exc()))
                print(f"FAIL {label}: {exc}")

    print(f"\n{passed} passed, {len(failures)} failed")
    for label, tb in failures:
        print(f"\n=== {label} ===\n{tb}")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
