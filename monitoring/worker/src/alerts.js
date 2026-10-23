// Decides what is worth a Discord message. A category alerts when every one of its
// servers has failed ("down"), and again when it recovers. Partly failing
// ("degraded") categories only show up as detail, the app fails over on its own.

// What is remembered about a category between runs: how many checks in a row it has failed
// (counting stops at the threshold) and whether its owner was already told.
function entryOf(stored, threshold) {
  if (stored && typeof stored === "object") {
    return { failures: stored.failures || 0, alerted: stored.alerted === true };
  }
  // The format of the first version stored just the status
  return stored === "down" ? { failures: threshold, alerted: true } : { failures: 0, alerted: false };
}

// threshold: how many failed checks in a row it takes to alert. A category that
// recovers in between starts counting again from zero.
export function planAlerts(results, previous, threshold = 1) {
  const state = {};
  const down = [];
  const recovered = [];
  for (const r of results) {
    const was = entryOf(previous ? previous[r.name] : undefined, threshold);
    let failures = 0;
    let alerted = false;
    if (r.status === "down") {
      failures = Math.min(was.failures + 1, threshold);
      alerted = was.alerted;
      // Without stored state there is nothing to count with, alert on every failure
      if (previous === null || (!was.alerted && failures >= threshold)) {
        down.push({ ...r, failures });
        alerted = true;
      }
    } else if (was.alerted) {
      recovered.push(r);
    }
    state[r.name] = { failures, alerted };
  }
  return { state, down, recovered };
}

function describe(r) {
  const lines = r.servers.map((s) => (s.ok ? `✅ ${s.label}` : `❌ ${s.label}: ${s.detail}`));
  return lines.join("\n");
}

// Who to ping for a category, mentions maps its key to a Discord user id
export function mentionFor(r, mentions) {
  const id = mentions && mentions[r.key];
  return id ? `<@${id}>` : "";
}

export function formatAlert({ down, recovered }, results, mentions = {}) {
  const parts = [];
  for (const r of down) {
    const title = `🔴 **${r.name} is down** ${mentionFor(r, mentions)}`.trimEnd();
    const streak = r.failures > 1 ? ` (failed ${r.failures} checks in a row)` : "";
    parts.push(`${title}${streak}\n${describe(r)}`);
  }
  for (const r of recovered) {
    parts.push(`🟢 **${r.name} recovered**`);
  }
  const degraded = results.filter((r) => r.status === "degraded");
  if (degraded.length > 0) {
    parts.push(`🟡 Partly failing, the app is failing over:\n${degraded.map(describe).join("\n")}`);
  }
  return truncate(parts.join("\n\n"));
}

export function formatHeartbeat(results) {
  const icon = { ok: "🟢", degraded: "🟡", down: "🔴" };
  const lines = results.map((r) => `${icon[r.status]} ${r.name}: ${r.status}`);
  return truncate(`📋 Weekly monitor check, still running.\n${lines.join("\n")}`);
}

// Discord rejects messages over 2000 characters
function truncate(text) {
  return text.length <= 2000 ? text : `${text.slice(0, 1990)}\n…`;
}
