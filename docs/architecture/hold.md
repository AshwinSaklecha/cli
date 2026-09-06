# Hold — architecture (E1, BTW Buildathon 2026)

Hold is a **gate**, not a report. It compiles lived checkpoint promises onto Graph
symbols, fails any later edit that intersects a frozen neighborhood (`check`
exit 1), and `prove`s remaining change against resurrected Checkpoint intent of
Graph dependents.

A static `AGENTS.md` cannot scan a transcript, bind `charge()`, compute blast
radius, `graph diff` the working tree, and **exit 1**. That failed check is the
product.

## Why Entire is essential

Remove Entire and you are guessing from a git diff. Checkpoint
prompts/transcripts/tool-JSONL are the only source of constraints. Graph
`search` / `def` / `impact` / `diff` / `checkpoint` / `verify` / `why` / `blame`
are the only bind, freeze, check, and dependent-intent walk.

## Surfaces (no invented APIs)

- Checkpoints: `entire checkpoint list --json`, `explain --full` / `--raw-transcript`
- Graph plugin (exec only, never add egress): `search`, `def`, `impact`,
  `neighbors`, `diff`, `checkpoint`, `verify`
- Authorship: `entire why` / `entire blame`, then `Entire-Checkpoint:` trailers

## Commands

```
entire hold compile   [--from-checkpoint ID] [--all] [--json]
entire hold status    [--json]
entire hold permit    [--out .entire/hold/permit.md]
entire hold check     [--base REF] [--head REF] [--json]
entire hold amend     "new constraint" [--supersede ID --reason "..."] [--json]
entire hold prove     [--test "npm test"] [--json]
entire hold publish   [--dry-run]   # optional Databricks; never load-bearing
entire hold pull                    # optional Databricks
```

Preferred: Cobra group in this fork (`cmd/entire/cli/hold.go` + `internal/hold`).
Fallback only if a built-in name collision appears: kubectl-style `entire-hold`
on PATH. Built-ins win.

## Charter (`.entire/hold/charter.json`)

Closed statuses: `UNBOUND`, `FROZEN`, `OPEN`, `UNVERIFIED`, `CONFLICT`,
`SUPERSEDED`, `INTENT_REGRESSION`, `INTACT_PROVEN`.

- **UNBOUND never enters `freeze_set`.** Wrong bind = failed demo; prefer UNBOUND.
- **FROZEN** = Graph-cited symbol ∪ `graph impact` neighborhood (`--freeze-depth` default 2).
- **OPEN** = promised tests/symbols Graph cannot find.
- **UNVERIFIED** = impact callers whose tests were not in checkpoint tool JSONL.
- `permit.md` is briefing only — **never parsed by `check`**.

`dependency_policy`: FROZEN “no new runtime deps” → hash of `package.json` / `go.mod`.

## Enforcement (`check`) — no LLM

1. Load charter; missing → exit 2.
2. Entity change set from `graph diff` (+ worktree if supported).
3. `changed ∩ freeze_set` (file+name, else file+line range).
4. Manifest hash vs `dependency_policy`.
5. Intersection or manifest break → **exit 1** with `file:line` and caller count.
6. Write `.entire/hold/last-check.json`.
7. Install a **pre-commit** hook running `entire hold check`. Entire’s managed
   git hooks are `prepare-commit-msg`, `commit-msg`, `post-commit`,
   `post-rewrite`, `pre-push` — Hold must not overwrite those. `HOLD_BYPASS=1`
   is documented, never used in the demo.

Failed check copy must include: frozen symbol `file:line`, checkpoint prefix,
constraint quote, changed kind, `graph impact` callers (~6), and:

> A static AGENTS.md cannot compute this intersection.

## Amend (noon path)

Extract + bind the new sentence. Overlap + contradiction → `CONFLICT` (freeze
still enforces). `--supersede ID --reason` → `SUPERSEDED` + new `FROZEN`.
Default: noon may allow a new parameter on `charge` **but** keep idempotency
freeze; SUPERSEDE signature freeze only if noon text says so.

## Prove — two legs, both required; Databricks unplugged

**Leg A:** still-`FROZEN` symbols must be empty in `graph diff` / `graph checkpoint`.
Run targeted tests (`graph verify` when available; fixture `npm test` fallback).

**Leg B (dependent-intent):** walk CALLS-in neighbors of the legal change;
resurrect each dependent’s authoring Checkpoint (`why`/`blame`/git trailers);
extract historical constraints; evaluate **deterministically** in
`internal/hold/intentcheck`. Fixture composition: VIP 40% + holiday 20% = 60%
violates “VIP never more than 50% total discount” while unit tests still pass.
Any `INTENT_REGRESSION` → `prove` exits 1. Write `last-prove.json`.

## Fixture `demo/charge-api`

Tiny TypeScript mock payments app. Hold is tested here, not a website.

- `charge()` — do not change public signature
- `retryCharge` — idempotent
- ≥6 callers of `charge` (`processPayment`, `webhookRetry`, plus wrappers)
- Tests for charge/retry; at least one caller untested → UNVERIFIED
- `vipDiscount` 40% with historical “VIP ≤ 50% total” checkpoint quote
- Later legal `holidayBonus` +20% with **no** composition test until after prove
- `package.json` hashed for “no new runtime dependency”

## Layout

```
cmd/entire/cli/hold.go          # Cobra group
internal/hold/                  # extract, bind, charter, check, amend, prove
internal/hold/intentcheck.go    # deterministic dependent-intent
internal/hold/graphexec.go      # exec `entire graph …`
internal/hold/entireexec.go     # exec `entire checkpoint|why|blame …`
demo/charge-api/                # fixture
docs/architecture/hold.md       # this plan
BUILDATHON.md                   # after final verification, before submit
```

Engine tests use stub Graph/`entire` clients and testdata JSON (no live Entire
required). Live exec is for compile/check/prove against this clone.

## Cut order if time dies

Databricks → Next.js cockpit → amend UI.

**Never cut** `compile` / `check` / `amend` / `prove` (including dependent-intent).

## Checkpoint story (this fork)

1. **This commit** — initial plan/architecture + Entire setup leftovers
   (Cursor hooks, settings with checkpoints on *this* origin, Graph agent
   guidance, fixture README).
2. Pre-noon stable — working Hold + fixture; `check` fails an illegal `charge`
   refactor.
3. Noon curveball — fresh session reconstructs from checkpoints + Graph, then
   `hold amend`.
4. Final verification — constraint implemented, `check`+`prove`, BUILDATHON.md.

Checkpoints sync to origin `entire://aws-ap-south-1.entire.io/gh/ashwinsaklecha/cli`
(India/Mumbai). Do not point `checkpoint_remote` at `entireio/cli-checkpoints`.

## Graph usage (mandatory, evidence not oracle)

- Search / def lookup while locating CLI registration and hook install.
- `graph impact` before high-risk edits (`root.go`, hook install, `charge()`).
- Final `graph diff` of submitted work, verified against source and tests.
- Never show raw Graph output as the product.

## Non-goals

Chat/RAG, LLM-as-judge of `check`, Databricks as the live blocker, review
dashboard, markdown-only permit, cockpit before Beat A, forking graph/external-agents.
