# Hold demo (E1)

Gate first. Then amend. Then catch the silent VIP regression.

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
