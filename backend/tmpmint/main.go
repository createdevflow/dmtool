package main

import (
  "crypto/x509"
  "encoding/pem"
  "fmt"

  "backend/internal/utils"
)

func main() {
  block, _ := pem.Decode([]byte(utils.DevPrivateKey))
  key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
  if err != nil {
    panic(err)
  }
  token, err := utils.GenerateAccessToken(key, 1, "aryanvish86@gmail.com", "owner")
  if err != nil {
    panic(err)
  }
  fmt.Println(token)
}
