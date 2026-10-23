import { test } from "node:test";
import assert from "node:assert/strict";
import { planAlerts, formatAlert, formatHeartbeat } from "../src/alerts.js";

const result = (name, status) => ({
  name,
  status,
  servers: [{ label: "a.example", ok: status !== "down", detail: status === "down" ? "HTTP 502" : "" }]
});

test("a category going down alerts once", () => {
  const first = planAlerts([result("Blockbook", "down")], { Blockbook: "ok" });
  assert.deepEqual(
    first.down.map((r) => r.name),
    ["Blockbook"]
  );

  const second = planAlerts([result("Blockbook", "down")], first.state);
  assert.equal(second.down.length, 0, "no repeat while it stays down");
});

test("recovery is announced", () => {
  const plan = planAlerts([result("Blockbook", "ok")], { Blockbook: "down" });
  assert.deepEqual(
    plan.recovered.map((r) => r.name),
    ["Blockbook"]
  );
});

test("degraded does not alert", () => {
  const plan = planAlerts([result("Blockbook", "degraded")], { Blockbook: "ok" });
  assert.equal(plan.down.length, 0);
  assert.equal(plan.recovered.length, 0);
});

test("going from down to degraded counts as recovered", () => {
  const plan = planAlerts([result("Blockbook", "degraded")], { Blockbook: "down" });
  assert.equal(plan.recovered.length, 1);
});

test("without stored state a down category alerts every run", () => {
  const plan = planAlerts([result("Blockbook", "down")], null);
  assert.equal(plan.down.length, 1);
});

test("the first run with empty state alerts for what is already down", () => {
  const plan = planAlerts([result("Blockbook", "down")], {});
  assert.equal(plan.down.length, 1);
});

test("messages list the failing servers and stay under Discord's limit", () => {
  const plan = planAlerts([result("Blockbook", "down")], {});
  const text = formatAlert(plan, [result("Blockbook", "down")]);
  assert.match(text, /Blockbook is down/);
  assert.match(text, /a\.example: HTTP 502/);

  const huge = { name: "X", status: "down", servers: [{ label: "s", ok: false, detail: "x".repeat(5000) }] };
  assert.ok(formatAlert({ down: [huge], recovered: [] }, [huge]).length <= 2000);
});

test("heartbeat summarizes every category", () => {
  const text = formatHeartbeat([result("A", "ok"), result("B", "degraded")]);
  assert.match(text, /A: ok/);
  assert.match(text, /B: degraded/);
});
