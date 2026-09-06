import assert from "node:assert/strict";
import test from "node:test";
import { resetLedger } from "./charge.ts";
import { retryCharge } from "./retry.ts";

test("retry-same-key-does-not-double-charge", () => {
  resetLedger();
  const req = { id: "r1", amount: 80, customerId: "u1" };
  const first = retryCharge(req);
  const second = retryCharge(req);
  assert.deepEqual(first, second);
  assert.equal(first.captured, 80);
});
