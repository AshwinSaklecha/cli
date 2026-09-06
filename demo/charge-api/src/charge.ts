import type { ChargeReq, ChargeResult } from "./types.ts";

const ledger = new Map<string, ChargeResult>();

/** Public charge entry. Do not change this signature. */
export function charge(req: ChargeReq): ChargeResult {
  const existing = ledger.get(req.id);
  if (existing) {
    return existing;
  }
  const result: ChargeResult = {
    id: req.id,
    amount: req.amount,
    captured: req.amount,
  };
  ledger.set(req.id, result);
  return result;
}

export function resetLedger(): void {
  ledger.clear();
}
