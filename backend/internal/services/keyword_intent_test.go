package services

import "testing"

func TestGuessSearchIntent(t *testing.T) {
	cases := map[string]string{
		"buy running shoes":     "transactional",
		"best crm for agencies": "commercial",
		"how to set up gsc":     "informational",
		"plumber near me":       "local",
		"dmtool":                "unknown",
		"":                      "",
	}
	for q, want := range cases {
		if got := GuessSearchIntent(q); got != want {
			t.Fatalf("%q: got %s want %s", q, got, want)
		}
	}
}
