// Package config loads all environment variables via Viper and exposes a
// single Config struct consumed by every other package in the application.
package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config holds every environment variable the application needs.
type Config struct {
	Port             string
	DatabaseURL      string
	JWTPrivateKeyPEM string
	JWTPublicKeyPEM  string
	EncryptionKey    string // 32 bytes, hex or raw
	AllowedOrigins   []string
	FrontendURL      string

	// Data Providers
	DataForSEOLogin    string
	DataForSEOPassword string
	RapidAPIKey        string

	// OpenAI
	OpenAIAPIKey string

	// Integrations
	MetaAppID          string
	MetaAppSecret      string
	MetaPageAccessToken string
	GoogleClientID     string
	GoogleClientSecret string

	// App
	AppEnv  string // development | production
	Version string
}

// Load reads environment variables (and an optional .env file) using Viper.
// It panics on startup if a required variable is missing so the problem is
// surfaced immediately rather than at runtime.
func Load() *Config {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// Best-effort read; if .env is absent, fall back to real environment
	if err := viper.ReadInConfig(); err != nil {
		log.Println("[config] .env not found — reading from environment:", err)
	}

	// Defaults
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("VERSION", "2.0.0")
	viper.SetDefault("DATABASE_URL", "file:dmtool.db")
	viper.SetDefault("FRONTEND_URL", "http://localhost:3000")
	viper.SetDefault("ALLOWED_ORIGINS", "http://localhost:3000")

	cfg := &Config{
		Port:             viper.GetString("PORT"),
		DatabaseURL:      viper.GetString("DATABASE_URL"),
		JWTPrivateKeyPEM: viper.GetString("JWT_PRIVATE_KEY"),
		JWTPublicKeyPEM:  viper.GetString("JWT_PUBLIC_KEY"),
		EncryptionKey:    viper.GetString("ENCRYPTION_KEY"),
		FrontendURL:      viper.GetString("FRONTEND_URL"),
		AllowedOrigins:   strings.Split(viper.GetString("ALLOWED_ORIGINS"), ","),

		DataForSEOLogin:    viper.GetString("DATAFORSEO_LOGIN"),
		DataForSEOPassword: viper.GetString("DATAFORSEO_PASSWORD"),
		RapidAPIKey:        viper.GetString("RAPIDAPI_KEY"),

		OpenAIAPIKey: viper.GetString("OPENAI_API_KEY"),

		MetaAppID:          viper.GetString("META_APP_ID"),
		MetaAppSecret:      viper.GetString("META_APP_SECRET"),
		MetaPageAccessToken: viper.GetString("META_PAGE_ACCESS_TOKEN"),
		GoogleClientID:     viper.GetString("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: viper.GetString("GOOGLE_CLIENT_SECRET"),

		AppEnv:  viper.GetString("APP_ENV"),
		Version: viper.GetString("VERSION"),
	}

	// Validation: RS256 keys are required in production
	if cfg.AppEnv == "production" {
		if cfg.JWTPrivateKeyPEM == "" || cfg.JWTPublicKeyPEM == "" {
			log.Fatal("[config] JWT_PRIVATE_KEY and JWT_PUBLIC_KEY are required in production")
		}
		if len(cfg.EncryptionKey) < 32 {
			log.Fatal("[config] ENCRYPTION_KEY must be at least 32 bytes in production")
		}
	}

	return cfg
}
