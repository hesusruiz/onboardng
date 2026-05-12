package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/hesusruiz/onboardng/internal/crypto"
)

func main() {
	// hexKey := "0x0826f120c769d4de55ad686b31c60242d87615a2bb25f53c07b580d9e7a074af"
	hexKey := "0xda688a0f68d777cc0a65b86f468fd7f2ddb4f995d4a5e3bc66610bb6a128c090"

	// Strip any '0x' or '0X' prefix from the key and decode it
	hexKey = strings.TrimSpace(hexKey)
	hexKey = strings.TrimPrefix(hexKey, "0x")
	hexKey = strings.TrimPrefix(hexKey, "0X")

	// Get the private key raw key bytes
	dBytes, _ := hex.DecodeString(hexKey)

	// Import the key
	privECDSA, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), dBytes)
	if err != nil {
		panic(err)
	}

	// Get the did:key from the private key
	didKeyDerived, err := crypto.DeriveDidKeyFromPrivateKey(privECDSA)
	if err != nil {
		panic(err)
	}
	fmt.Println("DERIVED DID:KEY:")
	fmt.Println(didKeyDerived)
	fmt.Println()

	// PEM (PKCS#8)
	derBytes, _ := x509.MarshalPKCS8PrivateKey(privECDSA)
	fmt.Println("PRIVATE KEY IN PEM FORMAT:")
	pem.Encode(os.Stdout, &pem.Block{Type: "PRIVATE KEY", Bytes: derBytes})

}
