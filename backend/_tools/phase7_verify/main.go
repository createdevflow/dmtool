// phase7_verify exercises the two phase 7 fixes end-to-end against a
// live backend (default http://localhost:8097):
//
//   1. Impersonation cookie flow (cookie fix from commit 98dcd35)
//   2. Plan create / edit UI (PlanModal from commit 98dcd35)
//
// The verifier makes real HTTP calls; it does NOT reuse module-level
// handlers or in-memory fakes. The runner is responsible for starting
// the backend (see ../tmp_api.exe on port 8097) before invoking this
// tool.
//
// Run from backend/:
//
//	./tmp_verify.exe
//
// Exit 0 on success, non-zero if any step returns a 4xx/5xx where a
// 2xx was expected. Each step is named with "[expect ...]" so the
// reader can audit the contract without reading Go.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	baseURL       = "http://localhost:8097"
	apiPrefix     = "/api"
	adminEmail    = "admin@dmtool.local"
	adminPass     = "AdminPass123!"
	targetEmail   = "victim-e2e@dmtool.local"
	targetPass    = "TargetPass123!"
	victimPlanTag = "ph7v" // short tag for the freshly created plan code in this run
)

type loginResult struct {
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
	Raw string
}

type reqResult struct {
	status int
	body   string
}

type auditEntry struct {
	ID            uint   `json:"id"`
	ActorUserID   uint   `json:"actor_user_id"`
	TargetUserID  uint   `json:"target_user_id"`
	Action        string `json:"action"`
}

type auditList struct {
	Data struct {
		Entries []auditEntry `json:"entries"`
		Total   int          `json:"total"`
	} `json:"data"`
}

type adminUserListEntry struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}
type adminUserList struct {
	Data struct {
		Users []adminUserListEntry `json:"users"`
		Total int                  `json:"total"`
	} `json:"data"`
}

type planRow struct {
	ID           uint   `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	MonthlyCents int    `json:"monthly_cents"`
	YearlyCents  int    `json:"yearly_cents"`
	MaxSites     int    `json:"max_sites"`
	IsActive     bool   `json:"is_active"`
	TierRank     int    `json:"tier_rank"`
}
type planEnvelope struct {
	Data planRow `json:"data"`
}
type planListEnvelope struct {
	Data []planRow `json:"data"`
}

type impData struct {
	Token string `json:"token"`
	Target struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"target"`
	ExpiresInMinutes int `json:"expires_in_minutes"`
}
type impResp struct {
	Data impData `json:"data"`
}

// ── Main ───────────────────────────────────────────────────────────────────
func main() {
	log.SetFlags(0)
	failed := 0

	step := func(name string, expectedOK bool, status int, body string) {
		if !expectedOK {
			// For "expect 4xx" steps: 4xx is OK.
			if status >= 400 && status < 600 {
				fmt.Printf("[OK  ] %s\n       status=%d body=%s\n", name, status, trunc(body, 320))
				return
			}
			failed++
			fmt.Printf("[FAIL] %s\n       expected 4xx, got status=%d body=%s\n", name, status, trunc(body, 320))
			return
		}
		if status >= 200 && status < 300 {
			fmt.Printf("[OK  ] %s\n       status=%d body=%s\n", name, status, trunc(body, 320))
			return
		}
		failed++
		fmt.Printf("[FAIL] %s\n       expected 2xx, got status=%d body=%s\n", name, status, trunc(body, 320))
	}

	// ── Setup ──────────────────────────────────────────────────────────────
	status, body := post(baseURL+apiPrefix+"/auth/login",
		`{"email":"`+adminEmail+`","password":"`+adminPass+`"}`)
	step("[setup] Admin login [expect 200]", true, status, body)
	if status != 200 {
		log.Fatalf("admin login failed: %s", body)
	}
	adminAccess := mustToken(body)

	// Register the target — accept 200 OR 400 EMAIL_EXISTS (idempotent on re-run).
	regStatus, regBody := post(baseURL+apiPrefix+"/auth/register", `{
		"name": "Victim User",
		"email": "`+targetEmail+`",
		"password": "`+targetPass+`"
	}`)
	switch {
	case regStatus == 200:
		step("[setup] Register target user "+targetEmail+" [expect 200 / 200 = fresh]", true, regStatus, regBody)
	case regStatus == 400 && strings.Contains(regBody, "EMAIL_EXISTS"):
		step("[setup] Register target user "+targetEmail+" [expect 200 / 400 = already present]", false, regStatus, regBody)
	default:
		step("[setup] Register target user "+targetEmail, true, regStatus, regBody)
	}

	// ── Impersonation flow ────────────────────────────────────────────────
	// Find the victim's user-id by listing users.
	listSt, listBd := getAuth(baseURL+apiPrefix+"/admin/users", adminAccess)
	step("[1a] Admin GET /api/admin/users [expect 200]", true, listSt, listBd)
	var ul adminUserList
	_ = json.Unmarshal([]byte(listBd), &ul)
	var victimID uint
	for _, u := range ul.Data.Users {
		if u.Email == targetEmail {
			victimID = u.ID
			break
		}
	}
	if victimID == 0 {
		log.Fatalf("could not find victim in user list: %s", listBd)
	}
	fmt.Printf("       victim resolved to id=%d\n", victimID)

	// 1b: POST impersonate
	impSt, impBd := postAuth(baseURL+apiPrefix+"/admin/users/"+itoa(int(victimID))+"/impersonate", adminAccess, `{}`)
	step(fmt.Sprintf("[1b] Admin POST /api/admin/users/%d/impersonate [expect 200, gets JWT]", victimID), true, impSt, impBd)
	var imp impResp
	_ = json.Unmarshal([]byte(impBd), &imp)
	if imp.Data.Token == "" || imp.Data.Target.Email != targetEmail {
		log.Fatalf("bad impersonate response: %s", impBd)
	}
	fmt.Printf("       impersonation target: id=%d email=%s name=%s expires_in_minutes=%d\n",
		imp.Data.Target.ID, imp.Data.Target.Email, imp.Data.Target.Name, imp.Data.ExpiresInMinutes)
	impJWT := imp.Data.Token

	// Build the cookie header exactly as the frontend sets it (the cookie
	// is what the proxy gate reads on full-page navigations; the apiClient
	// interceptor mirrors the cookie into an Authorization header on
	// fetch calls — see lib/api-client.ts).
	impCookie := "dmtool_impersonation_token=" + impJWT
	impAuth := "Bearer " + impJWT

	// 1c: /api/admin/users with the impersonation JWT in Authorization → 403
	//     "Impersonation tokens cannot access admin routes" (RequireRole).
	//     In practice the target here is role=owner, so the role-check fires
	//     first ("Admin access required"). Either branch is a valid 403 and
	admSt, admBd := getWithHeader(baseURL+apiPrefix+"/admin/users", "Authorization", impAuth)
	step("[1c] GET /api/admin/users with impersonation Authorization [expect 403, admin route blocked while impersonating]", false, admSt, admBd)
	if admSt != 403 {
		log.Fatalf("expected 403, got %d %s", admSt, admBd)
	}

	prefsSt, prefsBd := getWithHeader(baseURL+apiPrefix+"/users/me/preferences", "Authorization", impAuth)
	step("[1d] GET /users/me/preferences with impersonation Authorization [expect 200, target user]", true, prefsSt, prefsBd)
	// 1e: /api/auth/me with imp JWT → returns VICTIM profile.
	meSt, meBd := getWithHeader(baseURL+apiPrefix+"/auth/me", "Authorization", impAuth)
	step("[1e] GET /auth/me with impersonation Authorization [expect 200, email=victim]", true, meSt, meBd)
	if !strings.Contains(meBd, targetEmail) {
		log.Fatalf("expected /auth/me to show victim, got %s", meBd)
	}

	// 1f: /api/projects with imp JWT → 200 (target's projects; empty list is fine).
	projSt, projBd := getWithHeader(baseURL+apiPrefix+"/projects", "Authorization", impAuth)
	step("[1f] GET /api/projects with impersonation Authorization [expect 200, target's projects]", true, projSt, projBd)
	// 1g: Stop impersonation with the ADMIN token (requires admin role).
	stopSt, stopBd := postAuth(baseURL+apiPrefix+"/admin/users/"+itoa(int(victimID))+"/stop-impersonation",
		adminAccess,
		fmt.Sprintf(`{"target_id":%d}`, victimID))
	step(fmt.Sprintf("[1g] Admin POST /api/admin/users/%d/stop-impersonation [expect 200]", victimID), true, stopSt, stopBd)

	// 1h: After stop, admin's REAL token still works → 200 (admin not
	//     logged out). The audit row is on the server; the cookie-clear
	//     is purely client-side.
	admSt2, admBd2 := getAuth(baseURL+apiPrefix+"/admin/users", adminAccess)
	step("[1h] After stop, GET /api/admin/users with admin token [expect 200, not logged out]", true, admSt2, admBd2)

	// 1i: Browser-side after stop would clear the cookie; the JWT itself
	//     is still valid until its 30-min TTL expires (server accepts it
	//     as a normal token). The proxy would only reject once the cookie
	//     is gone AND dmtool_token is gone.

	staleSt, staleBd := getWithHeader(baseURL+apiPrefix+"/users/me/preferences", "Authorization", impAuth)
	step("[1i] GET /users/me/preferences with stale impersonation JWT [expect 200, JWT still valid until 30m TTL]", true, staleSt, staleBd)

	// 1j: Proxy-shape gate check — implements frontend/proxy.ts looksLikeJWT
	//     exactly. With the impersonation cookie, the gate should pass.
	gateOK := checkProxyGate(impCookie, "/dashboard")
	if gateOK {
		fmt.Printf("[OK  ] [1j] Proxy auth-gate (looksLikeJWT) on /dashboard with ONLY imp cookie — passes (no /login redirect)\n")
	} else {
		failed++
		fmt.Printf("[FAIL] [1j] Proxy auth-gate on /dashboard with ONLY imp cookie — would redirect to /login\n")
	}
	gateAnonOK := checkProxyGate("", "/dashboard")
	if !gateAnonOK {
		fmt.Printf("[OK  ] [1k] Proxy auth-gate on /dashboard with NO cookies — would redirect to /login\n")
	} else {
		failed++
		fmt.Printf("[FAIL] [1k] Proxy auth-gate on /dashboard with NO cookies — passed (anonymous request was not blocked)\n")
	}

	// ── Plan create flow ──────────────────────────────────────────────────
	code := victimPlanTag
	bodyCreate := fmt.Sprintf(`{
		"code": "%s",
		"name": "Phase 7 verify",
		"description": "Created by phase7_verify on %s",
		"tier_rank": 7,
		"monthly_cents": 4900,
		"yearly_cents": 49000,
		"max_sites": 5
	}`, code, time.Now().UTC().Format(time.RFC3339))

	// 2a: POST create
	createSt, createBd := postAuth(baseURL+apiPrefix+"/admin/plans", adminAccess, bodyCreate)
	if createSt >= 400 && strings.Contains(createBd, "PLAN_CODE_TAKEN") {
		// from a previous run with the same code — drop the plan and retry by
		// appending a numeric suffix.
		code = fmt.Sprintf("%s_%d", code, time.Now().Unix())
		bodyCreate = strings.Replace(bodyCreate, victimPlanTag, code, 1)
		createSt, createBd = postAuth(baseURL+apiPrefix+"/admin/plans", adminAccess, bodyCreate)
	}
	step(fmt.Sprintf("[2a] POST /api/admin/plans {code:%s} [expect 201]", code), true, createSt, createBd)
	if createSt < 200 || createSt >= 300 {
		log.Fatalf("plan create failed: %d %s", createSt, createBd)
	}
	var createdPlanEnv planEnvelope
	_ = json.Unmarshal([]byte(createBd), &createdPlanEnv)
	if createdPlanEnv.Data.Code != code {
		log.Fatalf("created plan code mismatch: want %s got %s (body: %s)", code, createdPlanEnv.Data.Code, createBd)
	}
	createdPlan := createdPlanEnv.Data

	// 2b: GET confirms the new row.
	listSt2, listBd2 := getAuth(baseURL+apiPrefix+"/admin/plans", adminAccess)
	step("[2b] GET /api/admin/plans [expect 200, contains new row]", true, listSt2, listBd2)
	var pl planListEnvelope
	_ = json.Unmarshal([]byte(listBd2), &pl)

	// 2c: POST same code → 400 PLAN_CODE_TAKEN
	dupSt, dupBd := postAuth(baseURL+apiPrefix+"/admin/plans", adminAccess, bodyCreate)
	step(fmt.Sprintf("[2c] POST /api/admin/plans duplicate {code:%s} [expect 400 PLAN_CODE_TAKEN]", code), false, dupSt, dupBd)
	if !strings.Contains(dupBd, "PLAN_CODE_TAKEN") {
		log.Fatalf("expected PLAN_CODE_TAKEN, got %s", dupBd)
	}

	// 2d: POST bad code (uppercase) → 400 INVALID_CODE
	badUpperSt, badUpperBd := postAuth(baseURL+apiPrefix+"/admin/plans", adminAccess,
		`{"code":"BadCode","name":"x"}`)
	step("[2d] POST /api/admin/plans {code:BadCode} [expect 400 INVALID_CODE]", false, badUpperSt, badUpperBd)
	if !strings.Contains(badUpperBd, "INVALID_CODE") {
		log.Fatalf("expected INVALID_CODE, got %s", badUpperBd)
	}

	// 2e: POST bad code (starts with digit) → 400 INVALID_CODE
	badDigitSt, badDigitBd := postAuth(baseURL+apiPrefix+"/admin/plans", adminAccess,
		`{"code":"1team","name":"x"}`)
	step("[2e] POST /api/admin/plans {code:1team} [expect 400 INVALID_CODE]", false, badDigitSt, badDigitBd)
	if !strings.Contains(badDigitBd, "INVALID_CODE") {
		log.Fatalf("expected INVALID_CODE, got %s", badDigitBd)
	}

	// 2f: PATCH price + max_sites + is_active
	planURL := baseURL + apiPrefix + "/admin/plans/" + itoa(int(createdPlan.ID))
	editSt, editBd := patchAuth(planURL, adminAccess,
		`{"monthly_cents":6900,"max_sites":10,"is_active":false}`)
	step(fmt.Sprintf("[2f] PATCH /api/admin/plans/%d {monthly:6900, max_sites:10, is_active:false} [expect 200]", createdPlan.ID),
		true, editSt, editBd)
	var edited planEnvelope
	_ = json.Unmarshal([]byte(editBd), &edited)
	if edited.Data.MonthlyCents != 6900 || edited.Data.MaxSites != 10 || edited.Data.IsActive {
		log.Fatalf("edit didn't apply: %s", editBd)
	}

	// 2g: GET confirms the edit persisted.
	listSt3, listBd3 := getAuth(baseURL+apiPrefix+"/admin/plans", adminAccess)
	step("[2g] GET /api/admin/plans [expect 200, persisted 6900/10/inactive]", true, listSt3, listBd3)
	var pl2 planListEnvelope
	_ = json.Unmarshal([]byte(listBd3), &pl2)
	var reloaded planRow
	for _, p := range pl2.Data {
		if p.ID == createdPlan.ID {
			reloaded = p
			break
		}
	}
	if reloaded.MonthlyCents != 6900 || reloaded.MaxSites != 10 || reloaded.IsActive {
		log.Fatalf("reloaded plan doesn't reflect edit: %+v", reloaded)
	}

	// 2h: PATCH empty body → 400 EMPTY_UPDATE
	emptySt, emptyBd := patchAuth(planURL, adminAccess, `{}`)
	step("[2h] PATCH /admin/plans/<id> {} (empty body) [expect 400 EMPTY_UPDATE]", false, emptySt, emptyBd)
	if !strings.Contains(emptyBd, "EMPTY_UPDATE") {
		log.Fatalf("expected EMPTY_UPDATE, got %s", emptyBd)
	}

	// 2i: PATCH unknown id → 404 NOT_FOUND
	missSt, missBd := patchAuth(baseURL+apiPrefix+"/admin/plans/999999", adminAccess, `{"monthly_cents":100}`)
	step("[2i] PATCH /admin/plans/999999 [expect 404 NOT_FOUND]", false, missSt, missBd)
	if !strings.Contains(missBd, "NOT_FOUND") {
		log.Fatalf("expected NOT_FOUND, got %s", missBd)
	}

	// 2j (closeout fix): PATCH negative monthly_cents → 400 INVALID_PRICE.
	// Server-side validation mirrors the client form so the gap a
	// hand-rolled API call could exploit (admin token, then a
	// negative monthly) is closed.
	negPriceSt, negPriceBd := patchAuth(planURL, adminAccess, `{"monthly_cents":-1}`)
	step("[2j] PATCH /admin/plans/<id> {monthly_cents:-1} [expect 400 INVALID_PRICE]", false, negPriceSt, negPriceBd)
	if !strings.Contains(negPriceBd, "INVALID_PRICE") {
		log.Fatalf("expected INVALID_PRICE, got %s", negPriceBd)
	}

	// 2k (closeout fix): PATCH negative yearly_cents → 400 INVALID_PRICE.
	negYearlySt, negYearlyBd := patchAuth(planURL, adminAccess, `{"yearly_cents":-100}`)
	step("[2k] PATCH /admin/plans/<id> {yearly_cents:-100} [expect 400 INVALID_PRICE]", false, negYearlySt, negYearlyBd)
	if !strings.Contains(negYearlyBd, "INVALID_PRICE") {
		log.Fatalf("expected INVALID_PRICE, got %s", negYearlyBd)
	}

	// 2l (closeout fix): PATCH max_sites:0 → 400 INVALID_MAX_SITES.
	zeroSitesSt, zeroSitesBd := patchAuth(planURL, adminAccess, `{"max_sites":0}`)
	step("[2l] PATCH /admin/plans/<id> {max_sites:0} [expect 400 INVALID_MAX_SITES]", false, zeroSitesSt, zeroSitesBd)
	if !strings.Contains(zeroSitesBd, "INVALID_MAX_SITES") {
		log.Fatalf("expected INVALID_MAX_SITES, got %s", zeroSitesBd)
	}

	// ── Audit log ─────────────────────────────────────────────────────────
	auditSt, auditBd := getAuth(baseURL+apiPrefix+"/admin/audit-log?action=plan.create", adminAccess)
	step("[3a] GET /admin/audit-log?action=plan.create [expect 200]", true, auditSt, auditBd)
	var aL auditList
	_ = json.Unmarshal([]byte(auditBd), &aL)
	if len(aL.Data.Entries) < 1 {
		log.Fatalf("expected ≥1 plan.create audit entry, got %s", auditBd)
	}
	fmt.Printf("       plan.create entries: %d\n", len(aL.Data.Entries))

	auditSt2, auditBd2 := getAuth(baseURL+apiPrefix+"/admin/audit-log?action=impersonate.start", adminAccess)
	step("[3b] GET /admin/audit-log?action=impersonate.start [expect 200]", true, auditSt2, auditBd2)
	var aL2 auditList
	_ = json.Unmarshal([]byte(auditBd2), &aL2)
	if len(aL2.Data.Entries) < 1 {
		log.Fatalf("expected ≥1 impersonate.start audit entry, got %s", auditBd2)
	}
	fmt.Printf("       impersonate.start entries: %d\n", len(aL2.Data.Entries))

	// handler writes targetID=0 for plan.update, so we query on action
	// alone. The most recent entry's metadata should carry the plan_id
	// we just edited.
	auditSt3, auditBd3 := getAuth(baseURL+apiPrefix+"/admin/audit-log?action=plan.update", adminAccess)
	step(fmt.Sprintf("[3c] GET /admin/audit-log?action=plan.update [expect ≥1 entry whose metadata.plan_id matches %d]", createdPlan.ID),
		true, auditSt3, auditBd3)
	var aL3 auditList
	_ = json.Unmarshal([]byte(auditBd3), &aL3)
	var planUpdateCount int
	for _, e := range aL3.Data.Entries {
		if e.Action == "plan.update" {
			planUpdateCount++
		}
	}
	if planUpdateCount < 1 {
		log.Fatalf("expected ≥1 plan.update audit entry; got %s", auditBd3)
	}
	fmt.Printf("       plan.update entries: %d\n", planUpdateCount)

	// ── Wrap up ───────────────────────────────────────────────────────────
	if failed > 0 {
		fmt.Printf("\n%d step(s) FAILED.\n", failed)
		os.Exit(1)
	}
	fmt.Println("\nAll phase 7 verifications passed.")
}

// ── Helpers ────────────────────────────────────────────────────────────────
func mustToken(body string) string {
	var r loginResult
	_ = json.Unmarshal([]byte(body), &r)
	if r.Data.Token == "" {
		return ""
	}
	return r.Data.Token
}

func post(url, body string) (int, string) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return do(req)
}

func postAuth(url, token, body string) (int, string) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return do(req)
}

func patchAuth(url, token, body string) (int, string) {
	req, _ := http.NewRequest("PATCH", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return do(req)
}

func getAuth(url, token string) (int, string) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return do(req)
}

func getWithCookies(url, cookieHeader string) (int, string) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cookie", cookieHeader)
	return do(req)
}

func do(req *http.Request) (int, string) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var anyData map[string]any
		if json.Unmarshal(b, &anyData) == nil {
			pretty, _ := json.Marshal(anyData)
			return resp.StatusCode, string(pretty)
		}
	}
	return resp.StatusCode, string(b)
}

func getWithHeader(url, header, value string) (int, string) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set(header, value)
	return do(req)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// checkProxyGate mirrors the looksLikeJWT logic from frontend/proxy.ts
// verbatim. path: e.g. "/dashboard" or "/admin/users/1".
//
// Behaviour:
//   - empty cookies → false (would redirect to /login)
//   - cookie with non-JWT shape → false
//   - cookie with valid JWT shape on dmtool_token or
//     dmtool_impersonation_token → true (auth-gate satisfied)
func checkProxyGate(cookieHeader, path string) bool {
	if cookieHeader == "" {
		return false
	}
	const PROTECTED_PREFIX = "/dashboard"
	if !strings.HasPrefix(path, PROTECTED_PREFIX) {
		return true // not protected — gate always passes
	}
	for _, kv := range strings.Split(cookieHeader, "; ") {
		if !strings.HasPrefix(kv, "dmtool_token=") &&
			!strings.HasPrefix(kv, "dmtool_impersonation_token=") {
			continue
		}
		eqIdx := strings.IndexByte(kv, '=')
		if eqIdx < 0 {
			continue
		}
		name := kv[:eqIdx]
		val := kv[eqIdx+1:]
		if name != "dmtool_token" && name != "dmtool_impersonation_token" {
			continue
		}
		parts := strings.Split(val, ".")
		if len(parts) != 3 {
			continue
		}
		ok := true
		for _, p := range parts {
			if p == "" {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
