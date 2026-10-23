# Service monitor

A Cloudflare Worker that checks everything the miner depends on every 6 hours and
posts to Discord when a service has failed completely. It runs on Cloudflare's free
plan and doesn't depend on GitHub's schedule trigger, which is switched off after
60 days without repository activity.

## What it checks

| Category | Passes when | Alerts when |
|---|---|---|
| Blockbook (Dogecoin wallet) | each server is in sync, its last block is under 30 minutes old, the UTXO lookup answers, and the broadcast endpoint rejects an undecodable transaction | **every** server fails |
| OCM backends (network stats) | `/info` returns a non-zero tip height and difficulty, like the app requires | every backend fails |
| CoinPaprika (prices) | DOGE and VTC both have a BTC price | both fail |
| Miner downloads | every URL in `miners.json` on `master` answers a HEAD request | every one fails (or `miners.json` can't be read) |

A category that is only partly failing ("degraded") is listed in the message but doesn't
alert by itself, the app fails over to the next server. It sends one message when a
category goes down and one when it recovers, and a weekly "still running" summary on
Mondays at 12:00 UTC so a silent monitor can be told apart from a healthy one.

The broadcast check never sends a real transaction, it posts the single byte `00`,
which every node rejects.

## Setting it up

You need a Cloudflare account and a Discord webhook (channel settings, Integrations,
Webhooks). The webhook URL is a secret, enter it only in the command below.

```bash
cd monitoring/worker

# Optional but recommended: remembers the last state so it only alerts on changes.
# Paste the printed id into the commented [[kv_namespaces]] block in wrangler.toml.
npx wrangler@4.146.0 kv namespace create STATE

npx wrangler@4.146.0 secret put DISCORD_WEBHOOK_URL
npx wrangler@4.146.0 deploy
```

Without the KV namespace it still works, but repeats the alert on every 6 hour run
while something stays down.

## Trying it

```bash
npm test                                        # unit tests, no network needed
npx wrangler@4.146.0 dev --test-scheduled       # then: curl "http://localhost:8787/__scheduled?cron=0+12+*+*+1"
```

The `0 12 * * 1` cron sends the weekly summary, so that URL also shows you what a
message looks like.

## Keeping it in sync

`BLOCKBOOK_URLS` and `BACKEND_URLS` in `worker/wrangler.toml` must match
`networks/networks.go`. A Go test (`go test ./networks`) fails when they drift apart.
Change both together, then run `npx wrangler@4.146.0 deploy` again.
