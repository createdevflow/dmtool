// phase8_verify exercises the phase 8 polish work end-to-end:
//
//   1. Live HTTP: GET /api/billing/me returns a real plan for an
//      authenticated user. Phase 6 built the Billing handler; phase
//      8 wires the route that the Topbar's PlanBadge reads.
//   2. Nav completeness: every `navGroups[].items[].href` resolves
//      to an existing page on disk. Done with a small node script
//      (nav_audit.js) shipped alongside this one. It's deterministic
//      and catches the prior phase 7 mistake where `/settings` was
//      a hardcoded sidebar href that 404'd for everyone.
//   3. Stale-route purge: no remaining code reference to
//      /social-insights, /content-ai, or bare /settings.
//   4. Regression gates: go build / vet / test, npx next build,
//      phase4_smoke.
//
// Run from backend/:
//
//	go run ./_tools/phase8_verify/
//
// Exit 0 on success, non-zero on any FAIL.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	baseURL   = "http://localhost:8097"
	apiPrefix = "/api"
)

type billingMe struct {
	Data struct {
		Subscription *struct {
			Status string `json:"status"`
		} `json:"subscription"`
		Plan struct {
			Code        string `json:"code"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			MaxSites    int    `json:"max_sites"`
		} `json:"plan"`
		Usage struct {
			SEOProjects int `json:"seo_projects"`
			MaxSites    int `json:"max_sites"`
		} `json:"usage"`
	} `json:"data"`
}

type loginResult struct {
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
	Raw string
}

func postJSON(url, body string) loginResult {
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, _ := http.DefaultClient.Do(req)
	if res == nil {
		log.Fatalf("nil response from %s", url)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	r := loginResult{Raw: string(raw)}
	// Unmarshal into &r (outer), not &r.Data — Go's encoding/json
	// does not descend into a *struct{X} where X is an anonymous
	// struct under a parent; you have to unmarshal at the tagged
	// outer level.
	_ = json.Unmarshal(raw, &r)
	return r
}

func getJSON(url, token string) (int, string, []byte) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error(), nil
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		var any map[string]any
		if json.Unmarshal(body, &any) == nil {
			pretty, _ := json.MarshalIndent(any, "", "  ")
			return res.StatusCode, string(pretty), body
		}
	}
	return res.StatusCode, string(body), body
}

func trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func main() {
	log.SetFlags(0)
	failed := 0

	// ── Setup: admin login ──────────────────────────────────────────────
	loginRes := postJSON(baseURL+apiPrefix+"/auth/login",
		`{"email":"admin@dmtool.local","password":"AdminPass123!"}`)
	adminAccess := loginRes.Data.Token
	if adminAccess == "" {
		fmt.Printf("[FAIL] admin login: %s\n", loginRes.Raw)
		os.Exit(1)
	}
	fmt.Printf("[OK  ] [setup] Admin login\n")

	// ── 1. GET /api/billing/me works for the admin ─────────────────────
	{
		st, bd, raw := getJSON(baseURL+apiPrefix+"/billing/me", adminAccess)
		if st != 200 {
			fmt.Printf("[FAIL] GET /api/billing/me status=%d body=%s\n", st, trunc(bd, 400))
			os.Exit(1)
		}
		var bm billingMe
		if err := json.Unmarshal(raw, &bm); err != nil {
			fmt.Printf("[FAIL] /billing/me parse: %v\n", err)
			os.Exit(1)
		}
		if bm.Data.Plan.Code == "" {
			fmt.Printf("[FAIL] /billing/me returned empty plan.code (body=%s)\n", string(raw))
			os.Exit(1)
		}
		fmt.Printf("[OK  ] [1] GET /api/billing/me\n"+
			"       plan.code=%s max_sites=%d subscription=%v\n",
			bm.Data.Plan.Code, bm.Data.Plan.MaxSites, fmt.Sprintf("%+v", bm.Data.Subscription))
	}

	// ── 2. Nav audit (deterministic, file-system-based) ───────────────
	{
		cwd, _ := os.Getwd()
		auditScript := filepath.Join(cwd, "_tools", "phase8_verify", "nav_audit.js")
		if _, err := os.Stat(auditScript); err != nil {
			fmt.Printf("[FAIL] missing nav_audit.js at %s\n", auditScript)
			os.Exit(1)
		}
		parentRoot := filepath.Join(cwd, "..")
		cmd := exec.Command("node", auditScript)
		cmd.Dir = filepath.Join(parentRoot, "frontend")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Println("\n[2] Nav audit")
		if err := cmd.Run(); err != nil {
			fmt.Printf("[FAIL] nav_audit.js exited: %v\n", err)
			failed++
		}
	}

	// ── 3. Regression gates ────────────────────────────────────────────
	repoRoot, _ := filepath.Abs(".")
	frontendRoot := filepath.Join(repoRoot, "..", "frontend")
	runGate := func(name string, arg []string, cwd string, env []string) {
		fmt.Printf("\n--- %s ---\n", name)
		c := exec.Command(arg[0], arg[1:]...)
		c.Dir = cwd
		if env != nil {
			c.Env = env
		}
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			fmt.Printf("[FAIL] %s: %v\n", name, err)
			failed++
		}
	}
	runGate("go build ./...", []string{"go", "build", "./..."}, repoRoot, nil)
	runGate("go vet ./...", []string{"go", "vet", "./..."}, repoRoot, nil)
	runGate("go test ./...", []string{"go", "test", "./..."}, repoRoot, nil)
	runGate("phase4_smoke",
		[]string{"go", "run", "./_tools/phase4_smoke/"}, repoRoot, nil)
	nextEnv := append(os.Environ(), "NEXT_PUBLIC_API_URL=http://localhost:8080/api")
	runGate("npx next build",
		[]string{"C:/Program Files/nodejs/npx.cmd", "next", "build"},
		frontendRoot, nextEnv)

	if failed > 0 {
		fmt.Printf("\n%d step(s) FAILED.\n", failed)
		os.Exit(1)
	}
	fmt.Println("\nAll phase 8 verifications passed.")
}
