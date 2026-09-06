# Hold continuation permit

Briefing for the next agent. **Not enforced.** Enforcement is `entire hold check` (exit 1).

## Frozen

| id | symbol | file:line | constraint | checkpoint |
|----|--------|-----------|------------|------------|
| H001 | charge | demo/charge-api/src/charge.ts:4 | Build Hold. Do not change public charge(). Retries must s... | b352152c8622 |

## OPEN

- `H002` Add tests for every caller of charge
- `H005` VIP customers never receive more than 50% total discount, regardless of overlapping promotions.

## UNVERIFIED

- `H003` caller webhookRetry has no tests in checkpoint tool JSONL
- `H004` caller invoiceCharge has no tests in checkpoint tool JSONL

## How to check

```
entire hold check
entire hold prove --test "npm test"
```

`HOLD_BYPASS=1` skips the pre-commit hook. Do not use it in the demo.

_Updated 2026-09-06T05:51:31Z_
