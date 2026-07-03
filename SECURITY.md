# Security Policy

## Reporting a vulnerability

Please email security issues to the project owners privately. Do not file public GitHub issues for suspected secrets or auth bypasses.

## Leaked credentials inventory (most recent security sweep: 2026-07-03)

The following credentials have appeared in git history or in tracked files at some point. All are treated as **public** regardless of later edits — git history is preserved, and `.env.example` files had real-looking values merged to GitHub.

| File (commit history) | Type | Coverage |
|---|---|---|
| `backend/internal/utils/jwt.go` — old `DevPrivateKey` constant (commit `b01cd78`) | RS256 private key, 2048-bit, JWT-signable | Removed from working tree. Replace any in-production key pairs via Render dashboard. Existing live access tokens issued under that key remain valid until expiry (15 min TTL). New keys: see § 3 below. |
| `backend/.env.example` — `META_APP_SECRET` (commits `b01cd78`, `150fa69` etc.) | Meta (Facebook/Instagram) OAuth client secret | Covered by § 5 below. |
| `.env.example` (root) — `GOOGLE_CLIENT_ID` + `GOOGLE_CLIENT_SECRET` + `META_APP_ID` + `META_APP_SECRET` | Google + Meta OAuth client IDs and secrets | Covered by §§ 4 and 5 below. |
| `backend/.env` (local, gitignored) — same Meta secret | Meta app secret | Same as `.env.example` row above. Treat as public even though never committed. |
| `backend/test_linkedin_api.go` — encrypted LinkedIn access token (deleted in phase 0, commit `f72185d`) | LinkedIn OAuth access token + legacy `ENCRYPTION_KEY` literal | **See § 6 for the explicit traceability row.** |

> Git history was **not** rewritten (per project decision). The leaked
> credentials remain visible in past commits until you explicitly choose a
> history-rewrite + force-push. The local secrets-hygiene rules below keep
> future commits clean, but do not retroactively remove leaks.

## 1. Local secrets hygiene

The repository's `.gitignore` covers:
- `.env`, `.env.*` at any depth.
- `*.pem`, `*.key`, `*.crt`.
- `backend/api.exe`, `backend/phase1_api.exe`, `backend/uploads/`.
- `*.db`, `*.sqlite`.

It does **not** and **cannot** cover credentials already in git history.

## 2. Coding conventions (don'ts)

- **Never** hardcode a credential, private key, or refresh token in source.
- **Never** commit `.env`, `.pem`, `.key`, `.crt`, or `*credentials*` outside templates.
- **Never** log a secret at info or above. Structured logger's secret-redaction is the only acceptable path.
- **Always** rotate per-environment — never share keys between dev, staging, and production.

## 3. JWT RS256 key pair

Generate locally, never commit, never log:

```bash
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem
```

Set `JWT_PRIVATE_KEY` and `JWT_PUBLIC_KEY` (PEM contents, newlines preserved
or escaped as `\n` depending on env loader). Production deployments MUST
set both. After rotation, **existing access tokens issued under the old key
will fail to verify** — users will need to log in again. Refresh tokens
are stored in DB and persist independently of the JWT-key rotation.

The old `DevPrivateKey` constant that lived in `backend/internal/utils/jwt.go`
(commits `b01cd78` and earlier) was itself a leaked production-capable key
and was removed in phase 0. Its removal does not retroactively invalidate
tokens already issued under it during the period it was live.

## 4. Google OAuth 2.0 (Search Console)

1. https://console.cloud.google.com → APIs & Services → Credentials.
2. Locate the OAuth 2.0 Client ID.
3. **Reset client secret.** Capture the new value.
4. Update OAuth Redirect URIs if production URL changed.
5. Place the new value in `.env` as `GOOGLE_CLIENT_SECRET`. Do not commit.

Resetting the client secret invalidates access tokens issued under that
secret. The deleted-but-historical `.env.example` and `.env` rows mentioned
specific google-client-secrets that are now treated as public.

## 5. Meta (Facebook / Instagram) OAuth

1. https://developers.facebook.com → My Apps → your app → Settings → Basic.
2. Click **Show** next to App Secret, then **Reset**.
3. Place the new value in `.env` as `META_APP_SECRET`.
4. Update the app's **Valid OAuth Redirect URIs** to match `FRONTEND_URL/api/integrations/callback`.

Resetting invalidates every access token issued under that secret, including
tokens stored in `oauth_credentials` rows. Users must reconnect Meta.

## 6. LinkedIn OAuth (and traceability for `test_linkedin_api.go`)

1. https://www.linkedin.com/developers/apps → your app → Auth.
2. **Reset client secret.** Revoking the secret invalidates every access
   token issued under that client.
3. After rotation, every user with a saved LinkedIn integration must reconnect.

### Cross-reference: `test_linkedin_api.go` leak (now deleted in phase 0)

The file `backend/test_linkedin_api.go` was deleted in phase 0 (commit
`f72185d`). It contained:

- An encrypted LinkedIn OAuth access token (encrypted with `sha256("change
  this later!")`).
- The legacy encryption-key literal `"change this later!"`.

Both of those credentials fall under **existing** rotation paths:

1. **The encrypted token** is an access token issued under the project's
   LinkedIn OAuth client (`cfg.LinkedinClientID` in `cmd/api/main.go`).
   Rotating the LinkedIn client secret per § 6 above **invalidates that
   token automatically**, regardless of where it was stored. No separate
   line item is needed.
2. **The encryption-key literal** `"change this later!"` is the same value
   that used to be the project's `ENCRYPTION_KEY` before it was rotated
   to the value currently in `backend/.env`. The current `ENCRYPTION_KEY`
   already invalidates anything encrypted under the prior value.

This cross-reference is informational — the coverage is provided by § 6
(LinkedIn client rotation) and § 7 (encryption-key rotation). No
additional concrete rotation step is required because of this leak. The
link from this specific token back to those sections is preserved here
for traceability so a future reader does not need the original
conversation context.

## 7. AES-256-GCM Encryption Key (`ENCRYPTION_KEY`)

Used to encrypt stored OAuth tokens at rest.

```bash
openssl rand -hex 32
```

Place the 64-character hex string in `.env`. **Rotating this key invalidates
every stored OAuth credential** — users will need to reconnect Google /
Meta / LinkedIn.

## 8. OpenAI

Rotate at https://platform.openai.com → API keys → Revoke + create new. Paste
into `OPENAI_API_KEY`.

## 9. Scan instructions

Quick:

```bash
git grep -nE "(PRIVATE KEY|SECRET|API_KEY|TOKEN)\s*=\s*\S{15,}" -- ':!.env.example' ':!.gitignore' ':!SECURITY.md'
```

Comprehensive (scans git history, not just HEAD):

```bash
docker run --rm -v "$PWD:/pwd" trufflesecurity/trufflehog git file:///pwd
```

TruffleHog / gitleaks add coverage of partial leaks, base64 blobs, and
historical commits.

---

End of policy. Update this file when credential inventory changes.
