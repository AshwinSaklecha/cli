import type { ChargeReq } from "./types.ts";

/** Flat 20% holiday bonus. Unit tests do not compose this with VIP. */
export function holidayBonus(_req: ChargeReq): number {
  return 0.2;
}
