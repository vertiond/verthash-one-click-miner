# How to add a coin

### Requirements

- Actively mined on [Zergpool.com](https://zergpool.com/site/exchangestatus) or [zpool.ca](https://www.zpool.ca/coins)
- BTC price available on an exchange found in [Coingecko's API](https://api.coingecko.com/api/v3/exchanges/)
- Submit a PR formatted in the following format

### What files must be created and edited?

In the Payouts folder, a new file titled `btcpayout.go` is created replacing btc with the ticker of the coin you are adding.
``````
package payouts

var _ Payout = &BTCPayout{}

type BTCPayout struct {}

func NewBTCPayout() *BTCPayout {
return &BTCPayout{}
}

func (p *BTCPayout) GetID() int {
return 2
}

func (p *BTCPayout) GetDisplayName() string {
return "Bitcoin"
}

func (p *BTCPayout) GetTicker() string {
return "BTC"
}

func (p *BTCPayout) GetCoingeckoExchange() string {
return "bittrex"
}

func (p *BTCPayout) GetCoingeckoCoinID() string {
return "bitcoin"
}
``````
 - Replace all instances of `BTC` in `BTCPayout` and `NewBTCPayout` with the ticker
 - Under `GetID` `return 2`, replace `2` with a new number that comes sequentially after the last coin added.  See IDs listed under other coins in the payouts folder
 - Under `GetDisplayName` `return "Bitcoin"`, replace `Bitcoin` with the name that you wish to be displayed in the Mining Pool dropdown menu
 - Under `GetTicker` `return "BTC"`, replace `BTC` with the ticker used by Zergpool and/or zpool for miner configuration
 - Under `GetCoingeckoExchange` `return "bittrex"`, replace `bittrex` with the exact name of an exchange as used within the Coingecko API
 - Under `GetCoingeckoCoinID` `return "bitcoin"`, replace `bitcoin` with the name of the CoinID as used within the Coingecko API

Finally, in the pools folder add the payout option in alphabetical order of GetDisplayName to zergpool.go and/or zpool.go under `return []payouts.Payout{`

### Coingecko API

This software pulls an exchange rate for a BTC pair from the Coingecko API using this format for example: https://api.coingecko.com/api/v3/exchanges/bittrex/tickers?coin_ids=dogecoin

`GetCoingeckoExchange` and `GetCoingeckoID` must match what would be in this URL
