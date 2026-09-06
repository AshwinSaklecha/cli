import { charge } from "./charge.ts";
import type { ChargeReq, ChargeResult } from "./types.ts";

/** Retry the same charge. Must stay idempotent: same id never double-captures. */
export function retryCharge(req: ChargeReq): ChargeResult {
  return charge(req);
}
