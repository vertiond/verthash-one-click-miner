# Service monitor

A Cloudflare Worker that checks everything the miner depends on and posts to Discord when a
service has failed, pinging whoever looks after it. It runs on Cloudflare's free plan and
doesn't depend on GitHub's schedule trigger, which is switched off after 60 days without
repository activity.

## What it checks, how often, and who gets pinged

| Service | Checked | Passes when | Alerts after | Pings |
|---|---|---|---|---|
| Blockbook (Dogecoin wallet) | once a day, 12:00 UTC | each server is in sync, its last block is under 30 minutes old, the UTXO lookup answers, and the broadcast endpoint rejects an undecodable transaction. Down only when **every** server fails | 2 failed days in a row (48 hours) | vertion |
| CoinPaprika (prices) | once a day, 12:00 UTC | answers with a price, or says it is rate limited (see below) | 2 failed days in a row (48 hours) | vertion |
| OCM backend `ocmbackend.javerity.com` | every minute | `/info` returns a non-zero tip height and difficulty, like the app requires | 5 failed minutes in a row | cruelnovo |
| OCM backend `ocmbackend.mindcraftblocks.com` | every minute | the same | 5 failed minutes in a row | numski |

One successful check starts a service's count over, so failures only add up when they are
consecutive. The limits are `FAILURES_BEFORE_ALERT` (backends) and
`DAILY_FAILURES_BEFORE_ALERT` (Blockbook and CoinPaprika) in `worker/wrangler.toml`.

Blockbook is a free public service, so it is only asked once a day. Checking it every minute
would send each server about 4,300 requests and 98 MB a day, and it also used up most of the
Worker's 10 ms CPU allowance per run.

Who is pinged is the `MENTIONS` setting in `worker/wrangler.toml`, a Discord user id per
service. A service with no entry still alerts, it just doesn't ping anyone. Only the people
named in a message can be pinged, nobody else gets notified.

## How double pings are avoided

The worker stores each service's failure count and whether it already alerted in KV, and only
speaks when that changes.

- A service going **down** (enough failed checks in a row) sends one message with its owner
  pinged.
- While it **stays down** nothing more is sent, however many runs go by. A backend that is
  down for weeks only pings once.
- Coming **back up** sends one message, without a ping. A blip that never reached the limit
  recovers silently.
- Only a partly failing Blockbook (one of two servers) shows up as extra detail in other
  messages, it never pings.
- The minute checks and the daily checks each keep the other's state, they don't overwrite it.
- State is only written to KV when something changed, the free plan allows 1,000 writes a day.
- The weekly "still running" summary is sent on Mondays with the 12:00 UTC run, even when
  nothing changed, so a silent monitor can be told apart from a healthy one. That run also
  checks the backends for the summary, without counting it as one of their checks.

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
npm test                                          # unit tests, no network needed
npx wrangler@4.152.0 dev --remote --test-scheduled
curl "http://localhost:8787/__scheduled?cron=0+12+*+*+*"   # the daily run
curl "http://localhost:8787/__scheduled?cron=*+*+*+*+*"    # the every minute run
```

`--remote` runs the Worker on Cloudflare, which is the only way to see how the services
answer from Cloudflare's network (CoinPaprika behaves differently there). It uses the real KV
namespace, so a failing service counts toward its alert limit. The weekly summary only
goes out when the daily run happens on a Monday.

## Keeping it in sync

`BLOCKBOOK_URLS` and `BACKEND_URLS` in `worker/wrangler.toml` must match
`networks/networks.go` (and the host of each backend needs its own `MENTIONS` entry). A Go test (`go test ./networks`) fails when they drift apart.
Change both together, then run `npx wrangler@4.152.0 deploy` again.
