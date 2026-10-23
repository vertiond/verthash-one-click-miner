import { runChecks, ALL_GROUPS } from "./checks.js";
import { planAlerts, formatAlert, formatHeartbeat } from "./alerts.js";

const STATE_KEY = "state";

// null when there is no KV namespace, raw is what is stored so a run can tell if anything changed
async function loadState(env) {
  if (!env.STATE) {
    return { raw: null, previous: null };
  }
  const raw = await env.STATE.get(STATE_KEY);
  return { raw, previous: raw ? JSON.parse(raw) : {} };
}

function parseMentions(env) {
  try {
    return JSON.parse(env.MENTIONS || "{}");
  } catch {
    return {};
  }
}

async function post(env, content, userIds = []) {
  const res = await fetch(env.DISCORD_WEBHOOK_URL, {
    method: "POST",
    headers: { "content-type": "application/json" },
    // Only the people named in the message are pinged
    body: JSON.stringify({ username: "Verthash OCM monitor", content, allowed_mentions: { users: userIds } })
  });
  if (!res.ok) {
    // Fails the run so it shows up in Cloudflare's cron logs
    throw new Error(`Discord answered HTTP ${res.status}`);
  }
}

// Every minute only the backends are checked, once a day (12:00 UTC) Blockbook and
// CoinPaprika. On Mondays the daily run also checks the backends to put them in the
// weekly "still running" summary, without counting that as one of their checks.
export const MINUTE_CRON = "* * * * *";
export const DAILY_CRON = "0 12 * * *";

export function plan(cron, scheduledTime) {
  if (cron === MINUTE_CRON) {
    return { check: ["backends"], count: ["backends"], summary: false };
  }
  const monday = new Date(scheduledTime || Date.now()).getUTCDay() === 1;
  return {
    check: monday ? ALL_GROUPS : ["blockbook", "prices"],
    count: ["blockbook", "prices"],
    summary: monday
  };
}

const GROUP_OF = (r) => (r.key === "blockbook" ? "blockbook" : r.key === "coinpaprika" ? "prices" : "backends");

export async function run(env, cron = DAILY_CRON, scheduledTime = Date.now()) {
  const scope = plan(cron, scheduledTime);
  const results = await runChecks(env, fetch, Date.now, scope.check);
  for (const r of results) {
    const timings = r.servers.map((s) => `${s.label}=${s.ok ? "ok" : "FAIL"} ${s.ms}ms`).join(", ");
    console.log(`${r.name}: ${r.status} (${timings})`, JSON.stringify(r.servers.filter((s) => !s.ok)));
  }

  const { raw, previous } = await loadState(env);
  const counted = results.filter((r) => scope.count.includes(GROUP_OF(r)));
  const outcome = planAlerts(counted, previous, 1);

  if (outcome.down.length > 0 || outcome.recovered.length > 0) {
    const mentions = parseMentions(env);
    const pinged = outcome.down.map((r) => mentions[r.key]).filter(Boolean);
    await post(env, formatAlert(outcome, results, mentions), pinged);
  } else if (scope.summary) {
    await post(env, formatHeartbeat(results));
  }

  // Keep what was remembered about the checks that didn't run this time. KV's free plan
  // allows 1000 writes a day, so only write when something changed.
  const next = JSON.stringify({ ...(previous || {}), ...outcome.state });
  if (env.STATE && next !== raw) {
    await env.STATE.put(STATE_KEY, next);
  }
  return { results, plan: outcome };
}

export default {
  async scheduled(event, env, ctx) {
    ctx.waitUntil(run(env, event.cron, event.scheduledTime));
  }
};
