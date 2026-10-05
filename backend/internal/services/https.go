package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const httpsCategory = "https"

// httpsDialTLSConfig is nil in production. Tests set it so httptest certs verify.
var httpsDialTLSConfig *tls.Config

// HTTPSReport is the structured HTTPS/TLS outcome from an audit of one URL.
type HTTPSReport struct {
	URL              string    `json:"url"`
	ServesHTTPS      bool      `json:"serves_https"`
	RedirectsHTTP    bool      `json:"redirects_http"`
	RedirectNote     string    `json:"redirect_note,omitempty"`
	TLSValid         bool      `json:"tls_valid"`
	TLSError         string    `json:"tls_error,omitempty"`
	CertExpiry       string    `json:"cert_expiry,omitempty"`
	CertIssuer       string    `json:"cert_issuer,omitempty"`
	HostnameMatch    bool      `json:"hostname_match"`
	HSTS             bool      `json:"hsts"`
	MixedActive      []string  `json:"mixed_active,omitempty"`
	MixedPassive     []string  `json:"mixed_passive,omitempty"`
	Score            int       `json:"score"`
	Status           string    `json:"status"`
	CheckedAt        time.Time `json:"checked_at"`
}

// InspectHTTPS checks scheme, HTTP→HTTPS redirect, certificate, HSTS, and mixed content
// on the audited page (not a sitewide crawl).
func InspectHTTPS(client *http.Client, pageURL *url.URL, doc *html.Node, headers http.Header, checkedAt time.Time) (HTTPSReport, []AuditCheck) {
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	report := HTTPSReport{CheckedAt: checkedAt.UTC()}
	if pageURL == nil {
		report.Status = CheckFail
		return report, []AuditCheck{httpsCheck("HTTPS implementation", CheckFail, "high", "No URL to inspect.", "")}
	}
	report.URL = pageURL.String()
	report.ServesHTTPS = strings.EqualFold(pageURL.Scheme, "https")
	if headers != nil && strings.TrimSpace(headers.Get("Strict-Transport-Security")) != "" {
		report.HSTS = true
	}

	checked := formatRobotsChecked(checkedAt)
	checks := make([]AuditCheck, 0, 6)

	if report.ServesHTTPS {
		checks = append(checks, httpsCheck("HTTPS implementation", CheckPass, "high",
			fmt.Sprintf("Page is served over HTTPS (%s). Last checked %s.", pageURL.Host, checked), ""))
	} else {
		checks = append(checks, httpsCheck("HTTPS implementation", CheckFail, "high",
			fmt.Sprintf("Page is served over HTTP: %s. Last checked %s.", pageURL.String(), checked),
			"Install a TLS certificate and serve the site on HTTPS only."))
	}

	redir, redirNote, redirOK := checkHTTPRedirect(client, pageURL)
	report.RedirectsHTTP = redir
	report.RedirectNote = redirNote
	switch {
	case redir:
		checks = append(checks, httpsCheck("HTTP → HTTPS redirect", CheckPass, "high",
			redirNote, ""))
	case redirOK && report.ServesHTTPS:
		checks = append(checks, httpsCheck("HTTP → HTTPS redirect", CheckWarning, "medium",
			redirNote,
			"Redirect http:// to https:// so old links and type-in traffic are encrypted."))
	case redirOK:
		checks = append(checks, httpsCheck("HTTP → HTTPS redirect", CheckFail, "high",
			redirNote,
			"Redirect all HTTP requests to HTTPS."))
	default:
		checks = append(checks, httpsCheck("HTTP → HTTPS redirect", CheckWarning, "low",
			redirNote,
			"Could not prove an HTTP→HTTPS redirect (port 80 closed is common on HTTPS-only hosts)."))
	}

	tlsRep := checkTLS(pageURL)
	report.TLSValid = tlsRep.valid
	report.TLSError = tlsRep.err
	report.CertExpiry = tlsRep.expiry
	report.CertIssuer = tlsRep.issuer
	report.HostnameMatch = tlsRep.hostname
	if tlsRep.valid {
		detail := fmt.Sprintf("TLS handshake succeeded. Certificate matches %s.", pageURL.Hostname())
		if tlsRep.expiry != "" {
			detail += " Expires " + tlsRep.expiry + "."
		}
		if tlsRep.issuer != "" {
			detail += " Issuer: " + tlsRep.issuer + "."
		}
		status, sev := CheckPass, "high"
		if tlsRep.expiresSoon {
			status, sev = CheckWarning, "medium"
			detail += " Certificate expires within 21 days."
		}
		checks = append(checks, httpsCheck("SSL certificate", status, sev, detail, ""))
	} else {
		checks = append(checks, httpsCheck("SSL certificate", CheckFail, "high",
			"TLS check failed: "+tlsRep.err,
			"Install a valid certificate that matches this hostname and is not expired."))
	}

	if report.ServesHTTPS && doc != nil {
		active, passive := findMixedContent(doc)
		report.MixedActive = capMixed(active, 8)
		report.MixedPassive = capMixed(passive, 8)
		switch {
		case len(active) > 0:
			checks = append(checks, httpsCheck("Mixed content", CheckFail, "high",
				fmt.Sprintf("Active mixed content (scripts/styles/iframes over HTTP): %s", strings.Join(report.MixedActive, ", ")),
				"Load scripts, stylesheets, and iframes over HTTPS only."))
		case len(passive) > 0:
			checks = append(checks, httpsCheck("Mixed content", CheckWarning, "medium",
				fmt.Sprintf("Passive mixed content (images/media over HTTP): %s", strings.Join(report.MixedPassive, ", ")),
				"Serve images and media over HTTPS."))
		default:
			checks = append(checks, httpsCheck("Mixed content", CheckPass, "medium",
				"No http:// resources found on this page.", ""))
		}
	} else if !report.ServesHTTPS {
		checks = append(checks, httpsCheck("Mixed content", CheckFail, "medium",
			"Page is HTTP, so every resource is unencrypted.",
			"Move the page to HTTPS first."))
	}

	if report.ServesHTTPS {
		if report.HSTS {
			checks = append(checks, httpsCheck("HSTS", CheckPass, "low",
				"Strict-Transport-Security header is present.", ""))
		} else {
			checks = append(checks, httpsCheck("HSTS", CheckWarning, "low",
				"No Strict-Transport-Security header.",
				"Add HSTS so browsers remember to use HTTPS."))
		}
	}

	report.Score = httpsScore(report)
	report.Status = worstRobotsStatus(checks)
	return report, checks
}

func httpsCheck(label, status, severity, detail, rec string) AuditCheck {
	return AuditCheck{
		Category:       httpsCategory,
		Label:          label,
		Status:         status,
		Severity:       severity,
		Detail:         detail,
		Recommendation: rec,
	}
}

func httpsScore(r HTTPSReport) int {
	n := 0
	if r.ServesHTTPS {
		n += 40
	}
	if r.TLSValid {
		n += 25
	}
	if r.RedirectsHTTP {
		n += 15
	} else if r.ServesHTTPS {
		n += 5
	}
	if r.ServesHTTPS && len(r.MixedActive) == 0 {
		n += 15
	}
	if r.HSTS {
		n += 5
	}
	if n > 100 {
		n = 100
	}
	return n
}

type tlsResult struct {
	valid        bool
	hostname     bool
	expiresSoon  bool
	err          string
	expiry       string
	issuer       string
}

func checkTLS(pageURL *url.URL) tlsResult {
	host := pageURL.Hostname()
	if host == "" {
		return tlsResult{err: "missing hostname"}
	}
	port := pageURL.Port()
	if port == "" || pageURL.Scheme == "http" {
		port = "443"
	}
	addr := net.JoinHostPort(host, port)
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	cfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if httpsDialTLSConfig != nil {
		cfg = httpsDialTLSConfig.Clone()
		if cfg.ServerName == "" {
			cfg.ServerName = host
		}
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, cfg)
	if err != nil {
		return tlsResult{err: err.Error()}
	}
	defer conn.Close()
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return tlsResult{err: "no peer certificate"}
	}
	cert := state.PeerCertificates[0]
	out := tlsResult{
		valid:     true,
		hostname:  true,
		expiry:    cert.NotAfter.UTC().Format("2006-01-02"),
		issuer:    cert.Issuer.CommonName,
	}
	if out.issuer == "" && len(cert.Issuer.Organization) > 0 {
		out.issuer = cert.Issuer.Organization[0]
	}
	if time.Until(cert.NotAfter) < 21*24*time.Hour {
		out.expiresSoon = true
	}
	return out
}

func checkHTTPRedirect(client *http.Client, pageURL *url.URL) (redirects bool, note string, checked bool) {
	httpURL := httpEquivalent(pageURL)
	if httpURL == nil {
		return false, "Could not build an HTTP URL to test redirect.", false
	}
	if client == nil {
		client = http.DefaultClient
	}
	redirClient := &http.Client{
		Timeout: 8 * time.Second,
		Transport: client.Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 8 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	req, err := http.NewRequest(http.MethodGet, httpURL.String(), nil)
	if err != nil {
		return false, "Could not build HTTP request: " + err.Error(), false
	}
	req.Header.Set("User-Agent", "DMTool-SEOCrawler/2.0 (+https://dmtool.app)")
	resp, err := redirClient.Do(req)
	if err != nil {
		return false, "HTTP (" + httpURL.String() + ") is not reachable: " + err.Error() + ". Redirect could not be verified.", false
	}
	defer resp.Body.Close()
	final := resp.Request.URL
	if final != nil && strings.EqualFold(final.Scheme, "https") {
		return true, "HTTP redirects to " + final.String() + ".", true
	}
	if final != nil {
		return false, "HTTP stayed on " + final.String() + " (HTTP " + fmt.Sprintf("%d", resp.StatusCode) + ").", true
	}
	return false, fmt.Sprintf("HTTP responded %d without an HTTPS redirect.", resp.StatusCode), true
}

func httpEquivalent(u *url.URL) *url.URL {
	if u == nil || u.Hostname() == "" {
		return nil
	}
	c := *u
	c.Scheme = "http"
	if c.Port() == "443" {
		c.Host = c.Hostname()
	}
	if c.Path == "" {
		c.Path = "/"
	}
	return &c
}

func findMixedContent(n *html.Node) (active, passive []string) {
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			tag := strings.ToLower(node.Data)
			for _, a := range node.Attr {
				name := strings.ToLower(a.Key)
				if name != "src" && name != "href" && name != "action" && name != "data" && name != "poster" && name != "srcset" {
					continue
				}
				for _, raw := range splitSrcset(a.Val) {
					if !isHTTPResource(raw) {
						continue
					}
					if isActiveMixed(tag, name) {
						active = append(active, raw)
					} else {
						passive = append(passive, raw)
					}
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return
}

func splitSrcset(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if i := strings.IndexByte(p, ' '); i > 0 {
			p = p[:i]
		}
		out = append(out, p)
	}
	return out
}

func isHTTPResource(raw string) bool {
	s := strings.TrimSpace(raw)
	return strings.HasPrefix(strings.ToLower(s), "http://")
}

func isActiveMixed(tag, attr string) bool {
	switch tag {
	case "script", "iframe", "object", "embed":
		return true
	case "link":
		return attr == "href"
	case "form":
		return attr == "action"
	default:
		return false
	}
}

func capMixed(in []string, n int) []string {
	if len(in) <= n {
		return in
	}
	return append(append([]string{}, in[:n]...), fmt.Sprintf("…and %d more", len(in)-n))
}
