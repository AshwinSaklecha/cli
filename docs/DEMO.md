# Hold demo (E1)

Gate first. Then the **Track 1 privacy** amend. Then catch the silent VIP regression.

**Noon is not “allow a new parameter on charge().”** That example below is a *generic* amend illustration. The live curveball was privacy: no raw transcripts off-machine, redacted/missing checkpoints → incomplete, never a fake complete freeze. Do not supersede H001 (`charge()`) in the judging demo.

## Judging path

1. `entire hold status` — `charge` FROZEN; OPEN / UNVERIFIED; privacy amend on the charter.
2. Illegal edit to `demo/charge-api/src/charge.ts` → `entire hold check` **exit 1** (six callers, AGENTS.md line).
3. Show redacted/missing compile (`internal/hold/testdata/redacted_checkpoint.jsonl`) → **INCOMPLETE**, not `HOLD CHECK PASSED`.
4. Optional: `entire hold prove --test "npm test --prefix demo/charge-api"` for VIP+holiday INTENT_REGRESSION (fixture-local; do not overclaim).

## Beat A — illegal `charge()` refactor (exit 1)

1. `entire checkpoint list` — prompt includes “do not change charge()”.
2. `entire hold status` — FROZEN `charge` file:line, plus OPEN / UNVERIFIED.
3. Edit `demo/charge-api/src/charge.ts` (rename a parameter, “make it nicer”).
4. `entire hold check` — **fails**, `graph impact` callers, AGENTS.md line.

## Beat B — noon `amend`

```
entire hold amend "allow a new parameter on charge() but keep retries idempotent" --supersede H001 --reason "noon curveball"
```

Prints CONFLICT / SUPERSEDED / FROZEN.

## Beat C — prove Leg B

`npm test` in `demo/charge-api` stays green (no composition test).

```
entire hold prove --test "npm test"
```

Fails INTENT_REGRESSION: VIP ≤ 50% vs 40% + 20% holiday.

## Bypass (never in the demo)

`HOLD_BYPASS=1` skips the pre-commit `entire hold check`.
