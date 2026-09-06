# charge-api

BTW fixture — payments toy app. Hold compiles and checks against this.

This is a tiny TypeScript mock, not a website.

- `charge()` — public signature must not change
- `retryCharge` — idempotent
- Callers of `charge`: `processPayment`, `webhookRetry`, `invoiceCharge`, `scheduledCharge`, `refundRetry`, `adminCharge`
- `vipDiscount` 40% with historical intent: VIP total discount never exceeds 50%
- `holidayBonus` flat 20% — unit tests do not compose the two

```
npm test
```

Hold:

```
entire hold compile
entire hold check
entire hold prove --test "npm test"
```
