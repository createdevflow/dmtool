package main

import (
	"backend/internal/utils"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	// Encryption key from .env (raw string)
	encKeyStr := "change this later!"
	encKey := sha256.Sum256([]byte(encKeyStr))
	
	// Full encrypted token from DB
	encToken := "329ffee717cc2973da1c9db521a788c9b66aa82ed5fff497088e65d0fde816651e9b851757d23a104e3e7929852b35b691367282bcac74ab098bf2f27f6803aa3c8af31198e3144932c3e4c3d91800386174d95823b7fb08fc1d87d459108159b10d3364af875a24f32bee99f603d5011c4fad0cd8f3237667b11be16894a1ee07f096106b0987ba06b13b9869c8fc7f263d25f66b65996f692d6e6fbed2e9e9cf70abae544fcff11c8f896171db038385018f36b53aaead911072b7ff0fd6ca6d9c8108c6ef8199d52711e7adce7d50870609c85de585a3f25bd9cdf7bfa553caee7b87d624b3f0d3d4a801f829744fe014493f44aafef8859593957a277e53c47f0414b32d307c3a9215691f83f4a8ff3bf3d18af2c7865b70117bebe93c2431683095944f3fb5e93360758856ea19ca4914d3cd6b8e020dde8ddb89214e3e2d005749284cacf3fb023439f4bd5f2aea73f422e5d1c19276fa06042d3e95a52507f03fe6f26609e5983488e2dd3c2d1e45b3e782e21c4b21ad"
	
	// Trim whitespace
	encToken = strings.TrimSpace(encToken)
	
	// Decrypt
	decrypted, err := utils.Decrypt(encToken, encKey[:])
	if err != nil {
		fmt.Println("Decrypt error:", err)
		return
	}

	fmt.Println("Token:", decrypted)

	// Now test the LinkedIn API
	req, _ := http.NewRequest("GET", "https://api.linkedin.com/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+decrypted)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("API error:", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Response:", string(body[:500]))
}
