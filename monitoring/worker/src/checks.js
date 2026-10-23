// Health checks for everything the miner depends on. Every check takes the fetch
// function as a parameter so the tests can run against fake servers.

const TIMEOUT_MS = 10000;
const MAX_BLOCK_AGE_MS = 30 * 60 * 1000;

async function request(fetchFn, url, options = {}) {
  return fetchFn(url, { ...options, signal: AbortSignal.timeout(TIMEOUT_MS) });
}

async function getJson(fetchFn, url) {
  const res = await request(fetchFn, url);
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }
  return res.json();
}

function hostOf(url) {
  try {
    return new URL(url).host;
  } catch {
    return url;
  }
}

// A server passes when run() resolves, and fails with the thrown message
async function probe(label, run) {
  const started = Date.now();
  try {
    await run();
    return { label, ok: true, detail: "", ms: Date.now() - started };
  } catch (err) {
    return { label, ok: false, detail: err && err.message ? err.message : String(err), ms: Date.now() - started };
  }
}

// ok: every server passes. degraded: some fail. down: all of them fail.
// key: what the MENTIONS setting uses to decide who to ping
function category(name, key, servers) {
  if (servers.length === 0) {
    return { name, key, status: "down", servers: [{ label: "-", ok: false, detail: "nothing is configured to check" }] };
  }
  const failed = servers.filter((s) => !s.ok).length;
  const status = failed === 0 ? "ok" : failed === servers.length ? "down" : "degraded";
  return { name, key, status, servers };
}

function list(value) {
  return (value || "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

async function checkBlockbookServer(fetchFn, base, address, now) {
  const root = base.replace(/\/+$/, "");

  const status = await getJson(fetchFn, `${root}/api/v2`);
  if (!status.blockbook || status.blockbook.inSync !== true) {
    throw new Error("not in sync with the chain");
  }
  const age = now() - Date.parse(status.blockbook.lastBlockTime);
  if (!(age < MAX_BLOCK_AGE_MS)) {
    throw new Error(`last block is ${Math.round(age / 60000)} minutes old`);
  }

  const utxos = await getJson(fetchFn, `${root}/api/v2/utxo/${address}`);
  if (!Array.isArray(utxos)) {
    throw new Error("UTXO lookup did not return a list");
  }

  // An undecodable transaction proves the broadcast endpoint reaches the node
  // without ever sending anything
  const res = await request(fetchFn, `${root}/api/v2/sendtx/`, { method: "POST", body: "00" });
  let body = null;
  try {
    body = await res.json();
  } catch {
    // reported below
  }
  if (res.status !== 400 || !body || typeof body.error !== "string") {
    throw new Error(`broadcast endpoint answered HTTP ${res.status} instead of rejecting a bad transaction`);
  }
}

async function checkBackendServer(fetchFn, base) {
  const info = await getJson(fetchFn, `${base.replace(/\/+$/, "")}/info`);
  for (const field of ["backendTipHeight", "tipHeight", "difficulty"]) {
    if (!info[field]) {
      throw new Error(`${field} is ${info[field]}`);
    }
  }
}

// CoinPaprika limits every IP to 60 requests an hour and Cloudflare's shared addresses
// hit that all the time, it answers HTTP 402 or 429. A rate limited answer still means
// the service is up, only timeouts, errors and bad data count as down.
async function checkPrices(fetchFn) {
  const res = await request(fetchFn, "https://api.coinpaprika.com/v1/tickers/doge-dogecoin?quotes=BTC");
  if (res.status === 402 || res.status === 429) {
    return;
  }
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }
  const data = await res.json();
  const price = data && data.quotes && data.quotes.BTC && data.quotes.BTC.price;
  if (!(price > 0)) {
    throw new Error("no BTC price");
  }
}

// groups: which checks to run, "blockbook", "prices" and "backends". The services are
// checked at different rates, each category says how many failed checks in a row it
// takes to alert (alertAfter): backends use FAILURES_BEFORE_ALERT, Blockbook and
// CoinPaprika use DAILY_FAILURES_BEFORE_ALERT.
export const ALL_GROUPS = ["blockbook", "prices", "backends"];

export async function runChecks(env, fetchFn = fetch, now = Date.now, groups = ALL_GROUPS) {
  const address = env.PROBE_ADDRESS;
  const backendAlertAfter = Number(env.FAILURES_BEFORE_ALERT) || 1;
  const dailyAlertAfter = Number(env.DAILY_FAILURES_BEFORE_ALERT) || 1;
  const results = [];

  const blockbook = groups.includes("blockbook")
    ? Promise.all(
        list(env.BLOCKBOOK_URLS).map((url) =>
          probe(hostOf(url), () => checkBlockbookServer(fetchFn, url, address, now))
        )
      )
    : null;
  const prices = groups.includes("prices") ? probe("api.coinpaprika.com", () => checkPrices(fetchFn)) : null;
  const backends = groups.includes("backends")
    ? Promise.all(list(env.BACKEND_URLS).map((url) => probe(hostOf(url), () => checkBackendServer(fetchFn, url))))
    : null;

  if (blockbook) {
    // Down only when every server fails, one is enough to keep the wallet working
    results.push({ ...category("Blockbook (Dogecoin wallet)", "blockbook", await blockbook), alertAfter: dailyAlertAfter });
  }
  if (prices) {
    results.push({ ...category("CoinPaprika (prices)", "coinpaprika", [await prices]), alertAfter: dailyAlertAfter });
  }
  if (backends) {
    // Each backend is its own category, they have different owners
    for (const b of await backends) {
      results.push({ ...category(`OCM backend ${b.label}`, b.label, [b]), alertAfter: backendAlertAfter });
    }
  }
  return results;
}
