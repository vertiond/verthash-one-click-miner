# Service monitor

A Cloudflare Worker that checks everything the miner depends on every 6 hours (00:00,
06:00, 12:00 and 18:00 UTC) and posts to Discord when a service has failed, pinging
whoever looks after it. It runs on Cloudflare's free plan and doesn't depend on GitHub's
schedule trigger, which is switched off after 60 days without repository activity.

## What it checks and who gets pinged

| Service | Passes when | Down when | Pings |
|---|---|---|---|
| Blockbook (Dogecoin wallet) | each server is in sync, its last block is under 30 minutes old, the UTXO lookup answers, and the broadcast endpoint rejects an undecodable transaction | **every** server fails | vertion |
| CoinPaprika (prices) | answers with a price, or says it is rate limited (see below) | errors, timeouts or no price | vertion |
| OCM backend `ocmbackend.javerity.com` | `/info` returns a non-zero tip height and difficulty, like the app requires | it fails | cruelnovo |
| OCM backend `ocmbackend.mindcraftblocks.com` | the same | it fails | numski |

Who is pinged is the `MENTIONS` setting in `worker/wrangler.toml`, a Discord user id per
service. A service with no entry still alerts, it just doesn't ping anyone. Only the people
named in a message can be pinged, nobody else gets notified.

## How double pings are avoided

The worker stores each service's last state in KV and only speaks when it changes.

- A service going **down** sends one message with its owner pinged.
- While it **stays down** nothing more is sent, however many runs go by. A backend that is
  down for weeks only pings once.
- Coming **back up** sends one message, without a ping.
- Only a partly failing Blockbook (one of two servers) shows up as extra detail in other
  messages, it never pings.
- The weekly "still running" summary on Mondays at 12:00 UTC is sent even when nothing
  changed, so a silent monitor can be told apart from a healthy one.

Without the KV namespace the worker has no memory and would repeat the ping on every run.

## Notes on specific checks

CoinPaprika allows 60 requests an hour per IP and Cloudflare's shared addresses use that up,
so it answers the Worker with HTTP 402 or 429. That still means the service is up, so only
real errors, timeouts and bad data count as down. Because of that, the check can't tell you
that prices are *usable* from Cloudflare, only that CoinPaprika is alive.

The broadcast check never sends a real transaction, it posts the single byte `00`,
which every node rejects.

## Setting it up

You need a Cloudflare account and a Discord webhook (channel settings, Integrations,
Webhooks). The webhook URL is a secret, enter it only in the command below.

```bash
cd monitoring/worker

# Optional but recommended: remembers the last state so it only alerts on changes.
# Paste the printed id into the commented [[kv_namespaces]] block in wrangler.toml.
npx wrangler@4.152.0 kv namespace create STATE

npx wrangler@4.152.0 secret put DISCORD_WEBHOOK_URL
npx wrangler@4.152.0 deploy
```

Without the KV namespace it still works, but repeats the alert on every 6 hour run
while something stays down.

## Trying it

```bash
npm test                                        # unit tests, no network needed
npx wrangler@4.152.0 dev --test-scheduled       # then: curl "http://localhost:8787/__scheduled?cron=0+12+*+*+1"
```

The `0 12 * * 1` cron sends the weekly summary, so that URL also shows you what a
message looks like.

## Keeping it in sync

`BLOCKBOOK_URLS` and `BACKEND_URLS` in `worker/wrangler.toml` must match
`networks/networks.go` (and the host of each backend needs its own `MENTIONS` entry). A Go test (`go test ./networks`) fails when they drift apart.
Change both together, then run `npx wrangler@4.152.0 deploy` again.
