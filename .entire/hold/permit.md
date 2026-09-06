# Hold continuation permit

Briefing for the next agent. **Not enforced.** Enforcement is `entire hold check` (exit 1).

## Context: complete

## Frozen

| id | symbol | file:line | constraint | checkpoint |
|----|--------|-----------|------------|------------|
| H001 | charge | demo/charge-api/src/charge.ts:4 | Build Hold. Do not change public charge(). Retries must s... | b352152c8622 |

## UNBOUND

(none)

## OPEN

- `H002` Add tests for every caller of charge
- `H005` VIP customers never receive more than 50% total discount, regardless of overlapping promotions.
- `H006` Raw prompts and transcripts must not be sent to a new external service. Compile, status, permit, and check must still run when checkpoint text is redacted or missing, printing UNBOUND/OPEN/UNVERIFIED instead of freezing holes. Never present incomplete context as a complete permit. Do not weaken the charge() freeze.

## UNVERIFIED

- `H003` caller webhookRetry has no tests in checkpoint tool JSONL
- `H004` caller invoiceCharge has no tests in checkpoint tool JSONL

## Last amend

Raw prompts and transcripts must not be sent to a new external service. Compile, status, permit, and check must still run when checkpoint text is redacted or missing, printing UNBOUND/OPEN/UNVERIFIED instead of freezing holes. Never present incomplete context as a complete permit. Do not weaken the charge() freeze.
Reason: noon Track 1 privacy boundary: no raw transcript egress

## How to check

```
entire hold check
entire hold prove --test "npm test"
```

`HOLD_BYPASS=1` skips the pre-commit hook. Do not use it in the demo.
