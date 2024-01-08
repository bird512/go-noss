package main

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
)

func checkWallet(privateKey string) error {
	s, err := hex.DecodeString(privateKey)
	if err != nil {
		return fmt.Errorf("Sign called with invalid private key '%s': %w", privateKey, err)
	}

	sk, pk := btcec.PrivKeyFromBytes(s)
	pkBytes := pk.SerializeCompressed()
	pubKey := hex.EncodeToString(pkBytes[1:])
	fmt.Println(sk, pubKey)
	return nil
}

func newPrivateKey() {
	wallet, err := btcec.NewPrivateKey()
	if err != nil {

	}
	fmt.Printf("private: %s, public key: %s \n", wallet.Key, wallet.PubKey())
}
