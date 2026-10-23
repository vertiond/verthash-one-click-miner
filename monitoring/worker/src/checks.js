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
  try {
    await run();
    return { label, ok: true, detail: "" };
  } catch (err) {
    return { label, ok: false, detail: err && err.message ? err.message : String(err) };
  }
}

// ok: every server passes. degraded: some fail. down: all of them fail.
function category(name, servers) {
  if (servers.length === 0) {
    return { name, status: "down", servers: [{ label: "-", ok: false, detail: "nothing is configured to check" }] };
  }
  const failed = servers.filter((s) => !s.ok).length;
  const status = failed === 0 ? "ok" : failed === servers.length ? "down" : "degraded";
  return { name, status, servers };
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

async function checkPrice(fetchFn, id) {
  const data = await getJson(fetchFn, `https://api.coinpaprika.com/v1/tickers/${id}?quotes=BTC`);
  const price = data && data.quotes && data.quotes.BTC && data.quotes.BTC.price;
  if (!(price > 0)) {
    throw new Error("no BTC price");
  }
}

async function checkMinerDownloads(fetchFn, minersJsonUrl) {
  let miners;
  try {
    miners = await getJson(fetchFn, minersJsonUrl);
  } catch (err) {
    return [{ label: "miners.json", ok: false, detail: err.message }];
  }
  const urls = [...new Set(miners.map((m) => m.url).filter(Boolean))];
  return Promise.all(
    urls.map((url) =>
      probe(url.split("/").slice(-2).join("/"), async () => {
        const res = await request(fetchFn, url, { method: "HEAD", redirect: "follow" });
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`);
        }
      })
    )
  );
}

export async function runChecks(env, fetchFn = fetch, now = Date.now) {
  const address = env.PROBE_ADDRESS;

  const [blockbook, backends, prices, miners] = await Promise.all([
    Promise.all(
      list(env.BLOCKBOOK_URLS).map((url) =>
        probe(hostOf(url), () => checkBlockbookServer(fetchFn, url, address, now))
      )
    ),
    Promise.all(list(env.BACKEND_URLS).map((url) => probe(hostOf(url), () => checkBackendServer(fetchFn, url)))),
    Promise.all(
      [
        ["DOGE", "doge-dogecoin"],
        ["VTC", "vtc-vertcoin"]
      ].map(([ticker, id]) => probe(ticker, () => checkPrice(fetchFn, id)))
    ),
    checkMinerDownloads(fetchFn, env.MINERS_JSON_URL)
  ]);

  return [
    category("Blockbook (Dogecoin wallet)", blockbook),
    category("OCM backends (network stats)", backends),
    category("CoinPaprika (prices)", prices),
    category("Miner downloads", miners)
  ];
}
