import { test, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";
import { run, plan, MINUTE_CRON, DAILY_CRON } from "../src/index.js";

const FRIDAY = Date.parse("2026-10-23T12:00:00Z");
const MONDAY = Date.parse("2026-10-26T12:00:00Z");

const json = (body, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });

const state = { kv: null, requests: [], discord: [], down: new Set() };
const realFetch = globalThis.fetch;

function env() {
  return {
    DISCORD_WEBHOOK_URL: "https://discord.test/hook",
    BLOCKBOOK_URLS: "https://bb1.example/",
    BACKEND_URLS: "https://be1.example/,https://be2.example/",
    PROBE_ADDRESS: "DTest",
    FAILURES_BEFORE_ALERT: "5",
    DAILY_FAILURES_BEFORE_ALERT: "2",
    MENTIONS: JSON.stringify({ blockbook: "111", coinpaprika: "111", "be1.example": "222", "be2.example": "333" }),
    STATE: {
      get: async () => state.kv,
      put: async (_key, value) => {
        state.kv = value;
      }
    }
  };
}

beforeEach(() => {
  state.kv = null;
  state.requests = [];
  state.discord = [];
  state.down = new Set();
  globalThis.fetch = async (url, options = {}) => {
    const u = String(url);
    if (u.startsWith("https://discord.test")) {
      state.discord.push(JSON.parse(options.body));
      return new Response(null, { status: 204 });
    }
    state.requests.push(u);
    for (const host of state.down) {
      if (u.includes(host)) return new Response("down", { status: 502 });
    }
    if (u === "https://bb1.example/api/v2") {
      return json({ blockbook: { inSync: true, lastBlockTime: new Date().toISOString() } });
    }
    if (u === "https://bb1.example/api/v2/utxo/DTest") return json([]);
    if (u === "https://bb1.example/api/v2/sendtx/") return json({ error: "-22: TX decode failed" }, 400);
    if (u.endsWith("/info")) return json({ backendTipHeight: 5, tipHeight: 5, difficulty: 1.5 });
    if (u.startsWith("https://api.coinpaprika.com/")) return json({ quotes: { BTC: { price: 1e-6 } } });
    throw new Error(`unexpected request ${u}`);
  };
});

afterEach(() => {
  globalThis.fetch = realFetch;
});

const touched = (host) => state.requests.some((u) => u.includes(host));

test("every minute only the backends are checked", async () => {
  await run(env(), MINUTE_CRON, FRIDAY);
  assert.ok(touched("be1.example") && touched("be2.example"));
  assert.ok(!touched("bb1.example"), "Blockbook is left alone");
  assert.ok(!touched("coinpaprika"), "CoinPaprika is left alone");
});

test("the daily run checks Blockbook and CoinPaprika but not the backends", async () => {
  await run(env(), DAILY_CRON, FRIDAY);
  assert.ok(touched("bb1.example") && touched("coinpaprika"));
  assert.ok(!touched("be1.example") && !touched("be2.example"));
  assert.equal(state.discord.length, 0, "nothing to say on a healthy Friday");
});

test("Blockbook has to fail two daily rounds before it alerts", async () => {
  state.down.add("bb1.example");

  await run(env(), DAILY_CRON, FRIDAY);
  assert.equal(state.discord.length, 0, "no alert after the first failed day");

  await run(env(), DAILY_CRON, FRIDAY + 24 * 3600 * 1000);
  assert.equal(state.discord.length, 1, "alert after the second");
  assert.match(state.discord[0].content, /Blockbook \(Dogecoin wallet\) is down\*\* <@111>/);
  assert.match(state.discord[0].content, /failed 2 checks in a row/);
  assert.deepEqual(state.discord[0].allowed_mentions, { users: ["111"] });

  await run(env(), DAILY_CRON, FRIDAY + 48 * 3600 * 1000);
  assert.equal(state.discord.length, 1, "and only once");
});

test("a Blockbook outage that ends after one day never alerts", async () => {
  state.down.add("bb1.example");
  await run(env(), DAILY_CRON, FRIDAY);
  state.down.clear();
  await run(env(), DAILY_CRON, FRIDAY + 24 * 3600 * 1000);
  state.down.add("bb1.example");
  await run(env(), DAILY_CRON, FRIDAY + 48 * 3600 * 1000);
  assert.equal(state.discord.length, 0, "failures that aren't in a row don't add up");
});

test("a backend alerts after five failed minutes and pings its own owner", async () => {
  state.down.add("be2.example");
  for (let minute = 1; minute <= 4; minute++) {
    await run(env(), MINUTE_CRON, FRIDAY + minute * 60000);
    assert.equal(state.discord.length, 0, `quiet after ${minute} failed minutes`);
  }
  await run(env(), MINUTE_CRON, FRIDAY + 5 * 60000);
  assert.equal(state.discord.length, 1);
  assert.match(state.discord[0].content, /OCM backend be2\.example is down\*\* <@333>/);
  assert.deepEqual(state.discord[0].allowed_mentions, { users: ["333"] });
});

test("the minute checks keep what the daily checks remembered, and the other way round", async () => {
  state.down.add("bb1.example");
  await run(env(), DAILY_CRON, FRIDAY);
  const afterDaily = JSON.parse(state.kv);
  assert.equal(afterDaily["Blockbook (Dogecoin wallet)"].failures, 1);

  await run(env(), MINUTE_CRON, FRIDAY + 60000);
  const afterMinute = JSON.parse(state.kv);
  assert.equal(afterMinute["Blockbook (Dogecoin wallet)"].failures, 1, "a minute run doesn't forget Blockbook");
  assert.ok(afterMinute["OCM backend be1.example"]);

  await run(env(), DAILY_CRON, FRIDAY + 24 * 3600 * 1000);
  assert.ok(JSON.parse(state.kv)["OCM backend be1.example"], "a daily run doesn't forget the backends");
});

test("a healthy steady state writes nothing to KV", async () => {
  const e = env();
  let writes = 0;
  const put = e.STATE.put;
  e.STATE.put = async (...args) => {
    writes++;
    return put(...args);
  };
  await run(e, MINUTE_CRON, FRIDAY);
  const first = writes;
  for (let i = 1; i <= 5; i++) await run(e, MINUTE_CRON, FRIDAY + i * 60000);
  assert.equal(writes, first, "no more writes once the state settled");
});

test("Monday's daily run sends the weekly summary with every service in it", async () => {
  await run(env(), DAILY_CRON, MONDAY);
  assert.equal(state.discord.length, 1);
  const text = state.discord[0].content;
  assert.match(text, /Weekly monitor check/);
  for (const name of ["Blockbook", "CoinPaprika", "OCM backend be1.example", "OCM backend be2.example"]) {
    assert.match(text, new RegExp(name.replace(/\./g, "\\.")));
  }
});

test("Monday's summary run doesn't count as an extra check for the backends", async () => {
  state.down.add("be1.example");
  for (let minute = 1; minute <= 3; minute++) await run(env(), MINUTE_CRON, MONDAY + minute * 60000);
  const before = JSON.parse(state.kv)["OCM backend be1.example"].failures;
  state.discord = [];
  await run(env(), DAILY_CRON, MONDAY + 3600000);
  assert.equal(JSON.parse(state.kv)["OCM backend be1.example"].failures, before);
  assert.equal(state.discord.length, 1, "just the summary, no backend alert");
  assert.match(state.discord[0].content, /Weekly monitor check/);
});

test("which checks a trigger runs", () => {
  assert.deepEqual(plan(MINUTE_CRON, MONDAY).check, ["backends"]);
  assert.equal(plan(MINUTE_CRON, MONDAY).summary, false);
  assert.deepEqual(plan(DAILY_CRON, FRIDAY).check, ["blockbook", "prices"]);
  assert.equal(plan(DAILY_CRON, FRIDAY).summary, false);
  assert.deepEqual(plan(DAILY_CRON, MONDAY).check, ["blockbook", "prices", "backends"]);
  assert.equal(plan(DAILY_CRON, MONDAY).summary, true);
});
