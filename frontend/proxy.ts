// proxy.ts — Next 16's proxy file (renamed from middleware). Runs on the
// edge before any route renders. Responsibilities:
//
//   1. Auth gate — block requests to /(dashboard|admin)/* that don't
//      carry a dmtool_token cookie. Unauthenticated requests redirect to
//      /login with a return-URL so the user lands back where they tried
//      to go.
//
//   (The former /dashboard → /dashboard/<mode> mode rewrite was removed
//   when the richer dashboard page became the permanent /dashboard.
//   Mode is read client-side via the DashboardMode context instead.)
//
// CSRF protection: state-changing requests (non-GET/HEAD) MUST come from
// a trusted origin. The Origin header is mandatory per the Fetch spec for
// POST/PUT/PATCH/DELETE; we compare it against an allow-list built from
// the ALLOWED_ORIGINS env var (the same one the Go backend reads for its
// CORS middleware in cmd/api/main.go). This keeps one source of truth.
//
// SECURITY: This proxy is a UX gate, not a security boundary. The real
// check happens in the Go backend's JWTAuth middleware. A forged token
// passes our format check here but is rejected at the API layer. See
// SECURITY.md § 6 for the rotation policy that keeps the JWT signature
// trustworthy.

import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Static lookup tables per project style (Record not Set for fixed membership).
const PROTECTED_PREFIXES: string[] = ["/dashboard", "/admin"];
const SAFE_METHODS: Record<string, true> = {
  GET: true,
  HEAD: true,
  OPTIONS: true,
};

// buildTrustedOrigins reads the same env var the backend reads for its
// CORS configuration (ALLOWED_ORIGINS, comma-separated). Localhost
// origins are appended automatically so a missing env doesn't break
// local development. We do NOT add speculative production hostnames;
// whoever deploys the app is responsible for setting ALLOWED_ORIGINS on
// both Vercel and Render to match. A change in either place without the
// other will cause CSRF failures — by design, since both sides must
// agree on what origins are trusted.
function buildTrustedOrigins(): Set<string> {
  const env = process.env.ALLOWED_ORIGINS ?? "";
  const parts = env
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
  // Local dev origins are always trusted. They are not secret; they
  // exist only because the dev server is local. Including them by
  // default means a fresh local checkout doesn't need an env tweak.
  parts.push("http://localhost:3000", "http://127.0.0.1:3000");
  return new Set(parts);
}

const TRUSTED_ORIGINS: Set<string> = buildTrustedOrigins();

// looksLikeJWT reports whether the cookie value has the shape of a JWT
// (three non-empty base64url chunks separated by '.'). We do NOT verify
// the signature here — that is the backend's job. The shape check only
// proves the cookie was set by a successful login and is not the empty
// string someone could send from a script. See SECURITY note at top.
function looksLikeJWT(value: string | undefined): boolean {
  if (!value) return false;
  const parts = value.split(".");
  return parts.length === 3 && parts.every((p) => p.length > 0);
}

export function proxy(request: NextRequest) {
  const { pathname, search } = request.nextUrl;
  const method = request.method.toUpperCase();

  // ----- CSRF guard for state-changing requests -------------------
  if (!SAFE_METHODS[method]) {
    const origin = request.headers.get("origin");
    if (origin !== null && !TRUSTED_ORIGINS.has(origin)) {
      return new NextResponse(
        JSON.stringify({
          success: false,
          error: {
            code: "CSRF_BLOCKED",
            message: "Cross-origin request blocked.",
          },
        }),
        {
          status: 403,
          headers: { "content-type": "application/json" },
        }
      );
    }
  }

  // ----- Auth gate ----------------------------------------------------
  // The gate is satisfied if EITHER the admin's own session token
  // (dmtool_token) OR an active impersonation token
  // (dmtool_impersonation_token) is present and JWT-shaped. The
  // apiClient (see lib/api-client.ts) prefers the impersonation
  // token when both are set, so the admin's real token keeps the
  // proxy happy while the impersonation token carries the actual
  // API calls. We do NOT require both — the admin's real token is
  // already absent during the normal user flow.
  const protectedPath = PROTECTED_PREFIXES.some(
    (p) => pathname === p || pathname.startsWith(p + "/")
  );
  if (
    protectedPath &&
    !looksLikeJWT(request.cookies.get("dmtool_token")?.value) &&
    !looksLikeJWT(request.cookies.get("dmtool_impersonation_token")?.value)
  ) {
    const loginUrl = request.nextUrl.clone();
    loginUrl.pathname = "/login";
    loginUrl.search = "";
    if (pathname !== "/login" && pathname !== "/register") {
      loginUrl.searchParams.set("return_to", pathname + (search || ""));
    }
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  // Skip the API (backend handles its own auth), Next internals, static
  // files, and explicit asset suffixes.
  matcher: [
    "/((?!api|_next/static|_next/image|favicon.ico|.*\\.(?:png|jpg|jpeg|gif|svg|ico|css|js)$).*)",
  ],
};
