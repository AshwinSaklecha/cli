import { charge } from "./charge.ts";
import type { ChargeReq, ChargeResult } from "./types.ts";

export function refundRetry(req: ChargeReq): ChargeResult {
  return charge(req);
}
