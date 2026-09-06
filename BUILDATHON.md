# Hold — BTW Buildathon 2026

**Project:** Hold  
**Track:** E1 — Checkpoint-Native Developer Experience  
**Team:** AshwinSaklecha  
**Repo:** https://github.com/AshwinSaklecha/cli  
**Entire mirror:** `entire://aws-ap-south-1.entire.io/gh/ashwinsaklecha/cli` (India / Mumbai)

## User and problem

After an agent session dies, git still has the new code and the promises are gone. “Do not change `charge()`” is not enforceable as `AGENTS.md`. Hold compiles those checkpoint promises onto Graph symbols, fails an illegal edit with exit 1, amends when the plan changes, and `prove`s remaining change against dependents’ historical intent (VIP + holiday silent regression).

## Original plan (morning)

Local gate on `demo/charge-api`:

1. `compile` — extract constraints from `entire checkpoint explain` (local regex) → Graph bind → freeze `charge()` neighborhood  
2. `check` — `graph diff` ∩ freeze_set → **exit 1** on an illegal `charge()` edit  
3. `amend` — noon path (CONFLICT / SUPERSEDED) without silently dropping a freeze  
4. `prove` — freeze intact + dependent-intent (VIP ≤ 50% vs holiday 20%)

Databricks and any cockpit were last and optional. `check` / `prove` must work unplugged.

## Assumption that changed (Noon Curveball — Track 1 Privacy Boundary)

Morning Hold assumed compile could later send **raw transcripts** to an external LLM for extraction.

The noon constraint forbids sending raw prompts/transcripts to a **new external service**. The product must still give useful output when checkpoint text is **redacted or missing**, and must **never present incomplete context as complete**.

## How the design changed

Smallest local change. Hold was not rebuilt.

- Extract stays **local regex**. `EntireClient.ExplainFull` / `ExplainRawTranscript` are `entire checkpoint explain` subprocesses, not HTTP. No OpenAI, no Databricks on compile.
- Redacted Entire tokens (`REDACTED`, `[REDACTED_…]`) and missing explain output mark the charter `context_quality: incomplete`.
- Incomplete compile still runs and writes UNBOUND / OPEN / UNVERIFIED. **Nothing is frozen from a redacted hole.**
- `check` still **exit 1** on a real `charge()` freeze hit. An incomplete charter never prints `HOLD CHECK PASSED`.
- Fixture: `internal/hold/testdata/redacted_checkpoint.jsonl` (Entire-style JSONL; no separate official file was attached to the Word card).

`entire hold amend` recorded the privacy rule on the pre-noon charter. `charge()` freeze **H001** is unchanged.

## Why it is safe

- Intersection `check` and VIP/holiday `prove` tests are unchanged and still pass.
- Complete, unredacted checkpoints still freeze `charge()` (`TestCompile_CompleteCheckpointStillFreezesCharge`).
- A redacted blob cannot freeze even if Graph can cite a lookalike symbol (`TestCompile_RedactedCheckpoint_IncompleteNoFakeFrozen`).
- Missing explain does not panic (`TestCompile_MissingCheckpoint_IncompleteNoPanic`).

## Dependencies / prior work / AI-generated parts

- Fork of [entireio/cli](https://github.com/entireio/cli); Graph used as the installed plugin (`entire graph …`), not forked.
- Fixture app: `demo/charge-api` (TypeScript).
- Implementation and this curveball pass were produced in Cursor agent sessions against Entire checkpoints.

## Pre-noon state

Working `entire hold compile / check / amend / prove` on `demo/charge-api`. Illegal uncommitted `charge()` edit failed check (see `.entire/hold/last-check.json`).

## Assigned curveball

**Track 1 — Privacy Boundary** (`D:\BTW Hackathon\12_00 _ The Noon Curveball is live.docx`).

## Adaptation

See “Assumption that changed” and “How the design changed”. Graph impact was run **before** edits on `ExtractConstraints`, `Compile`, and `newHoldCompileCmd`.

## Verification

```
go test ./internal/hold
```

Covered: illegal `charge()` intersection fail; VIP+holiday INTENT_REGRESSION; redacted/missing checkpoint → incomplete output, no fake FROZEN, no `HOLD CHECK PASSED`.

## Graph findings (before edits)

| Symbol | File | Callers (impact) |
|--------|------|------------------|
| `ExtractConstraints` | `internal/hold/extract.go` | `Compile`, `Amend`, `resurrectQuotes` |
| `Compile` | `internal/hold/compile.go` | `newHoldCompileCmd` → callees `ExplainFull`, `ExplainRawTranscript`, `ExtractConstraints`, `Bind` |
| `newHoldCompileCmd` | `cmd/entire/cli/hold.go` | `newHoldGroupCmd` → `NewRootCmd` |

Explain path is local subprocess, not `Client.Post`. No `net/http` in `internal/hold`.

## Checkpoint links

| When | ID | Commit |
|------|-----|--------|
| Initial Hold gate + fixture | `01M1TN87DYXKAN6629B0HPMV94` | `80ecdc6` hold: add compile/check/amend/prove gate and charge-api fixture |
| Pre-noon stable (check on uncommitted charge) | `01M1TNC2Z36KRPTTVATV7T1FT1` | `b5e7e15` hold: fail check on uncommitted charge() edits |
| Curveball (privacy code) | *(attached on the following agent commit)* | `c7ff40d` hold: keep extract local under the noon privacy boundary |

Reconstruct:

```
entire checkpoint list
entire checkpoint explain 01M1TNC2Z36KRPTTVATV7T1FT1
```

## Limitations

- Official redacted fixture was named on the Word card but not present as a separate file in the docx; testdata mirrors Entire’s `REDACTED` / `[REDACTED_LABEL]` tokens.
- Databricks publish/pull remain omitted on purpose.
- No OpenAI extract path exists and none will be added.

## Next steps

Keep extract local. If a later session must classify messy prose, do it on-machine or from already-redacted quotes — never POST raw transcripts.

## Demo access

CLI in this clone: `entire hold compile|status|check|amend|prove`.  
Tests: `go test ./internal/hold`.  
Cockpit: not built (CLI is the product).
