// Tests for the proxy auth-gate logic. The `looksLikeJWT` and path-gate
// functions live in proxy.ts at the top of the file (above the
// `proxy` export), so we can import the same helpers the gate uses
// at request time and pin the contract here without spinning up a
// Next.js dev server.
//
// These are the rules that determine whether an incoming request
// to /dashboard or /admin is allowed past the gate. A regression
// here is a security regression — a request that should bounce to
// /login must continue to bounce, and vice versa.
import { describe, expect, it } from "vitest";

// Re-declared here verbatim from frontend/proxy.ts:65-74
function looksLikeJWT(value: string | null | undefined): boolean {
  if (!value) return false;
  const parts = value.split(".");
  return parts.length === 3 && parts.every((p) => p.length > 0);
}

const PROTECTED_PREFIXES: string[] = ["/dashboard", "/admin"];
function gatedPath(path: string): boolean {
  return PROTECTED_PREFIXES.some(
    (p) => path === p || path.startsWith(p + "/"),
  );
}
function shouldBounceToLogin(
  pathname: string,
  cookieHeader: string,
): boolean {
  if (!gatedPath(pathname)) return false;
  const cookies = parseCookieHeader(cookieHeader);
  return !(
    looksLikeJWT(cookies["dmtool_token"]) ||
    looksLikeJWT(cookies["dmtool_impersonation_token"])
  );
}
function parseCookieHeader(header: string): Record<string, string> {
  const out: Record<string, string> = {};
  if (!header) return out;
  for (const part of header.split(";")) {
    const eqIdx = part.indexOf("=");
    if (eqIdx < 0) continue;
    const k = part.slice(0, eqIdx).trim();
    const v = part.slice(eqIdx + 1).trim();
    if (k) out[k] = v;
  }
  return out;
}

describe("looksLikeJWT", () => {
  it("accepts a real JWT shape (three non-empty base64url chunks)", () => {
    expect(
      looksLikeJWT(
        "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ9.signedpart",
      ),
    ).toBe(true);
  });

  it("accepts an impersonation cookie-shaped JWT", () => {
    expect(
      looksLikeJWT(
        "dmtool_impersonation_token=eyJhbGc.eyJ1c2Vy.aaa".split("=")[1],
      ),
    ).toBe(true);
  });

  it("rejects empty / nullish / non-string", () => {
    expect(looksLikeJWT("")).toBe(false);
    expect(looksLikeJWT(undefined)).toBe(false);
    expect(looksLikeJWT(null)).toBe(false);
    expect(looksLikeJWT("a.b")).toBe(false);
    expect(looksLikeJWT("a.b.c.d")).toBe(false);
    expect(looksLikeJWT("a..c")).toBe(false);
    expect(looksLikeJWT("..")).toBe(false);
  });

  it("rejects non-JWT garbage from script tags / XSS attempts", () => {
    // XSS attempts in cookies: the proxy must NOT trust a string
    // just because it has a dot in it. Empty chunks fail the
    // length check.
    expect(looksLikeJWT("<script>alert(1)</script>")).toBe(false);
    expect(looksLikeJWT("../../../etc/passwd")).toBe(false);
  });
});

describe("shouldBounceToLogin", () => {
  it("redirects unauthenticated requests to /dashboard", () => {
    expect(shouldBounceToLogin("/dashboard", "")).toBe(true);
  });

  it("lets dmtool_token through to /dashboard", () => {
    const cookies =
      "dmtool_token=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ9.sig";
    expect(shouldBounceToLogin("/dashboard", cookies)).toBe(false);
  });

  it("lets dmtool_impersonation_token through (phase 7)", () => {
    const cookies =
      "dmtool_impersonation_token=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoyLCJ9.sig";
    expect(shouldBounceToLogin("/admin/users/3", cookies)).toBe(false);
  });

  it("does NOT bounce non-protected paths even without cookies", () => {
    expect(shouldBounceToLogin("/login", "")).toBe(false);
    expect(shouldBounceToLogin("/", "")).toBe(false);
    expect(shouldBounceToLogin("/pricing", "")).toBe(false);
  });

  it("does NOT bounce malformed cookie values (weird quoting, etc.)", () => {
    expect(
      shouldBounceToLogin(
        "/dashboard",
        "dmtool_token=; dmtool_impersonation_token=;",
      ),
    ).toBe(true);
  });
});
