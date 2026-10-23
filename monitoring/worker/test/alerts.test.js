import { test } from "node:test";
import assert from "node:assert/strict";
import { planAlerts, formatAlert, formatHeartbeat, mentionFor } from "../src/alerts.js";

const result = (name, status, key = name) => ({
  name,
  key,
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

test("the owner of a failing service is pinged by id", () => {
  const mentions = { "ocm.example": "379781811108315136", blockbook: "368416065644003339" };
  const down = result("OCM backend ocm.example", "down", "ocm.example");
  const text = formatAlert(planAlerts([down], {}), [down], mentions);
  assert.match(text, /OCM backend ocm\.example is down\*\* <@379781811108315136>/);
  assert.doesNotMatch(text, /368416065644003339/, "other owners are left alone");
});

test("a service without an owner alerts without pinging anyone", () => {
  const down = result("Blockbook", "down", "blockbook");
  const text = formatAlert(planAlerts([down], {}), [down], {});
  assert.match(text, /Blockbook is down\*\*\n/);
  assert.doesNotMatch(text, /<@/);
  assert.equal(mentionFor(down, {}), "");
});

test("recoveries and degraded services never ping", () => {
  const mentions = { blockbook: "1", ocm: "2" };
  const recovered = result("Blockbook", "ok", "blockbook");
  const degraded = result("Other", "degraded", "ocm");
  const text = formatAlert(planAlerts([recovered, degraded], { Blockbook: "down" }), [recovered, degraded], mentions);
  assert.doesNotMatch(text, /<@/);
});

test("a backend that stays down only pings on the first run", () => {
  const down = result("OCM backend ocm.example", "down", "ocm.example");
  const first = planAlerts([down], {});
  assert.equal(first.down.length, 1);
  const second = planAlerts([down], first.state);
  const third = planAlerts([down], second.state);
  assert.equal(second.down.length + third.down.length, 0);
});

test("a service must fail several checks in a row before it alerts", () => {
  const down = result("Blockbook", "down", "blockbook");
  let state = {};
  for (let check = 1; check <= 4; check++) {
    const plan = planAlerts([down], state, 5);
    assert.equal(plan.down.length, 0, `no alert after ${check} failed checks`);
    state = plan.state;
  }
  const fifth = planAlerts([down], state, 5);
  assert.equal(fifth.down.length, 1, "alerts on the fifth failed check");
  assert.equal(fifth.down[0].failures, 5);
  assert.match(formatAlert(fifth, [down], {}), /failed 5 checks in a row/);

  const sixth = planAlerts([down], fifth.state, 5);
  assert.equal(sixth.down.length, 0, "and only once");
});

test("a success in between starts the count over", () => {
  const down = result("Blockbook", "down", "blockbook");
  const up = result("Blockbook", "ok", "blockbook");
  let state = {};
  for (const r of [down, down, down, down, up, down, down, down, down]) {
    const plan = planAlerts([r], state, 5);
    assert.equal(plan.down.length, 0, "four failures, a success, then four more never reaches five");
    state = plan.state;
  }
  assert.equal(planAlerts([down], state, 5).down.length, 1, "but five in a row does");
});

test("recovery after a confirmed outage is announced once", () => {
  const down = result("Blockbook", "down", "blockbook");
  const up = result("Blockbook", "ok", "blockbook");
  let state = {};
  for (let i = 0; i < 5; i++) state = planAlerts([down], state, 5).state;
  const back = planAlerts([up], state, 5);
  assert.equal(back.recovered.length, 1);
  assert.equal(planAlerts([up], back.state, 5).recovered.length, 0);
});

test("a blip that never reached the threshold recovers silently", () => {
  const down = result("Blockbook", "down", "blockbook");
  const up = result("Blockbook", "ok", "blockbook");
  const blip = planAlerts([down], {}, 5);
  assert.equal(planAlerts([up], blip.state, 5).recovered.length, 0);
});

test("state from the first version of the monitor is understood", () => {
  const down = result("Blockbook", "down", "blockbook");
  assert.equal(planAlerts([down], { Blockbook: "down" }, 5).down.length, 0, "already alerted");
  assert.equal(planAlerts([result("Blockbook", "ok", "blockbook")], { Blockbook: "down" }, 5).recovered.length, 1);
});

test("a steady state never changes, so nothing needs to be written", () => {
  const down = result("Blockbook", "down", "blockbook");
  const ok = result("Other", "ok", "other");
  let state = {};
  for (let i = 0; i < 5; i++) state = planAlerts([down, ok], state, 5).state;
  const again = planAlerts([down, ok], state, 5).state;
  assert.equal(JSON.stringify(again), JSON.stringify(state));
});
