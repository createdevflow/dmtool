// auth-cookie.ts — central place for the dmtool_token / dmtool_user /
// dmtool_mode cookies. The frontend used to store these in localStorage;
// phase 3 switches to non-HttpOnly cookies so the Next 16 proxy can read
// them at the edge for auth gating.
//
// Security notes (phase 3):
// - dmtool_token is NOT HttpOnly because the api-client reads it on
//   the client to build the Authorization header. Same as the prior
//   localStorage trade-off, just slightly more accessible (e.g. to
//   XSS), offset by the proxy's SameSite=Lax cookie default. A future
//   phase can move token into memory only and have a refresh cookie,
//   but that's a bigger refactor and out of scope here.
// - SameSite=Lax is the recommended default; we omit Secure in dev
//   where the cookie would otherwise block http://localhost.
// - Path=/ keeps it scoped to the app, not /api.

export const COOKIE_TOKEN = "dmtool_token";
export const COOKIE_USER = "dmtool_user";
export const COOKIE_MODE = "dmtool_mode";

function cookieOptions(days: number): string {
  const expires = new Date(Date.now() + days * 24 * 60 * 60 * 1000).toUTCString();
  // SameSite=Lax; Path=/; 7 days for token, 30 for user/mode.
  return `Path=/; SameSite=Lax; Expires=${expires}`;
}

export function setAuthCookie(name: string, value: string, days = 7): void {
  if (typeof document === "undefined") return;
  const safe = encodeURIComponent(value);
  document.cookie = `${name}=${safe}; ${cookieOptions(days)}`;
}

export function clearAuthCookie(name: string): void {
  if (typeof document === "undefined") return;
  document.cookie = `${name}=; Path=/; SameSite=Lax; Expires=Thu, 01 Jan 1970 00:00:00 GMT`;
}

export function setToken(token: string): void {
  setAuthCookie(COOKIE_TOKEN, token, 7);
}

export function setUser(user: unknown): void {
  setAuthCookie(COOKIE_USER, JSON.stringify(user), 30);
}

export function setMode(mode: "search" | "social" | "combined"): void {
  setAuthCookie(COOKIE_MODE, mode, 30);
}

export function clearAuth(): void {
  clearAuthCookie(COOKIE_TOKEN);
  clearAuthCookie(COOKIE_USER);
}

export function readCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const prefix = `${name}=`;
  for (const c of document.cookie.split(";")) {
    const trimmed = c.trim();
    if (trimmed.startsWith(prefix)) {
      return decodeURIComponent(trimmed.slice(prefix.length));
    }
  }
  return null;
}
