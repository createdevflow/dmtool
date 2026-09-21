// nav_mode_counts extracts the curated-subset nav layout from
// sidebar.tsx and prints per-mode item counts. The browser DOM isn't
// available in this repo's test runner, so we parse the source
// statically using only regexes. Pure-data, no string scanning.
//
// Run from the project root so the relative path resolves.
//
//   $ ./navmode
//   [OK  ] search  : 11 items in 4 groups (target: 11)
//   [OK  ] social  : 11 items in 4 groups (target: 11)
//   [OK  ] combined: 13 items in 5 groups (target: 13)
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type navItem struct {
	Name       string
	Href       string
	ModeExtras []string
}

type navGroup struct {
	Title string
	Modes []string
	Items []navItem
}

func main() {
	raw, err := os.ReadFile("frontend/components/dashboard/sidebar.tsx")
	if err != nil {
		fmt.Printf("read: %v\n", err)
		os.Exit(1)
	}
	src := string(raw)

	groups := parseGroups(src)
	if len(groups) == 0 {
		fmt.Println("no groups parsed — file shape changed?")
		os.Exit(1)
	}

	targets := map[string]int{
		"search":   13,
		"social":   13,
		"combined": 16,
	}

	failed := false
	for mode, want := range targets {
		count := 0
		groupCount := 0
		for _, g := range groups {
			if !contains(g.Modes, mode) {
				continue
			}
			visible := 0
			for _, it := range g.Items {
				if it.ModeExtras == nil || contains(it.ModeExtras, mode) {
					visible++
				}
			}
			if visible == 0 {
				continue
			}
			groupCount++
			count += visible
		}
		status := "OK  "
		if count != want {
			status = "FAIL"
			failed = true
		}
		fmt.Printf("[%s] %-8s: %2d items in %d groups (target: %d)\n",
			status, mode, count, groupCount, want)
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("\nAll per-mode counts match the curated-subset spec.")
}

// navGroupsHeader locates the start of the array literal. The body's
// outer `[` is the next character after this match.
var navGroupsHeader = regexp.MustCompile(`const navGroups:\s*NavGroup\[\]\s*=\s*\[`)

// modesField matches `modes: [a, b]` (array literal) OR `modes: ALL`
// (identifier shorthand) — see sidebar.tsx for the const declarations
// at the top of the file.
var modesField = regexp.MustCompile(`modes:\s*(?:\[([^\]]+)\]|([A-Z_][A-Z0-9_]*))`)

// navItemLine matches a single nav item. Items can be single-line
// `{ name: "...", href: "...", icon: X }` or multi-line (when
// modeExtras is set) — so we match any whitespace including newlines,
// but we *do* require `name:` and `href:` to appear before any
// `modeExtras:`. The closing `}` is mandatory.
var navItemLine = regexp.MustCompile(
	`(?s)\{\s*name:\s*"([^"]+)",\s*href:\s*"([^"]+)"(?:[^}]*?icon:\s*[A-Za-z_][A-Za-z0-9_]*)?(?:[^}]*?modeExtras:\s*\[([^\]]+)\])?\s*,?\s*\}?`,
)

// navGroupBoundary finds the `{` that opens a group and the matching
// `}` that closes it. We walk forward from a known `title:` and find
// the matching close brace using a simple counter.
var navGroupTitle = regexp.MustCompile(`title:\s*"([^"]+)"`)

func parseGroups(src string) []navGroup {
	// Locate the array literal.
	header := navGroupsHeader.FindStringIndex(src)
	if header == nil {
		return nil
	}
	body := src[header[1]:]

	// Find each `title:`. The corresponding `}` is the group's
	// close. Between title and close, we extract the `modes:` value
	// and every `name:` / `href:` pair.
	var groups []navGroup
	for {
		titleMatch := navGroupTitle.FindStringIndex(body)
		if titleMatch == nil {
			break
		}
		title := navGroupTitle.FindStringSubmatch(body)[1]

		// Find the matching close brace from the `{` before this title.
		titleAbs := titleMatch[0]
		openIdx := findOpenBraceBefore(body, titleAbs)
		if openIdx < 0 {
			break
		}
		closeIdx := matchBrace(body, openIdx)
		if closeIdx < 0 {
			break
		}
		groupText := body[openIdx : closeIdx+1]

		// Pull modes value.
		modesList := ""
		if m := modesField.FindStringSubmatch(groupText); m != nil {
			if m[1] != "" {
				modesList = m[1]
			} else {
				modesList = m[2]
			}
		}
		modes := parseModeList(modesList)

		// Pull items.
		items := parseItemsIn(groupText)

		groups = append(groups, navGroup{Title: title, Modes: modes, Items: items})
		// Continue searching after this group.
		body = body[closeIdx+1:]
	}
	return groups
}

func parseItemsIn(groupText string) []navItem {
	var items []navItem
	for _, m := range navItemLine.FindAllStringSubmatch(groupText, -1) {
		it := navItem{
			Name: m[1],
			Href: m[2],
		}
		if m[3] != "" {
			it.ModeExtras = parseModeList(m[3])
		}
		items = append(items, it)
	}
	return items
}

func parseModeList(s string) []string {
	// s is a comma-separated list of mode literals ("search", "social",
	// "combined") OR a single identifier ("ALL", "SEARCH_OR_COMBINED").
	// We expand an identifier to the matching constant set.
	words := modeWordRegex.FindAllStringSubmatch(s, -1)
	if len(words) > 0 {
		out := make([]string, 0, len(words))
		for _, w := range words {
			out = append(out, w[1])
		}
		return out
	}
	// Identifier — expand based on the const set.
	if strings.TrimSpace(s) == "ALL" {
		return []string{"search", "social", "combined"}
	}
	if strings.TrimSpace(s) == "SEARCH_OR_COMBINED" {
		return []string{"search", "combined"}
	}
	if strings.TrimSpace(s) == "SOCIAL_OR_COMBINED" {
		return []string{"social", "combined"}
	}
	return nil
}

var modeWordRegex = regexp.MustCompile(`"(search|social|combined)"`)

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func findOpenBraceBefore(body string, to int) int {
	depth := 0
	var inString byte
	for i := to - 1; i >= 0; i-- {
		c := body[i]
		if inString != 0 {
			if c == inString {
				inString = 0
			}
			continue
		}
		switch c {
		case '"', '\'', '`':
			inString = c
		case '}':
			depth++
		case '{':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func matchBrace(body string, openIdx int) int {
	depth := 0
	var inString byte
	for i := openIdx; i < len(body); i++ {
		c := body[i]
		if inString != 0 {
			if c == '\\' && i+1 < len(body) {
				i++
				continue
			}
			if c == inString {
				inString = 0
			}
			continue
		}
		switch c {
		case '"', '\'', '`':
			inString = c
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

var _ = strings.TrimSpace
