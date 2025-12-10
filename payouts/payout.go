package payouts

import (
	"fmt"
	"strconv"

	"github.com/vertiond/verthash-one-click-miner/util"
)

type Payout interface {
	GetID() int
	GetDisplayName() string
	GetTicker() string
	GetCoingeckoExchange() string
	GetCoingeckoCoinID() string
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

func GetBitcoinPerUnitCoin(coinID string, coinTicker string, exchange string) float64 {
	// Use CoinEx API (free, no API key required)
	// coinTicker is the base coin (e.g., "DOGE"), append "BTC" to form market symbol (e.g., "DOGEBTC")
	market := coinTicker + "BTC"
	url := fmt.Sprintf("https://api.coinex.com/v2/spot/ticker?market=%s", market)
	
	jsonPayload := map[string]interface{}{}
	err := util.GetJson(url, &jsonPayload)
	
	if err != nil {
		return 0.0
	}

	// Check if code is 0 (success)
	code, ok := jsonPayload["code"].(float64)
	if !ok || code != 0 {
		return 0.0
	}

	// Get data array
	jsonDataArr, ok := jsonPayload["data"].([]interface{})
	if !ok || len(jsonDataArr) == 0 {
		return 0.0
	}

	// Get first item from data array
	jsonDataItem, ok := jsonDataArr[0].(map[string]interface{})
	if !ok {
		return 0.0
	}

	// Extract "last" value
	jsonLast, ok := jsonDataItem["last"].(string)
	if ok {
		result, err := strconv.ParseFloat(jsonLast, 64)
		if err == nil {
			return result
		}
	}

	return 0.0

	/* COMMENTED OUT: freecryptoapi.com implementation (backup in case CoinEx stops working)
	// Use InsightURL from networks (points to Cloudflare Worker that handles both APIs)
	// InsightURL is the proxy URL that routes to both CryptoAPIs and freecryptoapi.com
	// The proxy handles API key authentication - no API key needed here
	// Note: DOGE (DOGEBTC) now uses the same freecryptoapi.com API as other coins
	baseURL := strings.TrimSuffix(networks.Active.InsightURL, "/")
	url := fmt.Sprintf("%s/v1/getExchange?exchange=%s", baseURL, exchange)
	
	jsonPayload := map[string]interface{}{}
	err := util.GetJson(url, &jsonPayload)
	
	if err != nil {
		return 0.0
	}

	// Check if status is success
	status, ok := jsonPayload["status"].(string)
	if !ok || status != "success" {
		return 0.0
	}

	// Get symbols array
	jsonSymbolsArr, ok := jsonPayload["symbols"].([]interface{})
	if !ok {
		return 0.0
	}

	// Find the symbol matching coinTicker (e.g., "DOGEBTC")
	for _, jsonSymbolInfo := range jsonSymbolsArr {
		jsonSymbolInfoMap := jsonSymbolInfo.(map[string]interface{})
		jsonSymbol, ok := jsonSymbolInfoMap["symbol"].(string)
		if !ok {
			continue
		}
		
		// Match the symbol (e.g., "DOGEBTC")
		if jsonSymbol == coinTicker {
			jsonLast, ok := jsonSymbolInfoMap["last"].(string)
			if ok {
				result, err := strconv.ParseFloat(jsonLast, 64)
				if err == nil {
					return result
				}
			}
		}
	}

	return 0.0
	*/
}

/* COMMENTED OUT: Old SoChain API implementation for DOGE (backup in case CoinEx stops working)
func GetBitcoinPerUnitDOGE() float64 {
	jsonPayload := map[string]interface{}{}
	err := util.GetJson("https://sochain.com/api/v2/get_price/DOGE/BTC", &jsonPayload)
	if err != nil {
		return 0.0
	}
	jsonData, ok := jsonPayload["data"].(map[string]interface{})
	if !ok {
		return 0.0
	}
	jsonPriceArr, ok := jsonData["prices"].([]interface{})
	if !ok {
		return 0.0
	}

	result := 0.0
	for _, jsonPriceInfo := range jsonPriceArr {
		jsonPriceInfoMap := jsonPriceInfo.(map[string]interface{})
		jsonPriceBase, ok := jsonPriceInfoMap["price_base"]
		if !ok {
			continue
		}
		// Could pull from Bittrex or Binance at any given time,
		// whichever one happens to be listed first.
		// Doesn't matter which, as long as "price_base" is "BTC".
		if jsonPriceBase == "BTC" {
			jsonExchangePrice, ok := jsonPriceInfoMap["price"].(string)
			if ok {
				result, _ = strconv.ParseFloat(jsonExchangePrice, 64)
			}
			break
		}
	}
	return result
}
*/

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
