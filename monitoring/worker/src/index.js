import { runChecks } from "./checks.js";
import { planAlerts, formatAlert, formatHeartbeat } from "./alerts.js";

const STATE_KEY = "state";
const HEARTBEAT_CRON = "0 12 * * 1";

async function loadState(env) {
  if (!env.STATE) {
    return null;
  }
  const stored = await env.STATE.get(STATE_KEY);
  return stored ? JSON.parse(stored) : {};
}

async function post(env, content) {
  const res = await fetch(env.DISCORD_WEBHOOK_URL, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ username: "Verthash OCM monitor", content })
  });
  if (!res.ok) {
    // Fails the run so it shows up in Cloudflare's cron logs
    throw new Error(`Discord answered HTTP ${res.status}`);
  }
}

export async function run(env, cron = "") {
  const results = await runChecks(env);
  for (const r of results) {
    console.log(`${r.name}: ${r.status}`, JSON.stringify(r.servers.filter((s) => !s.ok)));
  }

  const plan = planAlerts(results, await loadState(env));

  if (plan.down.length > 0 || plan.recovered.length > 0) {
    await post(env, formatAlert(plan, results));
  } else if (cron === HEARTBEAT_CRON) {
    await post(env, formatHeartbeat(results));
  }

  if (env.STATE) {
    await env.STATE.put(STATE_KEY, JSON.stringify(plan.state));
  }
  return { results, plan };
}

export default {
  async scheduled(event, env, ctx) {
    ctx.waitUntil(run(env, event.cron));
  }
};
