package wallet

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
	"github.com/tidwall/buntdb"
	"github.com/vertiond/verthash-one-click-miner/logging"
	"github.com/vertiond/verthash-one-click-miner/networks"
	"github.com/vertiond/verthash-one-click-miner/util"
	"github.com/vertiond/verthash-one-click-miner/util/bech32"
)

type Wallet struct {
	Address   string
	Script    []byte
	Spendable uint64
	Maturing  uint64
	db        *buntdb.DB
}

type Utxo struct {
	TxID   string `json:"txid"`
	Vout   uint   `json:"vout"`
	Amount uint64 `json:"satoshis"`
}

// Insight's limit is 100kB HEX (so 50kB raw bytes) - limiting this to 45kB. Once we have
// better backends (an insight that performs better and allows higher limits, an integrated
// full node or just using Vertcoin Core) we can scale this up.
var maxTxSize = 45000

func NewWallet(addr string, script []byte) (*Wallet, error) {
	logging.Infof("Initializing wallet %s", addr)
	db, err := buntdb.Open(filepath.Join(util.DataDirectory(), networks.Active.WalletDB))
	if err != nil {
		return nil, err
	}
	return &Wallet{Address: addr, Script: script, db: db}, nil
}

// isAddressSynced checks if the address has already been synced (cached in database)
func (w *Wallet) isAddressSynced() bool {
	synced := false
	err := w.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(fmt.Sprintf("cryptoapis_sync_%s", w.Address))
		if err == nil && val == "1" {
			synced = true
		}
		return nil
	})
	if err != nil {
		logging.Debugf("Error checking sync status: %v", err)
	}
	return synced
}

// setAddressSynced saves the sync status to the database
func (w *Wallet) setAddressSynced(synced bool) {
	err := w.db.Update(func(tx *buntdb.Tx) error {
		value := "0"
		if synced {
			value = "1"
		}
		_, _, err := tx.Set(fmt.Sprintf("cryptoapis_sync_%s", w.Address), value, nil)
		return err
	})
	if err != nil {
		logging.Warnf("Error saving sync status: %v", err)
	}
}

// isAddressActivated checks if the address has already been activated (cached in database)
func (w *Wallet) isAddressActivated() bool {
	activated := false
	err := w.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(fmt.Sprintf("cryptoapis_activated_%s", w.Address))
		if err == nil && val == "1" {
			activated = true
		}
		return nil
	})
	if err != nil {
		logging.Debugf("Error checking activation status: %v", err)
	}
	return activated
}

// setAddressActivated saves the activation status to the database
func (w *Wallet) setAddressActivated(activated bool) {
	err := w.db.Update(func(tx *buntdb.Tx) error {
		value := "0"
		if activated {
			value = "1"
		}
		_, _, err := tx.Set(fmt.Sprintf("cryptoapis_activated_%s", w.Address), value, nil)
		return err
	})
	if err != nil {
		logging.Warnf("Error saving activation status: %v", err)
	}
}

// syncAddress synchronizes the wallet address with CryptoAPIs
// This must be called before fetching UTXOs to ensure historical data is available
// Returns true if sync is completed, false otherwise
// Uses database cache to avoid redundant API calls
func (w *Wallet) syncAddress() (bool, error) {
	// Check if already synced (cached)
	if w.isAddressSynced() {
		logging.Debugf("Address %s already synced (cached), skipping API call", w.Address)
		return true, nil
	}
	
	url := fmt.Sprintf("%saddresses-historical/manage/dogecoin/mainnet", networks.Active.InsightURL)
	
	// Prepare request payload
	// Note: callbackUrl is optional and should only be included if a valid URL is provided
	syncPayload := map[string]interface{}{
		"context": "verthash-ocm",
		"data": map[string]interface{}{
			"item": map[string]interface{}{
				"address": w.Address,
				// callbackUrl omitted - not needed for our use case
			},
		},
	}
	
	// Only add API key if calling CryptoAPIs directly (not through proxy)
	headers := make(map[string]string)
	if strings.Contains(networks.Active.InsightURL, "cryptoapis.io") {
		apiKey := util.GetCryptoAPIsKey()
		if apiKey != "" {
			headers["x-api-key"] = apiKey
		}
	}
	
	jsonPayload := map[string]interface{}{}
	var err error
	if len(headers) > 0 {
		err = util.PostJsonWithHeaders(url, syncPayload, headers, &jsonPayload)
	} else {
		err = util.PostJson(url, syncPayload, &jsonPayload)
	}
	
	// Check for "already_exists" error (HTTP 409) - this means address is already synced
	if err != nil {
		errStr := err.Error()
		// Check if it's HTTP 409 with "already_exists" error code
		if strings.Contains(errStr, "HTTP 409") {
			if errorObj, ok := jsonPayload["error"].(map[string]interface{}); ok {
				if errorCode, _ := errorObj["code"].(string); errorCode == "already_exists" {
					logging.Debugf("Address %s already synced (already_exists), treating as success", w.Address)
					w.setAddressSynced(true)
					return true, nil
				}
			}
		}
		logging.Warnf("Error syncing address with CryptoAPIs: %s", err.Error())
		return false, err
	}
	
	// Check for API errors in response (non-HTTP errors)
	if errorObj, ok := jsonPayload["error"].(map[string]interface{}); ok {
		errorCode, _ := errorObj["code"].(string)
		errorMsg, _ := errorObj["message"].(string)
		// "already_exists" is actually a success case
		if errorCode == "already_exists" {
			logging.Debugf("Address %s already synced (already_exists), treating as success", w.Address)
			w.setAddressSynced(true)
			return true, nil
		}
		errMsg := fmt.Sprintf("API error: %s - %s", errorCode, errorMsg)
		logging.Warnf("CryptoAPIs sync error: %s", errMsg)
		return false, fmt.Errorf(errMsg)
	}
	
	// Check sync status
	syncCompleted := false
	if jsonData, ok := jsonPayload["data"].(map[string]interface{}); ok {
		if jsonItem, ok := jsonData["item"].(map[string]interface{}); ok {
			if syncStatus, ok := jsonItem["syncStatus"].(string); ok {
				logging.Debugf("Address sync status: %s", syncStatus)
				syncCompleted = (syncStatus == "completed")
			}
		}
	}
	
	// Cache the sync status
	if syncCompleted {
		w.setAddressSynced(true)
	}
	
	return syncCompleted, nil
}

// activateAddress activates a previously synced address with CryptoAPIs
// This must be called after sync is completed to resume tracking
// Uses database cache to avoid redundant API calls
func (w *Wallet) activateAddress() error {
	// Check if already activated (cached)
	if w.isAddressActivated() {
		logging.Debugf("Address %s already activated (cached), skipping API call", w.Address)
		return nil
	}
	
	url := fmt.Sprintf("%saddresses-historical/manage/dogecoin/mainnet/%s/activate", networks.Active.InsightURL, w.Address)
	
	// Prepare request payload - CryptoAPIs requires { data: { item: {...} } } structure
	activatePayload := map[string]interface{}{
		"context": "verthash-ocm",
		"data": map[string]interface{}{
			"item": map[string]interface{}{
				// Empty item - activation doesn't require additional properties
			},
		},
	}
	
	// Only add API key if calling CryptoAPIs directly (not through proxy)
	headers := make(map[string]string)
	if strings.Contains(networks.Active.InsightURL, "cryptoapis.io") {
		apiKey := util.GetCryptoAPIsKey()
		if apiKey != "" {
			headers["x-api-key"] = apiKey
		}
	}
	
	jsonPayload := map[string]interface{}{}
	var err error
	if len(headers) > 0 {
		err = util.PostJsonWithHeaders(url, activatePayload, headers, &jsonPayload)
	} else {
		err = util.PostJson(url, activatePayload, &jsonPayload)
	}
	
	// Check for "already_exists" or "sync_address_already_active" - these mean address is already activated
	if err != nil {
		errStr := err.Error()
		// Check if it's HTTP 409 with "already_exists" or "sync_address_already_active"
		if strings.Contains(errStr, "HTTP 409") {
			if errorObj, ok := jsonPayload["error"].(map[string]interface{}); ok {
				errorCode, _ := errorObj["code"].(string)
				if errorCode == "already_exists" || errorCode == "sync_address_already_active" {
					logging.Debugf("Address %s already activated (%s), treating as success", w.Address, errorCode)
					w.setAddressActivated(true)
					return nil
				}
			}
		}
		logging.Warnf("Error activating address with CryptoAPIs: %s", err.Error())
		return err
	}
	
	// Check for API errors in response (non-HTTP errors)
	if errorObj, ok := jsonPayload["error"].(map[string]interface{}); ok {
		errorCode, _ := errorObj["code"].(string)
		errorMsg, _ := errorObj["message"].(string)
		// "already_exists" or "sync_address_already_active" are success cases
		if errorCode == "already_exists" || errorCode == "sync_address_already_active" {
			logging.Debugf("Address %s already activated (%s), treating as success", w.Address, errorCode)
			w.setAddressActivated(true)
			return nil
		}
		errMsg := fmt.Sprintf("API error: %s - %s", errorCode, errorMsg)
		logging.Warnf("CryptoAPIs activation error: %s", errMsg)
		return fmt.Errorf(errMsg)
	}
	
	// Check activation status
	activated := false
	if jsonData, ok := jsonPayload["data"].(map[string]interface{}); ok {
		if jsonItem, ok := jsonData["item"].(map[string]interface{}); ok {
			if isActive, ok := jsonItem["isActive"].(bool); ok {
				logging.Debugf("Address activation status: %v", isActive)
				activated = isActive
			}
			if syncStatus, ok := jsonItem["syncStatus"].(string); ok {
				logging.Debugf("Address sync status after activation: %s", syncStatus)
			}
		}
	}
	
	// Cache the activation status
	if activated {
		w.setAddressActivated(true)
	}
	
	return nil
}

func (w *Wallet) Utxos() ([]Utxo, error) {
	// Step 1: Sync address first to ensure historical data is available
	// "already_exists" (HTTP 409) is treated as success - address is already synced
	syncCompleted, err := w.syncAddress()
	if err != nil {
		logging.Warnf("Address sync failed, continuing anyway: %s", err.Error())
	}
	
	// Step 2: Always try to activate the address after sync (whether new sync or already exists)
	// This ensures the address is active and can be queried for UTXOs
	if syncCompleted {
		err = w.activateAddress()
		if err != nil {
			logging.Warnf("Address activation failed, continuing anyway: %s", err.Error())
		}
	} else {
		// Even if sync didn't complete, try activation in case address was already synced
		// This handles the case where sync is in progress but address needs activation
		err = w.activateAddress()
		if err != nil {
			logging.Debugf("Address activation attempted (sync not completed): %s", err.Error())
		}
	}
	
	utxos := []Utxo{}
	jsonPayload := map[string]interface{}{}
	url := fmt.Sprintf("%saddresses-historical/utxo/dogecoin/mainnet/%s/unspent-outputs", networks.Active.InsightURL, w.Address)
	
	// Only add API key if calling CryptoAPIs directly (not through proxy)
	// Default approach: backend proxy handles API key authentication
	headers := make(map[string]string)
	if strings.Contains(networks.Active.InsightURL, "cryptoapis.io") {
		// Direct CryptoAPIs call - user must provide their own API key
		apiKey := util.GetCryptoAPIsKey()
		if apiKey != "" {
			headers["x-api-key"] = apiKey
		}
	}
	// If using a proxy (default), proxy handles authentication - no API key needed here
	
	if len(headers) > 0 {
		err = util.GetJsonWithHeaders(url, headers, &jsonPayload)
	} else {
		err = util.GetJson(url, &jsonPayload)
	}
	json_parse_success := false
	if err == nil {
		jsonData, ok := jsonPayload["data"].(map[string]interface{})
		if ok {
			jsonDataItemsArr, ok := jsonData["items"].([]interface{})
			if ok {
				json_parse_success = true
				for _, jsonDataItemInfo := range jsonDataItemsArr {
					jsonDataItemInfoMap := jsonDataItemInfo.(map[string]interface{})
					utxo_txid, ok1 := jsonDataItemInfoMap["transactionId"].(string)
					utxo_vout, ok2 := jsonDataItemInfoMap["index"].(float64)
					valueObj, ok3 := jsonDataItemInfoMap["value"].(map[string]interface{})
					if !ok1 || !ok2 || !ok3 {
						json_parse_success = false
						break
					}
					tx_value_in_dogecoin_str, ok4 := valueObj["amount"].(string)
					if !ok4 {
						json_parse_success = false
						break
					}
					tx_value_in_dogecoin_float, _ := strconv.ParseFloat(tx_value_in_dogecoin_str, 64)
					utxo_amount := uint64(math.Round(tx_value_in_dogecoin_float * float64(100000000)))
					u := Utxo{utxo_txid, uint(utxo_vout), utxo_amount}
					utxos = append(utxos, u)
				}
			}
		}
	}
	if !json_parse_success {
		if err != nil {
			logging.Errorf("Error fetching UTXOs from DOGE Backend: %s", err.Error())
		} else {
			logging.Errorf("Error fetching UTXOs from DOGE Backend")
		}
		return utxos, err
	}
	return utxos, nil
}

func (w *Wallet) PrepareSweep(addr string) ([]*wire.MsgTx, error) {
	utxos, err := w.Utxos()
	if err != nil {
		return nil, errors.New("backend_failure")
	}
	retArr := make([]*wire.MsgTx, 0)
	for {
		tx := wire.NewMsgTx(2)
		totalIn := uint64(0)
		for _, u := range utxos {
			alreadyIncluded := false
			for _, t := range retArr {
				for _, i := range t.TxIn {
					if i.PreviousOutPoint.Hash.String() == u.TxID && i.PreviousOutPoint.Index == uint32(u.Vout) {
						alreadyIncluded = true
						break
					}
				}
			}
			if alreadyIncluded {
				logging.Debugf("UTXO Already Included: %v", u)
				continue
			}
			totalIn += u.Amount
			h, _ := chainhash.NewHashFromStr(u.TxID)
			tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(h, uint32(u.Vout)), w.Script, nil))
		}

		if len(tx.TxIn) == 0 {
			logging.Warnf("Trying to sweep with zero UTXOs")
			return nil, errors.New("insufficient_funds")
		}

		hash, version, err := base58.CheckDecode(addr)
		if err == nil && version == networks.Active.Base58P2PKHVersion {
			pubKeyHash := hash
			if err != nil {
				return nil, fmt.Errorf("invalid_address")
			}
			if len(pubKeyHash) != 20 {
				return nil, fmt.Errorf("invalid_address")
			}
			p2pkhScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_DUP).
				AddOp(txscript.OP_HASH160).AddData(pubKeyHash).
				AddOp(txscript.OP_EQUALVERIFY).AddOp(txscript.OP_CHECKSIG).Script()
			if err != nil {
				return nil, fmt.Errorf("script_failure")
			}
			tx.AddTxOut(wire.NewTxOut(0, p2pkhScript))
		} else if err == nil && version == networks.Active.Base58P2SHVersion {
			scriptHash := hash
			if err != nil {
				return nil, fmt.Errorf("invalid_address")
			}
			if len(scriptHash) != 20 {
				return nil, fmt.Errorf("invalid_address")
			}
			p2shScript, err := txscript.NewScriptBuilder().AddOp(txscript.OP_HASH160).AddData(scriptHash).AddOp(txscript.OP_EQUAL).Script()
			if err != nil {
				return nil, fmt.Errorf("script_failure")
			}
			tx.AddTxOut(wire.NewTxOut(0, p2shScript))
		} else if strings.HasPrefix(addr, fmt.Sprintf("%s1", networks.Active.Bech32Prefix)) {
			script, err := bech32.SegWitAddressDecode(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid_address")
			}
			tx.AddTxOut(wire.NewTxOut(int64(totalIn), script))
		} else {
			return nil, fmt.Errorf("invalid_address")
		}

		for i := range tx.TxIn {
			tx.TxIn[i].SignatureScript = make([]byte, 107) // add dummy signature to properly calculate size
		}

		// Weight = (stripped_size * 4) + witness_size formula,
		// using only serialization with and without witness data. As witness_size
		// is equal to total_size - stripped_size, this formula is identical to:
		// weight = (stripped_size * 3) + total_size.
		logging.Debugf("Transaction raw serialize size is %d\n", tx.SerializeSize())
		logging.Debugf("Transaction serialize size stripped is %d\n", tx.SerializeSizeStripped())

		chunked := false
		// Chunk if needed
		if tx.SerializeSize() > maxTxSize {
			chunked = true
			// Remove some extra inputs so we have enough for the next TX to remain valid, we
			// want to have enough money to create an output with enough value
			valueRemoved := uint64(0)
			for tx.SerializeSize() > maxTxSize || valueRemoved < 100000 {
				for _, u := range utxos {
					if u.TxID == tx.TxIn[len(tx.TxIn)-1].PreviousOutPoint.Hash.String() &&
						uint32(u.Vout) == tx.TxIn[len(tx.TxIn)-1].PreviousOutPoint.Index {
						totalIn -= u.Amount
						valueRemoved += u.Amount
					}
				}
				tx.TxIn = tx.TxIn[:len(tx.TxIn)-1]
			}
		}

		txWeight := (tx.SerializeSizeStripped() * 3) + tx.SerializeSize()
		logging.Debugf("Transaction weight is %d\n", txWeight)
		btcTx := btcutil.NewTx(tx)

		sigOpCost, err := w.GetSigOpCost(btcTx, w.Script, false, true, true)
		if err != nil {
			return nil, fmt.Errorf("could_not_calculate_fee")
		}
		logging.Debugf("Transaction sigop cost is %d\n", sigOpCost)

		vSize := (math.Max(float64(txWeight), float64(sigOpCost*20)) + float64(3)) / float64(4)
		logging.Debugf("Transaction vSize is %.4f\n", vSize)
		vSizeInt := uint64(vSize + float64(0.5)) // Round Up
		logging.Debugf("Transaction vSizeInt is %d\n", vSizeInt)

		// Vertcoin fee calculation //
		// fee := uint64(vSizeInt * 100)

		// Dogecoin fee calculation //
		// 0.01 DOGE fee per 1000 bytes
		// Base fee is 0 DOGE
		fee_doge := math.Max(float64(0), float64(0.01) * (float64(vSizeInt) / float64(1000)))
		// Do not send if total transaction amount is below soft dust limit of 0.01 DOGE
		if (totalIn - uint64(math.Ceil(fee_doge * 1000000))) < 1000000 { // UTXO Amount is in Satoshis
			return nil, fmt.Errorf("insufficient_funds")
		}
		fee := uint64(math.Ceil(fee_doge * float64(100000000))) // Convert fee from DOGE to Satoshis

		logging.Debugf("Setting fee to %d\n", fee)

		// empty out the dummy sigs
		for i := range tx.TxIn {
			tx.TxIn[i].SignatureScript = nil
		}

		tx.TxOut[0].Value = int64(totalIn - fee)
		if tx.TxOut[0].Value < 50000 {
			return nil, fmt.Errorf("insufficient_funds")
		}
		retArr = append(retArr, tx)

		if !chunked {
			break
		}
	}
	return retArr, nil
}

func DirectWPKHScriptFromPKH(pkh [20]byte) []byte {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_0).AddData(pkh[:])
	b, _ := builder.Script()
	return b
}

type BalanceResponse struct {
	Spendable uint64 `json:"confirmed"`
	Maturing  uint64 `json:"maturing"`
}

// Update will reload balance from the backend
func (w *Wallet) Update() {
	bal := BalanceResponse{}
	jsonPayload := map[string]interface{}{}
	url := fmt.Sprintf("%saddresses-latest/utxo/dogecoin/mainnet/%s/balance", networks.Active.InsightURL, w.Address)
	
	// Only add API key if calling CryptoAPIs directly (not through proxy)
	// Default approach: backend proxy handles API key authentication
	headers := make(map[string]string)
	if strings.Contains(networks.Active.InsightURL, "cryptoapis.io") {
		// Direct CryptoAPIs call - user must provide their own API key
		apiKey := util.GetCryptoAPIsKey()
		if apiKey != "" {
			headers["x-api-key"] = apiKey
		}
	}
	// If using a proxy (default), proxy handles authentication - no API key needed here
	
	var err error
	if len(headers) > 0 {
		err = util.GetJsonWithHeaders(url, headers, &jsonPayload)
	} else {
		err = util.GetJson(url, &jsonPayload)
	}
	
	json_parse_success := false
	if err == nil {
		// Log the response for debugging
		logging.Debugf("Balance API response: %+v", jsonPayload)
		
		jsonData, ok := jsonPayload["data"].(map[string]interface{})
		if ok {
			jsonItem, ok := jsonData["item"].(map[string]interface{})
			if ok {
				confirmedBalanceObj, ok := jsonItem["confirmedBalance"].(map[string]interface{})
				if ok {
					balance_confirmed_in_doge_str, ok1 := confirmedBalanceObj["amount"].(string)
					if ok1 {
						balance_confirmed_in_doge_float, _ := strconv.ParseFloat(balance_confirmed_in_doge_str, 64)
						balance_spendable := uint64(math.Round(balance_confirmed_in_doge_float * float64(100000000)))
						balance_maturing := uint64(0)
						bal = BalanceResponse{balance_spendable, balance_maturing}
						json_parse_success = true
					} else {
						logging.Debugf("Failed to extract amount from confirmedBalance: %+v", confirmedBalanceObj)
					}
				} else {
					logging.Debugf("Failed to extract confirmedBalance from item: %+v", jsonItem)
				}
			} else {
				logging.Debugf("Failed to extract item from data: %+v", jsonData)
			}
		} else {
			logging.Debugf("Failed to extract data from response: %+v", jsonPayload)
		}
	}
	if !json_parse_success {
		if err != nil {
			logging.Errorf("Error fetching balance from backend (URL: %s): %s", url, err.Error())
		} else {
			logging.Errorf("Error fetching balance from backend (URL: %s): failed to parse response", url)
		}
		return
	}
	w.Spendable = bal.Spendable
	w.Maturing = bal.Maturing
}

// GetBalance will scan the utxos in the wallet and return
// two values: mature and immature balance. Mining outputs
// need to wait for 101 confirmations before being allowed
// to spend
func (w *Wallet) GetBalance() (bal uint64, balImmature uint64) {
	return w.Spendable, w.Maturing
}
