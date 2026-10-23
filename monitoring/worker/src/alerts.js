// Decides what is worth a Discord message. A category alerts when every one of its
// servers has failed ("down"), and again when it recovers. Partly failing
// ("degraded") categories only show up as detail, the app fails over on its own.

export function planAlerts(results, previous) {
  const state = {};
  const down = [];
  const recovered = [];
  for (const r of results) {
    state[r.name] = r.status;
    const was = previous ? previous[r.name] : undefined;
    // Without stored state every failing category alerts again on every run
    if (r.status === "down" && (previous === null || was !== "down")) {
      down.push(r);
    } else if (r.status !== "down" && was === "down") {
      recovered.push(r);
    }
  }
  return { state, down, recovered };
}

function describe(r) {
  const lines = r.servers.map((s) => (s.ok ? `✅ ${s.label}` : `❌ ${s.label}: ${s.detail}`));
  return lines.join("\n");
}

export function formatAlert({ down, recovered }, results) {
  const parts = [];
  for (const r of down) {
    parts.push(`🔴 **${r.name} is down**\n${describe(r)}`);
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
