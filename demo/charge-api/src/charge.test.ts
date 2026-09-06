import assert from "node:assert/strict";
import test from "node:test";
import { charge, resetLedger } from "./charge.ts";

test("charge captures amount", () => {
  resetLedger();
  const got = charge({ id: "c1", amount: 100, customerId: "u1" });
  assert.equal(got.captured, 100);
});

test("charge is idempotent on the same id", () => {
  resetLedger();
  const a = charge({ id: "c2", amount: 50, customerId: "u1" });
  const b = charge({ id: "c2", amount: 50, customerId: "u1" });
  assert.deepEqual(a, b);
});
