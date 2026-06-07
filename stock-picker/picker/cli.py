"""Command-line entry points (stdlib paths work with no extra deps).

  picker score    --panel prices.csv --weights weights.json
  picker backtest --panel prices.csv [--meta universe.csv] [--thesis t.json]
  picker select   --records records.json --as-of 2026-06-05 [--thesis t.json]
  picker fetch    --tickers AZN.L,SHEL.L --from 2025-01-01 --to 2026-06-01 --out prices.csv
"""

from __future__ import annotations

import argparse
import json
from datetime import date

from . import scoring
from .backtest import backtest_constant_mix, summarize
from .baseline_pick import baseline_pick, records_from_panel
from .data import prices as P
from .data import universe as U
from .selector import select_candidates
from .signals.thesis import Thesis


def _load_aligned_panel(path: str) -> P.Panel:
    _, panel = P.align_panel(P.load_panel_from_csv(path))
    return panel


def _meta_and_thesis(args) -> tuple[dict, Thesis | None]:
    meta: dict = {}
    if getattr(args, "meta", None):
        meta = U.meta_by_ticker(U.load_universe(args.meta))
    thesis = Thesis.from_json(args.thesis) if getattr(args, "thesis", None) else None
    return meta, thesis


def cmd_score(args) -> None:
    panel = _load_aligned_panel(args.panel)
    weights = json.loads(open(args.weights, encoding="utf-8").read())
    returns_by_ticker = {
        t: scoring.daily_returns_from_prices(
            panel[t]["opens"], panel[t]["closes"], entry_index=0
        )
        for t in weights
    }
    season = scoring.score_constant_mix(weights, returns_by_ticker)
    print(f"constant-mix season return: {season:+.2%}")


def cmd_backtest(args) -> None:
    panel = _load_aligned_panel(args.panel)
    meta, thesis = _meta_and_thesis(args)
    first = next(iter(panel))

    def weight_fn(pnl: P.Panel, t: int) -> dict[str, float]:
        records = records_from_panel(pnl, t, meta_by_ticker=meta)
        as_of = date.fromisoformat(pnl[first]["dates"][t])
        weights, _ = baseline_pick(records, as_of, thesis=thesis, n=args.n, scheme=args.scheme)
        return weights

    results = backtest_constant_mix(
        panel, weight_fn, window_days=args.window, step_days=args.step,
        min_history=args.min_history,
    )
    print(json.dumps(summarize(results), indent=2))


def _parse_record_dates(records: list[dict]) -> None:
    for r in records:
        for ev in r.get("earnings", []) + r.get("events", []):
            if isinstance(ev.get("date"), str):
                ev["date"] = date.fromisoformat(ev["date"])


def cmd_select(args) -> None:
    records = json.loads(open(args.records, encoding="utf-8").read())
    _parse_record_dates(records)
    thesis = Thesis.from_json(args.thesis) if args.thesis else None
    candidates = select_candidates(
        records, date.fromisoformat(args.as_of), thesis=thesis, top_n=args.top_n
    )
    for c in candidates:
        print(c.rationale)


def cmd_research(args) -> None:
    """Run the Phase-2 research agent over a panel (needs ANTHROPIC_API_KEY + .[llm])."""
    from .agent import PanelDataProvider, ResearchAgent, ToolKit, picks_to_weights
    from .agent.runner import default_live_client

    panel = _load_aligned_panel(args.panel)
    first = next(iter(panel))
    as_of_index = args.as_of_index if args.as_of_index is not None else len(panel[first]["dates"]) - 1
    as_of = date.fromisoformat(panel[first]["dates"][as_of_index])

    provider = PanelDataProvider(panel, as_of_index=as_of_index, as_of=as_of)
    agent = ResearchAgent(default_live_client(), ToolKit(provider))
    outcome = agent.run(provider.candidates(), as_of=as_of, target_picks=args.target_picks)

    for p in outcome.result.picks:
        print(f"[{p.conviction}/5] {p.ticker} (h={p.horizon_weeks}w): {p.thesis}")
        print(f"      risks: {p.key_risks}")
        print(f"      evidence: {p.evidence}")
    if outcome.errors:
        print("\nVALIDATION ERRORS (model output is not rules-valid):")
        for e in outcome.errors:
            print(f"  - {e}")
    else:
        weights = picks_to_weights(outcome.result, n=args.n, scheme=args.scheme)
        print(f"\nrules-valid portfolio: {json.dumps(weights)}")


def cmd_fetch(args) -> None:
    tickers = [t.strip() for t in args.tickers.split(",") if t.strip()]
    panel = P.fetch_prices(tickers, args.start, args.end)
    P.save_panel_to_csv(panel, args.out)
    print(f"wrote {sum(len(s['dates']) for s in panel.values())} rows to {args.out}")


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="picker")
    sub = parser.add_subparsers(dest="cmd", required=True)

    s = sub.add_parser("score", help="constant-mix season return for a weight set")
    s.add_argument("--panel", required=True)
    s.add_argument("--weights", required=True)
    s.set_defaults(func=cmd_score)

    b = sub.add_parser("backtest", help="rolling 8-week backtest of the baseline picker")
    b.add_argument("--panel", required=True)
    b.add_argument("--meta")
    b.add_argument("--thesis")
    b.add_argument("--n", type=int, default=5)
    b.add_argument("--scheme", default="equal", choices=["equal", "tilt"])
    b.add_argument("--window", type=int, default=40)
    b.add_argument("--step", type=int, default=5)
    b.add_argument("--min-history", dest="min_history", type=int, default=60)
    b.set_defaults(func=cmd_backtest)

    sl = sub.add_parser("select", help="rank candidates from a records JSON")
    sl.add_argument("--records", required=True)
    sl.add_argument("--as-of", dest="as_of", required=True)
    sl.add_argument("--thesis")
    sl.add_argument("--top-n", dest="top_n", type=int, default=20)
    sl.set_defaults(func=cmd_select)

    r = sub.add_parser("research", help="Phase-2 agent: research a panel -> picks (needs .[llm] + key)")
    r.add_argument("--panel", required=True)
    r.add_argument("--as-of-index", dest="as_of_index", type=int, default=None)
    r.add_argument("--target-picks", dest="target_picks", type=int, default=8)
    r.add_argument("--n", type=int, default=5)
    r.add_argument("--scheme", default="tilt", choices=["equal", "tilt"])
    r.set_defaults(func=cmd_research)

    f = sub.add_parser("fetch", help="fetch open/close via yfinance (needs .[data])")
    f.add_argument("--tickers", required=True)
    f.add_argument("--from", dest="start", required=True)
    f.add_argument("--to", dest="end", required=True)
    f.add_argument("--out", required=True)
    f.set_defaults(func=cmd_fetch)

    args = parser.parse_args(argv)
    args.func(args)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
