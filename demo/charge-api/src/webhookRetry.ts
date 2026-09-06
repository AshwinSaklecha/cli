import { charge } from "./charge.ts";
import type { ChargeReq, ChargeResult } from "./types.ts";

export function webhookRetry(req: ChargeReq): ChargeResult {
  return charge(req);
}
