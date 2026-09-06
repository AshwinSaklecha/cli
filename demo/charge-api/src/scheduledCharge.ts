import { charge } from "./charge.ts";
import type { ChargeReq, ChargeResult } from "./types.ts";

export function scheduledCharge(req: ChargeReq): ChargeResult {
  return charge(req);
}
