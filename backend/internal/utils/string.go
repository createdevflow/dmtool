package utils

import (
	"fmt"
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
