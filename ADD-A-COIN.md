# How to add a coin

### Requirements

- Actively mined on [zpool.ca](https://www.zpool.ca/coins) or [HashCryptos.com](https://www.hashcryptos.com/)
- Listed on [CoinPaprika](https://coinpaprika.com/), which the app uses for prices
- Submit a PR formatted in the following format

The app loads [pools.json](pools.json) and [backends.json](backends.json) from this repository's master branch when it starts, so a merged coin shows up without a new release.

### What files must be edited?

In `pools.json`, add the coin to `payouts`:

```json
{"id": 29, "name": "Bitcoin", "ticker": "BTC"}
```

 - `id` is a new number that comes after the highest ID already in the list. IDs are saved in users' settings, so never reuse or change one
 - `name` is the name shown in the Payout dropdown menu
 - `ticker` is the ticker the pool uses for miner configuration
 - If the pool needs to know which chain to pay on, add `"networks"` with the network names, for example `"networks": ["ERC20", "TRC20", "BEP20"]`

Then add the ticker to the `payouts` list of each pool that pays out in it, in the order it should appear in the dropdown menu.

In `backends.json`, add the coin's CoinPaprika ID to `coinpaprikaIds`, in alphabetical order of ticker:

```json
"BTC": "btc-bitcoin"
```

### CoinPaprika API

The app reads the coin's price in BTC from CoinPaprika to show Expected Earnings in it. A coin without a CoinPaprika ID has no price, so its Expected Earnings show zero.

The CoinPaprika ID is the last part of the coin's page URL, for example `btc-bitcoin` in https://coinpaprika.com/coin/btc-bitcoin/. It must work in this URL: https://api.coinpaprika.com/v1/tickers/btc-bitcoin
