# FT Stock Picker — Phase 0 + Phase 1 scaffold

A buildable starting point for the agentic stock picker described in
[`../docs/agentic-stock-picker-study-plan.md`](../docs/agentic-stock-picker-study-plan.md).
This covers **Phase 0** (constant-mix scoring harness) and **Phase 1** (deterministic
signal-based baseline + rolling backtest), plus the **stock-selection engine**
(momentum / earnings / catalyst signals + a macro-thesis filter).

> Not investment advice. The FT game's own rules state that portfolio decisions in
> the game are not investment recommendations. A thesis here is a *hypothesis to test*.

## Why it's split the way it is

The **core** (`scoring`, `portfolio`, `signals`, `selector`, `baseline_pick`,
`backtest`) is **stdlib-only** — it imports and tests with nothing installed. The
**data layer** (`data/prices.py` live fetch, `factors.py`) imports `pandas`/`yfinance`
**lazily**, so live data is opt-in and never blocks the rest.

## Quick start

```bash
cd stock-picker
python3 tests/run.py            # 21 tests, no dependencies required
```

Wire up live data when ready:

```bash
python3 -m venv .venv && source .venv/bin/activate
pip install -e '.[data,llm,dev]'
pytest                          # same tests via pytest

# Fetch open+close for a watchlist, then backtest the baseline over 8-week windows
picker fetch  --tickers AZN.L,SHEL.L,KO,JNJ,XOM,NG.L --from 2024-06-01 --to 2026-06-01 --out prices.csv
picker backtest --panel prices.csv --meta data/universe.sample.csv \
                --thesis theses/2026-06-example.json --n 5 --scheme tilt --window 40
```

## The two mechanics that matter

1. **Constant-mix scoring** (`scoring.py`). The game resets your % weights to target
   every day at the close, so `portfolio_daily_return = Σ(weightᵢ × daily_returnᵢ)`
   and the season return compounds those. A 25% holding up 40% adds **10%** that day,
   then snaps back to 25% — it does **not** drift. (Pinned by `test_scoring.py`.)
2. **Next-day-open execution lag** (`backtest.py`, `baseline_pick.records_from_panel`).
   You decide on the close of day *T* and fill at the **open of day T+1**. Signals only
   ever see data up to *T*; an overnight gap between *T*'s close and *T+1*'s open is
   **not** earned. (Pinned by `test_backtest.py`.)

## Layout

```
picker/
  scoring.py          # constant-mix, daily-rebalanced season return
  portfolio.py        # >=5 names, 5-25% box, sum=1, long-only; weight construction
  signals/
    momentum.py       # short/medium momentum + severe-move z-score
    earnings.py       # post-earnings-announcement drift (PEAD)
    catalyst.py       # event catalysts (guidance, buyback, M&A, ...) w/ recency decay
    thesis.py         # macro view as tunable sector/tag tilts (loaded from JSON)
  selector.py         # standardise + combine signals + apply thesis -> ranked candidates
  baseline_pick.py    # Phase 1: candidates -> rules-valid portfolio
  backtest.py         # rolling 8-week windows, strict T+1-open fills
  factors.py          # optional pandas cross-sectional factors
  agent/              # Phase 2: Claude tool-use research agent
    schema.py         #   structured-output schema + parse/validate + picks->weights
    provider.py       #   DataProvider (DictDataProvider / PanelDataProvider)
    tools.py          #   4 grounded tools + dispatch (facts in, no free-text invention)
    prompts.py        #   system/user prompts (grounding + 8-week + lag rules)
    runner.py         #   LLMClient (AnthropicClient | ScriptedLLMClient) + ResearchAgent
  data/
    universe.py       # eligible-stock list (CSV)
    prices.py         # yfinance fetch + sqlite cache + stdlib CSV panel I/O
  cli.py              # picker score | backtest | select | fetch
theses/2026-06-example.json   # example macro thesis (UNVERIFIED — test before use)
data/universe.sample.csv      # illustrative multi-exchange universe with tags
tests/                        # 21 stdlib tests + a zero-dependency runner
```

## The stock-selection process

`selector.select_candidates` runs the three signals over your universe, **cross-
sectionally standardises** each, combines them (configurable weights), then applies
the **thesis** as an additive tilt in z-space (`ln` of its multiplier; `0` excludes a
name). The top candidates flow into `baseline_pick` → a rules-valid portfolio.

Over an **8-week** horizon, short-term **momentum, earnings drift, and catalysts**
dominate — that's why those are the three signals, not slow value/quality factors.

## Turning a macro view into the thesis (your 2026-06 stance)

Edit `theses/2026-06-example.json`. It encodes "favour AI-immune defensives, fade
high-multiple AI-capex, tilt to oil beta" as `sector_tilts` / `tag_tilts`. Assign the
tags (`ai_immune`, `pricing_power`, `oil_beta`, `high_multiple_ai_capex`, ...) to names
in your universe CSV. **Then backtest the thesis vs the no-thesis baseline** on recent
rolling windows before trusting it.

⚠️ Watch the internal tension: "high oil" rewards `oil_beta`, but "as these problems
resolve" implies oil *falls* (rewarding oil *consumers* — transport, chemicals,
airlines). Pick one coherent direction and set the tilts to match; don't bet on both.

## Phase 2: the research agent (`picker/agent/`)

A single Claude agent (Opus 4.8, adaptive thinking) researches each candidate via
four grounded tools, then returns **structured picks** (conviction 1-5, thesis,
key risks, and tool-sourced `evidence`). `picks_to_weights` turns them into a
rules-valid portfolio.

Two design choices make it usable **before** you have data or API keys:
- **`LLMClient` is an interface.** `ScriptedLLMClient` drives the full loop offline
  in tests; `AnthropicClient` runs it live (lazy SDK import — the package imports
  fine without `anthropic`). The loop logic is therefore tested with no network.
- **Tools read a `DataProvider`.** The model only ever sees tool output, never a
  channel to invent numbers; missing data returns an explicit `"unknown"` so the
  model lowers conviction instead of hallucinating. Providers also enforce the
  next-day-open lag (news/prices pre-filtered to the decision date).

Run it live once you have data + a key:

```bash
pip install -e '.[data,llm]'
export ANTHROPIC_API_KEY=...
picker research --panel prices.csv --n 5 --scheme tilt   # research -> validated picks -> weights
```

## Next steps (from the study plan)

- Phase 3: dossiers (filings/news) + `web_search`/`web_fetch` server tools + prompt
  caching, feeding richer context into the same agent.
- Phase 5: walk-forward evaluation + an ablation scorecard — does the agent beat the
  deterministic baseline, net of API cost? (Wire the agent as a `weight_fn` into
  `backtest.py` using a `PanelDataProvider` at each decision date.)
- Phase 6: scheduled rebalancing loop + journal/reflection.
