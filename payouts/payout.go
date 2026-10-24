package payouts

import (
	"fmt"

	"github.com/vertiond/verthash-one-click-miner/networks"
	"github.com/vertiond/verthash-one-click-miner/util"
)

type Payout interface {
	GetID() int
	GetDisplayName() string
	GetTicker() string
	GetNetworks() []string
}

// func GetPayouts(testnet bool) []Payout {
// 	if testnet {
// 		return []Payout{
// 			NewVTCPayout(),
// 		}
// 	}
// 	return []Payout{
// 		NewDOGEPayout(),
// 		NewVTCPayout(),
// 		NewBTCPayout(),
// 		NewBCHPayout(),
// 		NewDASHPayout(),
// 		NewDGBPayout(),
// 		NewETHPayout(),
// 		NewFIROPayout(),
// 		NewGRSPayout(),
// 		NewLTCPayout(),
// 		NewXMRPayout(),
// 		NewRVNPayout(),
// 	}
// }

// coinpaprikaIDs maps payout tickers to their CoinPaprika coin id. Payouts that
// are missing here have no price source and report a rate of zero.
var coinpaprikaIDs = map[string]string{
	"BCH":   "bch-bitcoin-cash",
	"BNB":   "bnb-binance-coin",
	"BTC":   "btc-bitcoin",
	"CAKE":  "cake-pancakeswap",
	"DASH":  "dash-dash",
	"DGB":   "dgb-digibyte",
	"DOGE":  "doge-dogecoin",
	"ETC":   "etc-ethereum-classic",
	"FIRO":  "firo-firo",
	"FLUX":  "zel-zelcash",
	"GRS":   "grs-groestlcoin",
	"KMD":   "kmd-komodo",
	"LBC":   "lbc-lbry-credits",
	"LTC":   "ltc-litecoin",
	"MATIC": "matic-polygon",
	"PPC":   "ppc-peercoin",
	"RTM":   "rtm-raptoreum",
	"RVN":   "rvn-ravencoin",
	"SHIB":  "shib-shiba-inu",
	"TRX":   "trx-tron",
	"USDT":  "usdt-tether",
	"VTC":   "vtc-vertcoin",
	"XMR":   "xmr-monero",
	"XVG":   "xvg-verge",
	"ZEC":   "zec-zcash",
	"ZEN":   "zen-horizen",
}

// DefaultCoinpaprikaURL is the CoinPaprika API base, without a trailing slash,
// used when neither backends.json nor its cached copy has one.
const DefaultCoinpaprikaURL = "https://api.coinpaprika.com/v1"

// coinpaprikaURL is the CoinPaprika API base, a variable so tests can point it elsewhere.
var coinpaprikaURL = DefaultCoinpaprikaURL

// SetCoinpaprika replaces the CoinPaprika API base (without a trailing slash)
// and adds or replaces coin ids by ticker. Empty values are ignored, so the
// built-in ones stay in use.
func SetCoinpaprika(url string, ids map[string]string) {
	if url != "" {
		coinpaprikaURL = url
	}
	if len(ids) == 0 {
		return
	}
	merged := make(map[string]string, len(coinpaprikaIDs)+len(ids))
	for ticker, id := range coinpaprikaIDs {
		merged[ticker] = id
	}
	for ticker, id := range ids {
		if ticker != "" && id != "" {
			merged[ticker] = id
		}
	}
	coinpaprikaIDs = merged
}

// GetBitcoinPerUnitCoin returns the price of one unit of the coin in BTC from
// CoinPaprika, or 0 when the coin is unknown or the lookup fails.
func GetBitcoinPerUnitCoin(coinTicker string) float64 {
	networks.WaitForEndpoints()
	id, ok := coinpaprikaIDs[coinTicker]
	if !ok {
		return 0.0
	}
	if coinTicker == "BTC" {
		return 1.0
	}

	payload := struct {
		Quotes map[string]struct {
			Price float64 `json:"price"`
		} `json:"quotes"`
	}{}
	err := util.GetJson(fmt.Sprintf("%s/tickers/%s?quotes=BTC", coinpaprikaURL, id), &payload)
	if err != nil {
		return 0.0
	}
	return payload.Quotes["BTC"].Price
}

//func GetBitcoinPerUnitCoin(coinID string, coinTicker string, exchange string) float64 {
//	jsonPayload := map[string]interface{}{}
//	err := util.GetJson(fmt.Sprintf(
//		"https://api.cryptonator.com/api/ticker/%s-btc",
//		strings.ToLower(coinTicker)),
//		&jsonPayload)
//	if err != nil {
//		return 0.0
//	}
//	jsonTicker, ok := jsonPayload["ticker"].(map[string]interface{})
//	if !ok {
//		return 0.0
//	}
//	jsonTickerPrice, ok := jsonTicker["price"].(string)
//	if !ok {
//		return 0.0
//	}
//	result, _ := strconv.ParseFloat(jsonTickerPrice, 64)
//	return result
//}
