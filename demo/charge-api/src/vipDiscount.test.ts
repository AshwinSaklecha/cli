import assert from "node:assert/strict";
import test from "node:test";
import { vipDiscount } from "./vipDiscount.ts";

test("VIP receives 40 percent", () => {
  assert.equal(vipDiscount({ id: "v1", amount: 100, customerId: "vip", vip: true }), 0.4);
});

test("non-VIP receives 0", () => {
  assert.equal(vipDiscount({ id: "v2", amount: 100, customerId: "std" }), 0);
});
