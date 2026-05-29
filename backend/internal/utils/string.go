package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// DeriveNameFromURL extracts a clean brand name from a URL.
// Example: "https://app.google.com/search" -> "Google"
func DeriveNameFromURL(rawURL string) string {
	clean := strings.TrimPrefix(rawURL, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.TrimPrefix(clean, "www.")
	
	parts := strings.Split(clean, ".")
	if len(parts) > 0 {
		name := parts[0]
		if name == "app" || name == "dashboard" || name == "www" {
			if len(parts) > 1 {
				name = parts[1]
			}
		}
		return strings.Title(name)
	}
	
	return "New Project"
}

// NormalizeSocialHandle strips common URL prefixes and leading @ symbols so
// social handles can be compared against API usernames consistently.
func NormalizeSocialHandle(raw string) string {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, "@")
	clean = strings.TrimSuffix(clean, "/")

	if clean == "" {
		return ""
	}

	if strings.Contains(clean, "://") {
		if parsed, err := url.Parse(clean); err == nil {
			clean = parsed.Path
		}
	}

	clean = strings.Trim(clean, "/")
	if clean == "" {
		return ""
	}

	parts := strings.Split(clean, "/")
	clean = parts[len(parts)-1]
	clean = strings.TrimSpace(clean)
	clean = strings.TrimPrefix(clean, "@")
	return strings.ToLower(clean)
}

// CalculateChange returns the percentage difference between current and previous values.
func CalculateChange(current, previous int64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0
		}
		return 0.0
	}
	return (float64(current-previous) / float64(previous)) * 100.0
}

// FormatNumber converts a large integer into a readable string (e.g. 1.2k).
func FormatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	}
	return fmt.Sprintf("%.1fm", float64(n)/1000000.0)
}
