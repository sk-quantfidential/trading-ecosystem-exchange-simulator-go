# Agentic Stock Picker — Hands-On Study Plan (FT Stock-Picking Competition)

> A practical, build-as-you-learn program. Every phase ships a **working, competition-submittable artifact** and teaches the concepts behind it. You always have something that produces picks; each phase makes those picks smarter.

---

## 0. How to read this plan

- **Each phase = one shippable deliverable.** You could enter the competition after Phase 1. Later phases improve the picks; they are not prerequisites for *having* an entry.
- **Effort model:** sized for ~6–10 hrs/week. The "Day" / "Week" labels are relative effort, not a calendar. Compress or stretch to taste.
- **Default stack:** **Python** for the picker brain (richest data/backtest/LLM ecosystem), with an optional path to integrate into your existing **Go** trading ecosystem later (see Phase 6). Rationale in §2.
- **Definition of done (DoD)** is stated per phase — don't move on until it's green.

### ⚠️ Phase 0, Step 0 — pin down the actual competition rules first

Newspaper stock-picking competitions vary a lot, and the rules drive your whole design. Before writing code, get **written answers** to:

| Question | Why it changes the design |
|---|---|
| **Universe** — which exchange/index? (e.g. FTSE 350, FTSE All-Share, global, US-allowed?) | Defines your data sources and ticker symbology. |
| **How many picks?** (a single stock? a basket of N?) | Single-pick → high-conviction; basket → portfolio construction matters. |
| **Holding period** — fixed (a year/quarter) or rebalanced? | One-shot vs. ongoing rebalancing changes the whole agent loop. |
| **Scoring** — total return? relative to benchmark? risk-adjusted? | You must replicate the scoring function exactly in your backtest. |
| **Constraints** — long-only? min market cap? no investment trusts/ETFs? | Encode as hard filters on the universe. |
| **Entry mechanics & deadline** — how/when do you submit? | Drives the automation in Phase 6. |

Write these into `docs/competition-rules.md` and treat that file as the spec the whole system targets. **The rest of this plan assumes the common format: pick a basket of N long-only equities from a defined index, held for a fixed period, ranked by total return.** Adjust where your rules differ.

> Reality check: this is a single-shot game with huge variance. A great process can lose to a lucky dart throw over one period. Optimise for a *defensible, repeatable edge* and for *learning*, not for guaranteeing first place.

---

## 1. The system you are building

An **agentic stock picker** is a loop where an LLM (Claude) does the *reasoning and judgement*, while deterministic tools supply *ground-truth data* and *math*. The LLM never invents numbers; it calls tools to get them.

```
            ┌────────────────────────────────────────────────────────┐
            │                     PICKER PIPELINE                      │
            ├────────────────────────────────────────────────────────┤
 Universe ─▶│ 1. Screen      deterministic factor filter → candidates │
            │ 2. Research    agent + tools build a dossier per name   │
            │ 3. Debate      bull vs bear vs risk (multi-agent)       │
            │ 4. Construct   PM agent sizes a portfolio under rules    │
            │ 5. Backtest    evaluate the process out-of-sample        │
            │ 6. Submit      format entry + log + reflect for next run │
            └────────────────────────────────────────────────────────┘
                     ▲ tools: prices, fundamentals, filings, news, factor math
```

You will build these in roughly the order **1 → 5 → 2 → 3 → 4 → 6** (baseline + evaluation first, intelligence second, automation last), because a baseline and an honest scoreboard are what let you tell whether the "agentic" part is actually adding value.

---

## 2. Tooling & accounts (set up once)

**Language:** Python 3.11+. The LLM agent, the data wrangling (pandas), and backtesting libraries all live here. You *can* do it in Go (the Anthropic Go SDK has a `BetaToolRunner` for tool-use loops, and it would slot into your existing ecosystem), but you'd be fighting a much thinner data/quant ecosystem. **Recommendation: prototype in Python; only port to Go if/when you want to run the picker as a service inside the trading-ecosystem (Phase 6).**

| Concern | Pick (free → paid) |
|---|---|
| Env / packaging | `uv` (fast) or `poetry` |
| Data wrangling | `pandas`, `numpy` |
| Market data | `yfinance` (free, fine to start) → Financial Modeling Prep / Tiingo / EOD Historical Data (UK + global, paid) |
| Fundamentals & filings | SEC EDGAR (US, free); for UK: RNS / Investegate / London Stock Exchange announcements, Companies House |
| Backtesting | Start with **plain pandas** (you must understand the mechanics), then `vectorbt` or `backtrader` |
| Caching / local store | DuckDB or SQLite (cache API pulls; never re-hit a paid API for the same data) |
| LLM | **Anthropic Python SDK** (`pip install anthropic`) |
| Charts / reports | `matplotlib`; optionally `python-docx` / `pypdf` for memo output |
| Tests | `pytest` |

**Claude model choices** (use the right tier per job — this controls cost):

| Job | Model | Why |
|---|---|---|
| PM reasoning, debate, final judgement | `claude-opus-4-8` | Most capable; long-horizon reasoning. Use adaptive thinking. |
| Per-candidate screening, sentiment, summarising filings | `claude-haiku-4-5` | Cheap and fast; you'll run it hundreds of times. |
| (Optional middle tier) bulk dossier synthesis | `claude-sonnet-4-6` | Balance of cost/quality. |

Set `ANTHROPIC_API_KEY` in your environment. The SDK reads it automatically.

> **Note for this repo's web sessions:** outbound network access is governed by your environment's network policy (see https://code.claude.com/docs/en/claude-code-on-the-web). If you build/run the picker inside a Claude Code web session, market-data APIs and `api.anthropic.com` must be reachable under the chosen policy.

---

## Phase 0 — Foundations & the scoring harness  *(Days 1–3)*

**Learning objectives**
- The data you can actually get, and its quirks (tickers, currencies, splits/dividends, total vs price return).
- How the competition *scores* — and how to reproduce it exactly.

**Build**
1. `data/prices.py` — pull adjusted daily prices for a list of tickers, cache to DuckDB/SQLite.
2. `scoring.py` — given a basket and a start/end date, compute the **exact metric the competition uses** (e.g. equal-weighted total return). This is your source of truth for every later experiment.
3. A tiny CLI: `python -m picker.score --tickers AZN.L,SHEL.L,... --from 2025-01-01 --to 2025-12-31`.

**DoD:** you can take any basket + date range and produce the competition's score number, and you've sanity-checked it by hand on 2 names.

---

## Phase 1 — Deterministic baseline picker + backtest  *(Week 1)*

This is your **first competition-ready entry** and the **baseline every later version must beat**. No LLM yet — that's deliberate. If a simple factor screen beats your fancy agent, you need to know.

**Learning objectives**
- Factor investing basics: **value, momentum, quality, size, low-vol** — what each is and how to compute a simple proxy.
- Backtesting mechanics and its three deadly sins:
  - **Lookahead bias** (using data you wouldn't have had at decision time),
  - **Survivorship bias** (universe excludes delisted/failed names),
  - **Data snooping / overfitting** (tuning until the backtest looks great).
- Benchmark-relative thinking (return vs the index, not just absolute).

**Build**
- `factors.py` — compute a composite score over the universe (e.g. z-score of 12-1 month momentum + earnings yield + ROE), point-in-time.
- `baseline_pick.py` — rank by composite, take top N, output tickers + a one-line reason each.
- `backtest.py` — walk the strategy over several historical periods, score each with `scoring.py`, compare to the index.

**DoD:** a reproducible backtest report (CSV + a chart) showing baseline vs benchmark across ≥5 historical windows, with a written paragraph on where it wins/loses and *why you don't fully trust the number* (list the biases that remain).

---

## Phase 2 — First LLM agent: single agent with tools  *(Week 2)*

Now introduce Claude as the **researcher/judge**, grounded in tool data. The goal: ranked picks **with rationale**, where every factual claim comes from a tool call, not the model's memory.

**Learning objectives**
- The Claude **tool-use loop** (model asks for a tool → you run it → feed result back → repeat).
- **Structured outputs** so picks come back as validated JSON, not prose.
- **Adaptive thinking** and the **effort** parameter for reasoning depth vs cost.
- The cardinal rule: **the LLM must not state numbers it didn't get from a tool** (training-cutoff + hallucination risk). Enforce via prompt + by only giving it tool-sourced data.

**Build** — tools the agent can call: `get_fundamentals(ticker)`, `get_price_history(ticker)`, `get_factor_scores(ticker)`, `get_recent_news(ticker)` (Phase 3 enriches these). The agent ranks the Phase-1 candidate shortlist and returns structured picks.

**Reference skeleton** (Anthropic Python SDK; tool-runner handles the loop):

```python
import anthropic
from anthropic.lib.tools import beta_tool

client = anthropic.Anthropic()  # reads ANTHROPIC_API_KEY

@beta_tool
def get_fundamentals(ticker: str) -> str:
    """Return key fundamentals (P/E, ROE, debt/equity, revenue growth) as JSON for a ticker."""
    return fundamentals_lookup(ticker)  # your cached data layer — never the model's memory

# Strict JSON shape for the picks so the result is machine-checkable
PICK_SCHEMA = {
    "type": "object",
    "additionalProperties": False,
    "properties": {
        "picks": {
            "type": "array",
            "items": {
                "type": "object",
                "additionalProperties": False,
                "properties": {
                    "ticker": {"type": "string"},
                    "conviction": {"type": "integer"},      # 1-5
                    "thesis": {"type": "string"},
                    "key_risks": {"type": "string"},
                },
                "required": ["ticker", "conviction", "thesis", "key_risks"],
            },
        }
    },
    "required": ["picks"],
}

runner = client.beta.messages.tool_runner(
    model="claude-opus-4-8",
    max_tokens=8000,
    thinking={"type": "adaptive"},
    output_config={"effort": "high", "format": {"type": "json_schema", "schema": PICK_SCHEMA}},
    tools=[get_fundamentals],  # add the other tools here
    messages=[{
        "role": "user",
        "content": (
            "From this shortlist, choose the 5 strongest long-only picks for a 12-month hold. "
            "Use the tools for every number you cite; do not rely on prior knowledge. "
            f"Shortlist: {shortlist}"
        ),
    }],
)
final = runner.until_done()
```

> API notes (current as of this plan): on Opus 4.8 use `thinking={"type":"adaptive"}` — `budget_tokens` and `temperature`/`top_p` are removed and will 400. Structured output goes in `output_config={"format": {...}}` (the old top-level `output_format` is deprecated). Stream if you ever set `max_tokens` above ~16K.

**DoD:** the agent outputs valid structured picks with tool-grounded theses, and you've run them through `scoring.py`/`backtest.py` head-to-head against the Phase-1 baseline. Record which wins and by how much.

---

## Phase 3 — Research depth: filings, news, sentiment, catalysts  *(Weeks 3–4)*

Give the agent real reading material so its judgement is informed, not vibes.

**Learning objectives**
- Ingesting and chunking long documents (annual reports / 10-Ks, earnings/results statements, RNS announcements).
- **Retrieval** (pull the relevant passages into context) and **citations** (every claim links to a source).
- **Sentiment & catalyst extraction** from noisy text; separating signal from PR spin.
- Using Claude's **server-side `web_search` / `web_fetch` tools** for current events past the training cutoff.

**Build**
- `ingest.py` — fetch filings/announcements, store text + metadata.
- A per-company **research dossier** the agent reads before judging.
- Add `web_search` / `web_fetch` server tools to the agent for recent developments.
- **Prompt caching** on the big, stable parts of the prompt (system instructions + dossiers) to cut cost when you run many candidates.

**DoD:** for any candidate, you can generate a dossier; the agent's thesis cites specific sources; and a quick ablation shows whether the dossier changes picks vs Phase 2.

---

## Phase 4 — Multi-agent debate + portfolio construction  *(Weeks 5–6)*

Move from "rank some stocks" to "construct a defensible portfolio under the competition's constraints."

**Learning objectives**
- **Multi-agent patterns:** a **bull** analyst and a **bear** analyst argue each name; a **risk** agent flags concentration/liquidity; a **PM** agent makes the final allocation. Adversarial structure surfaces weak theses.
- **Portfolio construction:** position sizing by conviction, diversification, sector caps, and the competition's hard rules as constraints.
- Writing an **investment memo** per pick (the artifact a real PM would defend).

**Build**
- Per candidate: bull case → bear case → synthesis (you can run bull/bear cheaply on Haiku, synthesis on Opus).
- A PM step that takes the surviving names + the rules and emits a sized portfolio + memos.
- Encode the competition constraints from `competition-rules.md` as explicit checks the PM output must satisfy.

> If you want the orchestration managed for you rather than hand-rolling the loop, Anthropic's **Managed Agents** can run a coordinator + sub-agents (bull/bear/risk) server-side. Optional — only reach for it once the hand-rolled version works and you want less plumbing.

**DoD:** the pipeline emits a rules-valid portfolio with sizing and one memo per holding, and it backtests at least as well as Phase 2/3 on your harness.

---

## Phase 5 — Evaluation rigor & guardrails  *(Weeks 7–8)*

The most important phase for not fooling yourself. Prove (or disprove) that the agent adds value.

**Learning objectives**
- **Walk-forward / out-of-sample** testing: never tune and test on the same data.
- **Overfitting & data snooping** controls; the more configurations you try, the luckier some look.
- **Transaction costs, liquidity, and point-in-time data** (no using restated/forward data).
- Metrics beyond return: **alpha vs benchmark, Sharpe, hit rate, max drawdown**, and stability across regimes.
- **Ablations:** baseline vs LLM vs LLM+dossier vs multi-agent — does each layer actually help, net of cost?

**Build**
- A clean evaluation harness with strict train/test separation and a leakage audit checklist.
- A one-page **scorecard** comparing every version honestly.

**DoD:** you can state, with evidence, *whether and where* the agentic layers beat the deterministic baseline — and you're comfortable defending the number.

---

## Phase 6 — Productionise: submission, journaling, reflection loop  *(Ongoing)*

Turn it into something you actually run each competition cycle, and that improves over time.

**Learning objectives**
- Scheduling/automation; a one-command (or cron-triggered) run that produces the entry.
- **Memory across runs:** persist each cycle's picks, theses, and outcomes; feed prior post-mortems back into the next run so the system learns from its mistakes.
- Monitoring live performance vs the basket you submitted.

**Build**
- `run_cycle.py` — produce the formatted competition entry + an investment-committee report.
- A **journal** (DuckDB table or memory store): picks, rationale, scores, and a post-cycle reflection write-up.
- A reflection step: at period end, the agent reviews what worked, writes lessons to memory, and those lessons become context for the next cycle.

**Optional — integrate with your Go trading-ecosystem:**
- Your `exchange-simulator-go` already does **order matching + account/position tracking**. It can serve as a **paper-trading ledger** for the submitted basket: open positions at entry, mark-to-market daily, and track P&L with the same machinery you use for crypto. (Equities aren't the sim's native asset, but the account/position/market-data services map cleanly onto a long-only equity basket.)
- Expose the picker as a **gRPC service** following the repo's clean-architecture + TDD patterns, so it fits the ecosystem (config client, service discovery, metrics, graceful shutdown) you've already built.

**DoD:** one command produces a valid, logged entry; after a cycle, the system records outcomes and a reflection that measurably informs the next run.

---

## Cross-cutting: the pitfalls that sink stock pickers

Keep this list visible; most failures are one of these.

1. **Lookahead bias** — using data (prices, restated fundamentals, news) you wouldn't have had at decision time. The single most common backtest lie.
2. **Survivorship bias** — testing only on companies that still exist today flatters every strategy.
3. **Overfitting / data snooping** — tuning until the backtest sparkles; it won't generalise. Fewer knobs, out-of-sample discipline.
4. **LLM hallucinated financials** — the model "remembering" a P/E or a revenue figure. Ground every number in a tool; forbid memory-sourced facts in the prompt.
5. **Single-shot variance** — one period is mostly noise. A sound process can lose; don't over-update on one result.
6. **Costs & liquidity** — spreads, fees, and thin small-caps erode paper returns.
7. **Cost blowups (API)** — running Opus over hundreds of candidates. Use Haiku for bulk steps, prompt caching for stable context, and cache all market data locally.

---

## Suggested weekly rhythm

- **~60%** building the current phase's deliverable.
- **~25%** learning the concept behind it (one focused topic/week: factors → backtesting → tool use → retrieval → multi-agent → evaluation).
- **~15%** running it against the scoring harness and writing down what you learned.

Ship something runnable every week. Keep `docs/competition-rules.md` and a running `docs/decision-journal.md` as the two living documents.

---

## Quick-start checklist

- [ ] Get the competition rules in writing → `docs/competition-rules.md`
- [ ] Python env (`uv`/`poetry`), `ANTHROPIC_API_KEY` set, market-data account
- [ ] Phase 0: prices puller + scoring harness (reproduce the competition metric)
- [ ] Phase 1: deterministic factor baseline + honest backtest
- [ ] Phase 2: single Claude agent with tool use + structured picks
- [ ] Phase 3: dossiers (filings/news) + web search + prompt caching
- [ ] Phase 4: bull/bear/risk/PM multi-agent + portfolio construction
- [ ] Phase 5: walk-forward evaluation + ablation scorecard
- [ ] Phase 6: automated submission + journal + reflection (optional Go integration)
