package wallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vertiond/verthash-one-click-miner/logging"
	"github.com/vertiond/verthash-one-click-miner/networks"
)

// Dogecoin coinbase outputs can only be spent after this many confirmations
const coinbaseMaturity = 240

var blockbookClient = &http.Client{Timeout: 30 * time.Second}

type blockbookUtxo struct {
	TxID          string `json:"txid"`
	Vout          uint   `json:"vout"`
	Value         string `json:"value"`
	Confirmations int64  `json:"confirmations"`
	Coinbase      bool   `json:"coinbase"`
}

type blockbookError struct {
	Error string `json:"error"`
}

// blockbookRequest tries each configured Blockbook server in turn and returns the
// first successful response body.
func blockbookRequest(method, path string, body []byte) ([]byte, error) {
	var lastErr error
	for _, base := range networks.Active.BlockbookURLs {
		url := strings.TrimRight(base, "/") + "/" + path
		req, err := http.NewRequest(method, url, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		r, err := blockbookClient.Do(req)
		if err != nil {
			logging.Warnf("Blockbook %s failed: %s", base, err.Error())
			lastErr = err
			continue
		}
		b, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if r.StatusCode != http.StatusOK {
			be := blockbookError{}
			json.Unmarshal(b, &be)
			lastErr = fmt.Errorf("blockbook %s returned %d: %s", base, r.StatusCode, be.Error)
			logging.Warnf("%s", lastErr.Error())
			// A rejected transaction is not a server problem, don't retry elsewhere
			if method == http.MethodPost && r.StatusCode == http.StatusBadRequest {
				return nil, lastErr
			}
			continue
		}
		return b, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no blockbook servers configured")
	}
	return nil, lastErr
}

// parseSatoshis parses an integer amount in satoshis as returned by Blockbook
func parseSatoshis(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

// blockbookUtxos returns all UTXOs of the address, split in spendable and immature
func blockbookUtxos(addr string) (spendable []Utxo, immature uint64, err error) {
	b, err := blockbookRequest(http.MethodGet, "api/v2/utxo/"+addr, nil)
	if err != nil {
		return nil, 0, err
	}
	raw := []blockbookUtxo{}
	if err = json.Unmarshal(b, &raw); err != nil {
		return nil, 0, err
	}
	for _, u := range raw {
		v, err := parseSatoshis(u.Value)
		if err != nil {
			return nil, 0, err
		}
		if u.Coinbase && u.Confirmations < coinbaseMaturity {
			immature += v
			continue
		}
		spendable = append(spendable, Utxo{u.TxID, u.Vout, v})
	}
	return spendable, immature, nil
}

func blockbookSend(rawHex string) (string, error) {
	b, err := blockbookRequest(http.MethodPost, "api/v2/sendtx/", []byte(rawHex))
	if err != nil {
		return "", err
	}
	reply := struct {
		Result string `json:"result"`
	}{}
	if err = json.Unmarshal(b, &reply); err != nil || reply.Result == "" {
		return "", fmt.Errorf("unexpected sendtx response: %s", string(b))
	}
	return reply.Result, nil
}
