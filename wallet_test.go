package main

import (
	"testing"
)

func TestCheckWallet(t *testing.T) {
	// Add your test cases here
}

func TestNewPrivateKey(t *testing.T) {
	// Add your test cases here
	generateWalletsToFile(2, "test.json")
	loadWalletFromFile("test.json")
}
