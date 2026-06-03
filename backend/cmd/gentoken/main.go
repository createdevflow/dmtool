package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"backend/internal/utils"
)

func main() {
	block, _ := pem.Decode([]byte(utils.DevPrivateKey))
	if block == nil {
		panic("failed to decode PEM block containing public key")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	token, err := utils.GenerateAccessToken(key, 1, "admin@dmtool.com", "owner")
	if err != nil {
		panic(err)
	}
	fmt.Print(token)
}
