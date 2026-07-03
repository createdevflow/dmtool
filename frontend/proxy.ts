// proxy.ts — Next 16's renamed middleware file. Runs on the edge before
// any route renders. Two responsibilities in phase 3:
//
//   1. Auth gate — block requests to /(dashboard|admin)/* that don't
//      carry a dmtool_token cookie. Unauthenticated requests redirect to
//      /login with a return-URL so the user lands back where they tried
//      to go.
//
//   2. Mode rewrite — /dashboard and /admin without a mode segment get
//      rewritten to /dashboard/<mode> and /admin/<mode>. Mode is read
//      from the dmtool_mode cookie, falling back to "combined" until
//      phase 4 wires the server-side preference.
//
// CSRF protection: state-changing requests (non-GET/HEAD) MUST come from
// a trusted origin. The Origin header is mandatory per the Fetch spec for
// POST/PUT/PATCH/DELETE; we compare it to a small allow-list. Browsers
// that strip Origin are tolerated only for safe verbs.
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
const DASHBOARD_MODES: Record<string, true> = {
  search: true,
  social: true,
  combined: true,
};
const TRUSTED_ORIGINS: Record<string, true> = {
  "http://localhost:3000": true,
  "http://127.0.0.1:3000": true,
  "https://dmtool-eight.vercel.app": true,
  "https://dmtool.com": true,
};
const SAFE_METHODS: Record<string, true> = {
  GET: true,
  HEAD: true,
  OPTIONS: true,
};

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
    if (origin !== null && !TRUSTED_ORIGINS[origin]) {
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

  // ----- Mode rewrite ----------------------------------------------
  if (pathname === "/dashboard" || pathname === "/dashboard/") {
    const cookieMode = request.cookies.get("dmtool_mode")?.value;
    const mode =
      cookieMode !== undefined && DASHBOARD_MODES[cookieMode] === true
        ? cookieMode
        : "combined";
    const url = request.nextUrl.clone();
    url.pathname = `/dashboard/${mode}`;
    return NextResponse.rewrite(url);
  }
  if (pathname === "/admin" || pathname === "/admin/") {
    const url = request.nextUrl.clone();
    url.pathname = `/admin/combined`;
    return NextResponse.rewrite(url);
  }

  // ----- Auth gate -----------------------------------------------
  const protectedPath = PROTECTED_PREFIXES.some(
    (p) => pathname === p || pathname.startsWith(p + "/")
  );
  if (
    protectedPath &&
    !looksLikeJWT(request.cookies.get("dmtool_token")?.value)
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
