package wallet

import (
	"bytes"
	"encoding/hex"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/vertiond/verthash-one-click-miner/keyfile"
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

func (w *Wallet) Send(tx *wire.MsgTx) (string, error) {
	var b bytes.Buffer
	tx.Serialize(&b)
	return blockbookSend(hex.EncodeToString(b.Bytes()))
}
