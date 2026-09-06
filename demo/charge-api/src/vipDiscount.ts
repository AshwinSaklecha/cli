import type { ChargeReq } from "./types.ts";

/** 40% for VIP customers. Historical intent: total discount never exceeds 50%. */
export function vipDiscount(req: ChargeReq): number {
  if (!req.vip) {
    return 0;
  }
  return 0.4;
}
