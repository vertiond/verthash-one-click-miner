import { test } from "node:test";
import assert from "node:assert/strict";
import { runChecks } from "../src/checks.js";

const NOW = Date.parse("2026-10-23T02:00:00Z");
const env = {
  BLOCKBOOK_URLS: "https://bb1.example/,https://bb2.example/",
  BACKEND_URLS: "https://be1.example/,https://be2.example/",
  PROBE_ADDRESS: "DTest",
  MINERS_JSON_URL: "https://raw.example/miners.json"
};

function json(body, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });
}

// Answers like healthy servers, `overrides` replace individual requests
function fakeFetch(overrides = {}) {
  return async (url, options = {}) => {
    const key = `${options.method || "GET"} ${url}`;
    if (key in overrides) {
      const answer = overrides[key];
      if (answer instanceof Error) throw answer;
      return answer;
    }
    for (const host of ["bb1.example", "bb2.example"]) {
      if (url === `https://${host}/api/v2`) {
        return json({ blockbook: { inSync: true, lastBlockTime: "2026-10-23T01:59:00Z" } });
      }
      if (url === `https://${host}/api/v2/utxo/DTest`) return json([]);
      if (url === `https://${host}/api/v2/sendtx/`) return json({ error: "-22: TX decode failed" }, 400);
    }
    for (const host of ["be1.example", "be2.example"]) {
      if (url === `https://${host}/info`) return json({ backendTipHeight: 5, tipHeight: 5, difficulty: 1.5 });
    }
    if (url.startsWith("https://api.coinpaprika.com/v1/tickers/")) return json({ quotes: { BTC: { price: 1e-6 } } });
    if (url === "https://raw.example/miners.json") {
      return json([{ url: "https://dl.example/a/Miner.zip" }, { url: "https://dl.example/a/Miner.zip" }]);
    }
    if (url === "https://dl.example/a/Miner.zip") return new Response(null, { status: 200 });
    throw new Error(`unexpected request ${key}`);
  };
}

const byName = (results, name) => results.find((r) => r.name.startsWith(name));

test("everything healthy reports ok", async () => {
  const results = await runChecks(env, fakeFetch(), () => NOW);
  assert.deepEqual(
    results.map((r) => r.status),
    ["ok", "ok", "ok", "ok"]
  );
});

test("one failing Blockbook server is degraded, not down", async () => {
  const results = await runChecks(env, fakeFetch({ "GET https://bb1.example/api/v2": json({}, 502) }), () => NOW);
  const bb = byName(results, "Blockbook");
  assert.equal(bb.status, "degraded");
  assert.match(bb.servers.find((s) => !s.ok).detail, /HTTP 502/);
});

test("all Blockbook servers failing is down", async () => {
  const results = await runChecks(
    env,
    fakeFetch({
      "GET https://bb1.example/api/v2": json({}, 502),
      "GET https://bb2.example/api/v2/utxo/DTest": new Error("timed out")
    }),
    () => NOW
  );
  assert.equal(byName(results, "Blockbook").status, "down");
});

test("a Blockbook server that fell behind the chain fails", async () => {
  const stale = json({ blockbook: { inSync: true, lastBlockTime: "2026-10-23T00:00:00Z" } });
  const results = await runChecks(env, fakeFetch({ "GET https://bb1.example/api/v2": stale }), () => NOW);
  assert.match(byName(results, "Blockbook").servers[0].detail, /minutes old/);
});

test("a Blockbook server that is not in sync fails", async () => {
  const notSynced = json({ blockbook: { inSync: false, lastBlockTime: "2026-10-23T01:59:00Z" } });
  const results = await runChecks(env, fakeFetch({ "GET https://bb1.example/api/v2": notSynced }), () => NOW);
  assert.match(byName(results, "Blockbook").servers[0].detail, /not in sync/);
});

test("a broadcast endpoint that accepts garbage is not trusted", async () => {
  const results = await runChecks(
    env,
    fakeFetch({ "POST https://bb1.example/api/v2/sendtx/": json({ result: "abc" }) }),
    () => NOW
  );
  assert.match(byName(results, "Blockbook").servers[0].detail, /broadcast endpoint/);
});

test("a backend that returns zeros fails like the app treats it", async () => {
  const zeros = json({ backendTipHeight: 0, difficulty: 0, tipHeight: 0 });
  const results = await runChecks(env, fakeFetch({ "GET https://be2.example/info": zeros }), () => NOW);
  const backends = byName(results, "OCM backends");
  assert.equal(backends.status, "degraded");
  assert.match(backends.servers[1].detail, /backendTipHeight is 0/);
});

test("a missing price fails that coin only", async () => {
  const results = await runChecks(
    env,
    fakeFetch({
      "GET https://api.coinpaprika.com/v1/tickers/vtc-vertcoin?quotes=BTC": json({ error: "id not found" }, 404)
    }),
    () => NOW
  );
  const prices = byName(results, "CoinPaprika");
  assert.equal(prices.status, "degraded");
  assert.equal(prices.servers.find((s) => !s.ok).label, "VTC");
});

test("a dead miner download is reported", async () => {
  const results = await runChecks(
    env,
    fakeFetch({ "HEAD https://dl.example/a/Miner.zip": new Response(null, { status: 404 }) }),
    () => NOW
  );
  const miners = byName(results, "Miner downloads");
  assert.equal(miners.status, "down");
  assert.equal(miners.servers.length, 1, "duplicate URLs are only checked once");
});

test("an unreachable miners.json is down", async () => {
  const results = await runChecks(
    env,
    fakeFetch({ "GET https://raw.example/miners.json": new Error("boom") }),
    () => NOW
  );
  assert.equal(byName(results, "Miner downloads").status, "down");
});

test("an empty server list is down, not ok", async () => {
  const results = await runChecks({ ...env, BLOCKBOOK_URLS: "" }, fakeFetch(), () => NOW);
  assert.equal(byName(results, "Blockbook").status, "down");
});
