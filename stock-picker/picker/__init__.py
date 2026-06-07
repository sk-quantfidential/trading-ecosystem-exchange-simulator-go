"""FT Stock Picking Game — agentic stock picker.

Core modules (``scoring``, ``portfolio``, ``signals``, ``selector``) are
stdlib-only so they import and test without pandas/yfinance installed.
The data/factor/backtest layers import pandas/yfinance lazily — only when used.
"""

__all__ = ["scoring", "portfolio", "selector", "signals"]
