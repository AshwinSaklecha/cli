import assert from "node:assert/strict";
import test from "node:test";
import { holidayBonus } from "./holiday.ts";

test("holiday bonus is a flat 20 percent", () => {
  assert.equal(holidayBonus({ id: "h1", amount: 100, customerId: "u1" }), 0.2);
});
