import { charge } from "./charge.ts";
import { holidayBonus } from "./holiday.ts";
import { vipDiscount } from "./vipDiscount.ts";
import type { ChargeReq, ChargeResult } from "./types.ts";

/** Composes VIP + holiday onto a charge. No composition cap — prove must catch that. */
export function processPayment(req: ChargeReq): ChargeResult {
  const charged = charge(req);
  const vip = vipDiscount(req);
  const holiday = holidayBonus(req);
  const discount = vip + holiday;
  return {
    ...charged,
    discount,
    captured: charged.amount * (1 - discount),
  };
}
