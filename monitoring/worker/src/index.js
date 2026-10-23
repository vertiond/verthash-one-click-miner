import { runChecks } from "./checks.js";
import { planAlerts, formatAlert, formatHeartbeat } from "./alerts.js";

const STATE_KEY = "state";
// Only the weekly trigger sends the "still running" summary when nothing else is worth a
// message, keep this in sync with wrangler.toml
const SUMMARY_CRON = "0 12 * * 1";

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

export async function run(env, cron = "") {
  const results = await runChecks(env);
  for (const r of results) {
    const timings = r.servers.map((s) => `${s.label}=${s.ok ? "ok" : "FAIL"} ${s.ms}ms`).join(", ");
    console.log(`${r.name}: ${r.status} (${timings})`, JSON.stringify(r.servers.filter((s) => !s.ok)));
  }

  const { raw, previous } = await loadState(env);
  const threshold = Number(env.FAILURES_BEFORE_ALERT) || 1;
  const plan = planAlerts(results, previous, threshold);

  if (plan.down.length > 0 || plan.recovered.length > 0) {
    const mentions = parseMentions(env);
    const pinged = plan.down.map((r) => mentions[r.key]).filter(Boolean);
    await post(env, formatAlert(plan, results, mentions), pinged);
  } else if (cron === SUMMARY_CRON) {
    await post(env, formatHeartbeat(results));
  }

  // KV's free plan allows 1000 writes a day, so only write when something changed
  const next = JSON.stringify(plan.state);
  if (env.STATE && next !== raw) {
    await env.STATE.put(STATE_KEY, next);
  }
  return { results, plan };
}

export default {
  async scheduled(event, env, ctx) {
    ctx.waitUntil(run(env, event.cron));
  }
};
