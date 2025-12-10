package wallet

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/vertiond/verthash-one-click-miner/keyfile"
	"github.com/vertiond/verthash-one-click-miner/networks"
	"github.com/vertiond/verthash-one-click-miner/util"
)

// SignMyInputs finds the inputs in a transaction that came from our own wallet, and signs them with our private keys.
// Will modify the transaction in place, but will ignore inputs that we can't sign and leave them unsigned.
func (w *Wallet) SignMyInputs(tx *wire.MsgTx, password string) error {

	// For now using only P2PKH signing - since we generate
	// a legacy address. Will have to use segwit stuff at some point

	// generate tx-wide hashCache for segwit stuff
	// might not be needed (non-witness) but make it anyway
	// hCache := txscript.NewTxSigHashes(tx)

	// make the stashes for signatures / witnesses
	sigStash := make([][]byte, len(tx.TxIn))

	// get key
	privBytes, err := keyfile.LoadPrivateKey(password)
	if err != nil {
		return err
	}

	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), privBytes)

	for i := range tx.TxIn {
		sigStash[i], err = txscript.SignatureScript(tx, i, w.Script, txscript.SigHashAll, priv, true)
		if err != nil {
			return err
		}
	}
	// swap sigs into sigScripts in txins
	for i, txin := range tx.TxIn {
		if sigStash[i] != nil {
			txin.SignatureScript = sigStash[i]
		}
	}

	return nil
}

type txSendReply struct {
	TxId string `json:"txid"`
}

func (w *Wallet) Send(tx *wire.MsgTx) (string, error) {
	var b bytes.Buffer
	tx.Serialize(&b)
	txHex := hex.EncodeToString(b.Bytes())

	// CryptoAPIs requires { data: { item: {...} } } structure
	// Field name must be "signedTransactionHex" (not "transactionHex")
	sendPayload := map[string]interface{}{
		"context": "verthash-ocm",
		"data": map[string]interface{}{
			"item": map[string]interface{}{
				"signedTransactionHex": txHex,
			},
		},
	}

	url := fmt.Sprintf("%sbroadcast-transactions/dogecoin/mainnet", networks.Active.InsightURL)
	
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

	jsonPayload := map[string]interface{}{}
	var err error
	if len(headers) > 0 {
		err = util.PostJsonWithHeaders(url, sendPayload, headers, &jsonPayload)
	} else {
		err = util.PostJson(url, sendPayload, &jsonPayload)
	}
	
	// Try to parse the response even if there was an HTTP error
	// (PostJsonWithHeaders decodes the response even on error status codes)
	jsonData, ok := jsonPayload["data"].(map[string]interface{})
	if ok {
		jsonItem, ok := jsonData["item"].(map[string]interface{})
		if ok {
			txid, ok := jsonItem["transactionId"].(string)
			if ok {
				// Successfully parsed transactionId - transaction was sent
				return txid, nil
			}
		}
	}

	// If we couldn't parse the response, return the error
	if err != nil {
		return "", err
	}

	// No error but couldn't parse - return generic error
	return "", fmt.Errorf("failed to parse transaction response")
}
