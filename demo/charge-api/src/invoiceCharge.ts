import { charge } from "./charge.ts";
import type { ChargeReq, ChargeResult } from "./types.ts";

export function invoiceCharge(req: ChargeReq): ChargeResult {
  return charge(req);
}
