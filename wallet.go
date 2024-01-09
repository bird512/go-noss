package main

import (
	"encoding/json"
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"os"
)

type Wallet struct {
	PrivateKey string
	PublicKey  string
	PublicNpub string
}

func checkWallet(privateKey string, publicKey string) error {
	pk, _ := nostr.GetPublicKey(privateKey)
	//nsec, _ := nip19.EncodePrivateKey(sk)
	npub, _ := nip19.EncodePublicKey(pk)
	fmt.Println("loaded public key is:", pk, "npub is:", npub)

	if pk != publicKey {
		return fmt.Errorf("public key is not valid, the recoved public key is %s", pk)
	}
	return nil
}

func generateWalletsToFile(count uint, filename string) {
	f, err := os.Create(filename)
	var wallets []Wallet

	defer f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	var i uint = 0
	for i < count {
		sk := nostr.GeneratePrivateKey()
		pk, _ := nostr.GetPublicKey(sk)
		npub, _ := nip19.EncodePublicKey(pk)
		w := Wallet{
			PrivateKey: sk,
			PublicKey:  pk,
			PublicNpub: npub,
		}
		i++
		// save with json
		wallets = append(wallets, w)
	}
	json.NewEncoder(f).Encode(wallets)
}

func loadWalletFromFile(file string) (w []Wallet) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = json.NewDecoder(f).Decode(&w)
	if err != nil {
		fmt.Println(err)
		return
	}
	return

}
