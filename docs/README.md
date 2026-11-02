## API

Reference for the HTTP API this backend serves. Start with **Conventions** — it covers what every endpoint shares (auth header, error body, `null` handling, PATCH semantics, the full error code list), so the per-resource pages don't repeat it.

### API

- [Conventions](./api/conventions.md) — read first
- [Health](./api/health.md)
- [Users](./api/users.md)
- [Tokens](./api/tokens.md)
- [Profiles](./api/profiles.md)
- [Uploads](./api/uploads.md)
- [Messages](./api/messages.md)

### Development

- [Testing](./testing.md) — test layers, integration suite, tests needing external services

### Typical client flows

**Sign up**

1. `POST /api/v1/users/signup` → keep `accessToken` in memory; the refresh token arrives as an httpOnly cookie the browser stores on its own.
2. The server mails a verification link to `<FRONTEND_BASE_URL>/verify-email?token=<token>` (valid **5 minutes**).
3. Frontend page `/verify-email` reads `token` and calls `POST /api/v1/users/verify-email?token=<token>` with the user's access token.
   - Link expired or lost → `POST /api/v1/messages/verification-email` (no body) mails a fresh one; on `429 verification_email_rate_limited` wait `details.retryAfterSeconds`.
4. `POST /api/v1/profiles` to record first/last name.

**Forgotten password**

1. Sign-in page → "forgot password" → `POST /api/v1/messages/password-reset-email` with `{"email": ...}`. Always `200` after ~1 second; tell the user to check their inbox if the account exists.
2. The server mails `<FRONTEND_BASE_URL>/reset-password?token=<token>` (valid **15 minutes**, single use) — only to password accounts.
3. Frontend page `/reset-password` reads `token`, asks for the new password, calls `POST /api/v1/users/reset-password` with `{"token": ..., "newPassword": ...}` (no access token).
   - `400 password_reset_link_invalid` → offer to request a new link.
4. Every session was ended → send the user to sign-in with the new password.

**Sign up / in with Google**

1. `GET /api/v1/users/google-config` → `clientId` (browser-cached 1 hour). Render Google's Sign in with Google button (Google Identity Services) with that client ID; its callback hands back a `credential` (an ID token).
2. New user → `POST /api/v1/users/google-signup` with `{"idToken": credential}`; returning user → `POST /api/v1/users/google-signin`. Same session handling as password sign-in; the email is verified already, so skip the verification step.
   - `404 google_account_not_found` on sign-in → offer google signup; `409 google_account_taken` on signup → sign in instead.
   - `409 google_account_not_linked` / `409 email_taken` → the email has a password account; send the user to password sign-in.
3. Optional: signed-in Google user → `POST /api/v1/users/add-password` with `{"password": ...}` → the account also signs in with email + password.
4. Optional: signed-in password user → Google button → `POST /api/v1/users/add-google` with `{"idToken": credential}` → the account also signs in with Google. Google's email must equal the account's (`409 google_email_mismatch` otherwise).

**Session**

- Access token lives **1 hour**; refresh token **30 days**.
- The refresh token never reaches JS: it lives in the `refresh_token` httpOnly cookie, scoped to `/api/v1/tokens`. Serve the API same-origin (Vite dev: proxy `/api` → `http://localhost:3000`) so the browser sends it.
- On page load / `401 access_token_expired` → `POST /api/v1/tokens/refresh` (no body, no header) → replace the access token; the cookie rotates on its own.
- On `401 refresh_token_invalid` → send the user to sign-in.
- Logout → `POST /api/v1/tokens/revoke` (clears the cookie), then drop the access token.

**Profile with a picture**

1. `POST /api/v1/uploads` (`purpose=profile_picture`, one image) → take `uploads[0].id`.
2. `POST /api/v1/profiles` (or `PUT` to replace) with `pictureUploadId` set to that id.
3. Unattached uploads are deleted after **24 hours**; a replaced picture is deleted within about **24 hours** of being replaced.
