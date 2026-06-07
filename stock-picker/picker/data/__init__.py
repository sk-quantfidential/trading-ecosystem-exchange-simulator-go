"""Data layer: eligible universe + price panels.

``universe`` is stdlib (CSV). ``prices`` imports pandas/yfinance lazily — only the
live-fetch path needs them; CSV panel load/save is stdlib so backtests run offline.
"""
