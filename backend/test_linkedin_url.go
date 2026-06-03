package main

import (
	"fmt"
	"strings"
	"backend/internal/config"
	"golang.org/x/oauth2"
)

func main() {
	cfg := config.Load()
	linkedinOAuthConfig := &oauth2.Config{
		ClientID:     cfg.LinkedinClientID,
		ClientSecret: cfg.LinkedinClientSecret,
		RedirectURL:  cfg.FrontendURL + "/integrations/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.linkedin.com/oauth/v2/authorization",
			TokenURL: "https://www.linkedin.com/oauth/v2/accessToken",
		},
		Scopes: strings.Split(cfg.LinkedinScopes, ","),
	}
	fmt.Println(linkedinOAuthConfig.AuthCodeURL("linkedin"))
}
