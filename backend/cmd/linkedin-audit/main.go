package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	clientID     = "77ko22ex6t33ed"
	clientSecret = "" // TODO: Use os.Getenv("LINKEDIN_CLIENT_SECRET") or inject via config
	redirectURI  = "http://localhost:8081/callback"
)

func main() {
	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Scopes to request
		scopes := []string{
			"r_liteprofile", "r_emailaddress",
			"w_member_social",
		}
		authURL := fmt.Sprintf("https://www.linkedin.com/oauth/v2/authorization?response_type=code&client_id=%s&redirect_uri=%s&scope=%s",
			clientID, url.QueryEscape(redirectURI), url.QueryEscape(strings.Join(scopes, " ")))

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<h1>LinkedIn Audit Start</h1>
			<p>Please click the link below to authorize the application:</p>
			<a href="%s" style="font-size: 20px; font-weight: bold; padding: 10px; background: #0a66c2; color: white; text-decoration: none; border-radius: 5px;">Authorize with LinkedIn</a>
		`, authURL)
	})

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintf(w, "Error: No code returned. URL: %s", r.URL.String())
			return
		}

		fmt.Fprintf(w, "<h2>Authorization Code Received!</h2><p>Exchanging for Access Token...</p>")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Exchange code for token
		token, err := exchangeToken(code)
		if err != nil {
			fmt.Fprintf(w, "<p style='color:red;'>Token Exchange Error: %v</p>", err)
			return
		}

		fmt.Fprintf(w, "<h3>Access Token Acquired Successfully. Running Audit...</h3>")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		runAudit(w, token)

		go func() {
			time.Sleep(2 * time.Second)
			server.Shutdown(context.Background())
		}()
	})

	fmt.Println("=========================================================")
	fmt.Println("🚀 LinkedIn Audit Server Running!")
	fmt.Println("👉 OPEN YOUR BROWSER AND GO TO: http://localhost:8081")
	fmt.Println("=========================================================")

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
	fmt.Println("Audit complete. Server shut down.")
}

func exchangeToken(code string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)

	req, _ := http.NewRequest("POST", "https://www.linkedin.com/oauth/v2/accessToken", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", err
	}

	return res.AccessToken, nil
}

func runAudit(w io.Writer, token string) {
	fmt.Fprintf(w, "<pre style='background: #1e1e1e; color: #00ff00; padding: 20px; border-radius: 8px;'>\n")
	
	type EndpointTest struct {
		Name     string
		URL      string
		Expected string
	}

	tests := []EndpointTest{
		{"User Info (OpenID)", "https://api.linkedin.com/v2/userinfo", "sub"},
		{"Me (r_liteprofile / r_basicprofile)", "https://api.linkedin.com/v2/me", "id"},
		{"Company Page Access (r_organization_admin)", "https://api.linkedin.com/v2/organizationAcls?q=roleAssignee", "elements"},
		{"Post Access (w_member_social)", "https://api.linkedin.com/v2/ugcPosts", ""}, // Can't fully test without payload, but can check 403 vs 400
	}

	for _, test := range tests {
		fmt.Fprintf(w, "Testing: %s\n", test.Name)
		fmt.Fprintf(w, "Endpoint: %s\n", test.URL)
		
		req, _ := http.NewRequest("GET", test.URL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		
		if err != nil {
			fmt.Fprintf(w, "Result: FAILED (%v)\n\n", err)
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		fmt.Fprintf(w, "Status: %d\n", resp.StatusCode)
		
		var parsed map[string]interface{}
		json.Unmarshal(body, &parsed)
		
		if resp.StatusCode == 200 {
			fmt.Fprintf(w, "Result: SUCCESS\n")
			// Print snippet of data safely
			if len(body) > 300 {
				fmt.Fprintf(w, "Response: %s...\n\n", string(body[:300]))
			} else {
				fmt.Fprintf(w, "Response: %s\n\n", string(body))
			}
		} else {
			fmt.Fprintf(w, "Result: FAILED\n")
			fmt.Fprintf(w, "Error: %s\n\n", string(body))
		}
	}

	// Introspection
	fmt.Fprintf(w, "--- Introspecting Token Scopes ---\n")
	introReq, _ := http.NewRequest("GET", "https://api.linkedin.com/v2/introspectToken", nil)
	introReq.Header.Set("Authorization", "Bearer "+token) // Note: introspection typically requires client_credentials but we can see what error we get.
	introResp, _ := http.DefaultClient.Do(introReq)
	introBody, _ := io.ReadAll(introResp.Body)
	introResp.Body.Close()
	fmt.Fprintf(w, "Status: %d\nResponse: %s\n", introResp.StatusCode, string(introBody))

	fmt.Fprintf(w, "</pre>\n")
	fmt.Fprintf(w, "<h2>Audit Script Finished. Please check your terminal or this page for results.</h2>")
}
