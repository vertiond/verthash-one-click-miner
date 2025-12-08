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

type txSend struct {
	RawTx string `json:"tx_hex"`
}

type txSendReply struct {
	TxId string `json:"txid"`
}

func (w *Wallet) Send(tx *wire.MsgTx) (string, error) {
	var b bytes.Buffer
	tx.Serialize(&b)
	s := txSend{
		RawTx: hex.EncodeToString(b.Bytes()),
	}

	r := txSendReply{}

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
		err = util.PostJsonWithHeaders(url, s, headers, &jsonPayload)
	} else {
		err = util.PostJson(url, s, &jsonPayload)
	}
	
	json_parse_success := false
	if err == nil {
		jsonData, ok := jsonPayload["data"].(map[string]interface{})
		if ok {
			jsonItem, ok := jsonData["item"].(map[string]interface{})
			if ok {
				txid, ok := jsonItem["transactionId"].(string)
				if ok {
					r = txSendReply{txid}
					json_parse_success = true
				}
			}
		}
	}

	if !json_parse_success {
		return "", err
	}

	return r.TxId, err
}
