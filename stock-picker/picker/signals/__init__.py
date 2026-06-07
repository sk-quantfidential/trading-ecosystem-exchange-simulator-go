"""Signal-selection engine: turn raw data into ranked candidate stocks.

Three bottom-up signals — momentum, earnings anomaly (PEAD), and event catalysts —
plus a top-down ``Thesis`` filter that applies a macro view as tunable sector/tag
tilts. All stdlib; all respect the operational lag: a signal is computed *as of*
the decision-day close, and the resulting trade fills at the NEXT day's open.
"""

from .base import SignalScore, standardize
from .momentum import momentum_signal
from .earnings import earnings_signal
from .catalyst import catalyst_signal
from .thesis import Thesis

__all__ = [
    "SignalScore",
    "standardize",
    "momentum_signal",
    "earnings_signal",
    "catalyst_signal",
    "Thesis",
]
