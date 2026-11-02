# baseUrl/api/v1/users

Common decoding, auth and 500 errors are listed in [Conventions](./conventions.md) and not repeated here.

---

## POST /signup

- ### Description:

  Create an account with email + password. Answers with a session right away (access token in the body, refresh token in the `refresh_token` httpOnly cookie — see [Tokens](./tokens.md)) — the user is signed in but their email is **unverified**.

- ### Constraints:

  - Email: trimmed + lowercased, must be an email.
  - Password: 8–64 characters, no whitespace anywhere. Taken verbatim (not trimmed).

- ### Authorization:

  None.

- ### Request:

```typescript
interface SignUpRequest {
  email: string;
  password: string;
}
```

Example:

```json
{
  "email": "jane@example.com",
  "password": "correct-horse-battery"
}
```

- ### Response:

  `201 Created`

```typescript
interface SignUpResponse {
  id: string; // uuid
  email: string;
  createdAt: string; // ISO 8601
  emailVerifiedAt: string | null; // always null on a fresh account
  signinAt: string; // signing up counts as the first sign-in
  firstName: string | null; // always null: a new account has no profile yet
  lastName: string | null; // always null: a new account has no profile yet
  signInMethods: SignInMethod[]; // see SignInMethod in POST /signin
  accessToken: string; // JWT, 1 hour
}
```

Same user fields as [POST /signin](#post-signin), read back after the account is stored.

```json
{
  "id": "6f1c2a8e-3b4d-4e5f-9a0b-1c2d3e4f5a6b",
  "email": "jane@example.com",
  "createdAt": "2026-10-27T09:15:00Z",
  "emailVerifiedAt": null,
  "signinAt": "2026-10-27T09:15:00Z",
  "firstName": null,
  "lastName": null,
  "signInMethods": ["email_password"],
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

Also sets the refresh token cookie (30 days) — see [Tokens](./tokens.md#refresh-token-cookie):

```
Set-Cookie: refresh_token=<opaque>; Path=/api/v1/tokens; Max-Age=2592000; HttpOnly; Secure; SameSite=Strict
```

- ### Side effects:

  Verification email sent to the address with a link to `<FRONTEND_BASE_URL>/verify-email?token=<token>`. Link valid **5 minutes**. See [POST /verify-email](#post-verify-email).

- ### Errors:

| Status | Code                           | When                         |
| ------ | ------------------------------ | ---------------------------- |
| 400    | `email_invalid`                | Not a plain address          |
| 400    | `password_too_short`           | Under 8 chars                |
| 400    | `password_too_long`            | Over 64 chars                |
| 400    | `password_contains_whitespace` | Holds whitespace             |
| 409    | `email_taken`                  | Email already has an account |

```json
{
  "code": "password_too_short",
  "message": "password is 5 characters long: expected at least 8",
  "details": null
}
```

```json
{
  "code": "email_taken",
  "message": "email \"jane@example.com\" is already taken: expected an email that does not already have an account",
  "details": null
}
```

---

## POST /signin

- ### Description:

  Sign in with email + password. Answers with the user and a new session. Unverified users can sign in.

- ### Constraints:

  - Email: trimmed + lowercased, must be a plain address.
  - Password: must be non-empty.

- ### Authorization:

  None.

- ### Request:

```typescript
interface SignInRequest {
  email: string;
  password: string;
}
```

Example:

```json
{
  "email": "jane@example.com",
  "password": "correct-horse-battery"
}
```

- ### Response:

  `200 OK`

```typescript
type SignInMethod = "email_password" | "google"; // more may be added (e.g. "apple")

interface SignInResponse {
  id: string; // uuid
  email: string;
  createdAt: string; // ISO 8601
  emailVerifiedAt: string | null; // null → email not verified yet
  signinAt: string; // last sign-in
  firstName: string | null; // null → no profile yet
  lastName: string | null; // null → no profile yet
  signInMethods: SignInMethod[]; // ways this account can sign in; never null, email_password first
  accessToken: string;
}
```

```json
{
  "id": "6f1c2a8e-3b4d-4e5f-9a0b-1c2d3e4f5a6b",
  "email": "jane@example.com",
  "createdAt": "2026-10-01T09:15:00Z",
  "emailVerifiedAt": null,
  "signinAt": "2026-10-17T08:30:12Z",
  "firstName": null,
  "lastName": null,
  "signInMethods": ["email_password"],
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

Also sets the refresh token cookie (30 days) — see [Tokens](./tokens.md#refresh-token-cookie):

```
Set-Cookie: refresh_token=<opaque>; Path=/api/v1/tokens; Max-Age=2592000; HttpOnly; Secure; SameSite=Strict
```

Use `emailVerifiedAt === null` to show a "verify your email" banner, and `firstName === null` to route to profile setup. `signInMethods` without `"email_password"` → Google-only account; offer [POST /add-password](#post-add-password). `signInMethods` without `"google"` → offer [POST /add-google](#post-add-google).

- ### Errors:

| Status | Code                  | When                                |
| ------ | --------------------- | ----------------------------------- |
| 400    | `email_invalid`       | Not a plain address                 |
| 400    | `password_missing`    | Empty password                      |
| 401    | `invalid_credentials` | No account with that email/password |

```json
{
  "code": "invalid_credentials",
  "message": "invalid credentials",
  "details": null
}
```

---

## POST /google-signup

- ### Description:

  Create an account from a Sign in with Google ID token. The frontend renders Google's button (Google Identity Services, client ID = `GOOGLE_CLIENT_ID`) and posts the `credential` it gets back as `idToken`. The server verifies the token's signature, issuer, audience and expiry against Google, then stores a **password-less** account under Google's email. Answers with a session right away, like [POST /signup](#post-signup), but the email is already **verified** and no verification email is sent.

- ### Constraints:

  - `idToken`: trimmed, must be non-empty; must be a Google ID token issued for this app and not expired (Google tokens live about 1 hour; send it straight after the button callback).
  - Google must report the email as verified.
  - The Google account (matched by Google's stable account id, not the email) must not be signed up yet.
  - The email must not belong to any account, including a password one — accounts are never linked automatically.
  - The account has no password: [POST /signin](#post-signin) answers `invalid_credentials` for it, and [POST /change-password](#post-change-password) answers `password_not_set`, until the user adds one through [POST /add-password](#post-add-password).

- ### Authorization:

  None.

- ### Request:

```typescript
interface GoogleSignUpRequest {
  idToken: string; // the `credential` from Google's button callback
}
```

Example:

```json
{
  "idToken": "eyJhbGciOiJSUzI1NiIsImtpZCI6Ij..."
}
```

- ### Response:

  `201 Created`

```typescript
interface GoogleSignUpResponse {
  id: string; // uuid
  email: string; // Google's email, lowercased
  createdAt: string; // ISO 8601
  emailVerifiedAt: string | null; // set at signup for google accounts
  signinAt: string; // signup counts as the first sign-in
  firstName: string | null; // null → no profile yet
  lastName: string | null; // null → no profile yet
  signInMethods: SignInMethod[]; // see SignInMethod in POST /signin
  accessToken: string;
}
```

```json
{
  "id": "6f1c2a8e-3b4d-4e5f-9a0b-1c2d3e4f5a6b",
  "email": "jane@example.com",
  "createdAt": "2026-10-30T09:15:00Z",
  "emailVerifiedAt": "2026-10-30T09:15:00Z",
  "signinAt": "2026-10-30T09:15:00Z",
  "firstName": null,
  "lastName": null,
  "signInMethods": ["google"],
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

Also sets the refresh token cookie, as [POST /signup](#post-signup) does.

- ### Errors:

| Status | Code                      | When                                                         |
| ------ | ------------------------- | ------------------------------------------------------------ |
| 400    | `google_id_token_missing` | `idToken` empty or blank                                     |
| 401    | `google_id_token_invalid` | Token malformed, forged, expired, or issued for another app  |
| 403    | `google_email_unverified` | Google has not verified the account's email                  |
| 409    | `google_account_taken`    | This Google account is already signed up → use google-signin |
| 409    | `email_taken`             | The email belongs to an account (e.g. password signup)       |

```json
{
  "code": "google_account_taken",
  "message": "google account \"109876543210987654321\" is already signed up: expected a google account that is not signed up yet",
  "details": null
}
```

- ### Side effects:
  - Dispatches `user.signed_up`, as [POST /signup](#post-signup) does.

---

## POST /google-signin

- ### Description:

  Sign in with a Sign in with Google ID token (the `credential` from Google's button). The account is matched by Google's stable account id, so it still matches after the user changed their email here. Answers with the user and a new session.

- ### Constraints:

  - `idToken`: same rules as [POST /google-signup](#post-google-signup).
  - Google must report the email as verified.
  - Only accounts created through [POST /google-signup](#post-google-signup) can sign in this way.

- ### Authorization:

  None.

- ### Request:

```typescript
interface GoogleSignInRequest {
  idToken: string; // the `credential` from Google's button callback
}
```

Example:

```json
{
  "idToken": "eyJhbGciOiJSUzI1NiIsImtpZCI6Ij..."
}
```

- ### Response:

  `200 OK`

```typescript
interface GoogleSignInResponse {
  id: string; // uuid
  email: string;
  createdAt: string; // ISO 8601
  emailVerifiedAt: string | null;
  signinAt: string; // this sign-in
  firstName: string | null; // null → no profile yet
  lastName: string | null; // null → no profile yet
  signInMethods: SignInMethod[]; // see SignInMethod in POST /signin
  accessToken: string;
}
```

```json
{
  "id": "6f1c2a8e-3b4d-4e5f-9a0b-1c2d3e4f5a6b",
  "email": "jane@example.com",
  "createdAt": "2026-10-01T09:15:00Z",
  "emailVerifiedAt": "2026-10-01T09:15:00Z",
  "signinAt": "2026-10-30T08:30:12Z",
  "firstName": "Jane",
  "lastName": "Doe",
  "signInMethods": ["google"],
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

Also sets the refresh token cookie, as [POST /signin](#post-signin) does.

- ### Errors:

| Status | Code                        | When                                                                                                        |
| ------ | --------------------------- | ----------------------------------------------------------------------------------------------------------- |
| 400    | `google_id_token_missing`   | `idToken` empty or blank                                                                                    |
| 401    | `google_id_token_invalid`   | Token malformed, forged, expired, or issued for another app                                                 |
| 403    | `google_email_unverified`   | Google has not verified the account's email                                                                 |
| 404    | `google_account_not_found`  | No account for this Google account or its email → offer google-signup                                       |
| 409    | `google_account_not_linked` | The email belongs to a password account → offer password sign-in, then [POST /add-google](#post-add-google) |

```json
{
  "code": "google_account_not_linked",
  "message": "email \"jane@example.com\" is not linked to google: expected the email's account to be signed up with google, not a password",
  "details": null
}
```

- ### Side effects:
  - Dispatches `user.signed_in`, as [POST /signin](#post-signin) does.

---

## GET /google-config

- ### Description:

  Get the Google OAuth client ID the Sign in with Google button must be rendered with. The server is the only source of truth for it; ID tokens issued for any other client ID are rejected by [POST /google-signup](#post-google-signup) and [POST /google-signin](#post-google-signin).

- ### Constraints:

  - Answers `Cache-Control: public, max-age=3600`: the browser serves repeat calls from its HTTP cache for up to 1 hour. After a client ID rotation, browsers may keep the old ID until then.

- ### Authorization:

  None.

- ### Request:

  No body.

- ### Response:

  `200 OK`

```typescript
interface GetGoogleSignInConfigResponse {
  clientId: string; // pass as `client_id` to Google Identity Services
}
```

```json
{
  "clientId": "1234567890-abc123def456.apps.googleusercontent.com"
}
```

- ### Errors:

  None beyond the common ones.

---

## POST /verify-email

- ### Description:

  Redeem an email verification link. The frontend page at `/verify-email` reads `token` from its own URL and forwards it here.

  Same endpoint serves two flows — the server decides by the address the link was issued for:

  - link for the user's **current** email → email marked verified;
  - link for a **new** email (from [POST /change-email](#post-change-email)) → that address becomes the user's email (verified).

- ### Constraints:

  - User must be signed in **as the account the link was issued to**.
  - Link valid **5 minutes**, single use.
  - Idempotent for an already-verified user: re-clicking a used link answers `200` if the email is already verified.

- ### Authorization:

  Bearer access token.

- ### Query Params:

```
?token=<token from the verification link>

example:
POST /api/v1/users/verify-email?token=Xk2b9QeR...
```

- ### Request:

  No body.

- ### Response:

  `200 OK`, empty body.

- ### Errors:

| Status | Code                          | When                                                       |
| ------ | ----------------------------- | ---------------------------------------------------------- |
| 400    | `verification_token_missing`  | No `token` query param                                     |
| 400    | `verification_link_invalid`   | Link used, expired, unknown, or for another purpose        |
| 403    | `verification_link_not_owned` | Link issued to a different account than the signed-in one  |
| 409    | `email_taken`                 | New email was claimed by another account since the request |

```json
{
  "code": "verification_link_invalid",
  "message": "this verification link is not valid or has expired",
  "details": null
}
```

```json
{
  "code": "verification_link_not_owned",
  "message": "this verification link was not issued to the account you are signed in as",
  "details": null
}
```

---

## POST /change-email

- ### Description:

  Request an email change. Nothing changes yet — a verification link is mailed to the **new** address; the change applies once that link is redeemed via [POST /verify-email](#post-verify-email).

- ### Constraints:

  - New email: trimmed + lowercased, plain address, different from the current one, not used by another account.
  - Same verification email rate limits as [POST /messages/verification-email](./messages.md#post-verification-email) (30 s gap and 5 per 24 h per address, 10 per 24 h per user).

- ### Authorization:

  Bearer access token.

- ### Request:

```typescript
interface RequestEmailChangeRequest {
  newEmail: string;
}
```

Example:

```json
{
  "newEmail": "jane.doe@example.com"
}
```

- ### Response:

  `202 Accepted`, empty body.

- ### Side effects:

  Verification email to `newEmail`, link valid **5 minutes**.

- ### Errors:

| Status | Code                              | When                                                                |
| ------ | --------------------------------- | ------------------------------------------------------------------- |
| 400    | `email_invalid`                   | Not a plain address                                                 |
| 400    | `email_unchanged`                 | Same as the current email                                           |
| 409    | `email_taken`                     | Another account already uses it                                     |
| 429    | `verification_email_rate_limited` | Rate limit hit — `details.retryAfterSeconds` + `Retry-After` header |

```json
{
  "code": "email_unchanged",
  "message": "email \"jane@example.com\" is unchanged: expected a new email different from the current one",
  "details": null
}
```

---

## POST /change-password

- ### Description:

  Replace the signed-in user's password after confirming the current one.

- ### Constraints:

  - `currentPassword`: non-empty, must match.
  - `newPassword`: 8–64 chars, no whitespace, different from current.
  - Existing sessions (refresh tokens) stay valid.

- ### Authorization:

  Bearer access token.

- ### Request:

```typescript
interface ChangePasswordRequest {
  currentPassword: string;
  newPassword: string; // 8–64 chars, no whitespace
}
```

Example:

```json
{
  "currentPassword": "correct-horse-battery",
  "newPassword": "tr0ub4dor-and-3"
}
```

- ### Response:

  `200 OK`, empty body.

- ### Errors:

| Status | Code                           | When                                       |
| ------ | ------------------------------ | ------------------------------------------ |
| 400    | `password_missing`             | `currentPassword` empty                    |
| 400    | `password_too_short`           | `newPassword` under 8 chars                |
| 400    | `password_too_long`            | `newPassword` over 64 chars                |
| 400    | `password_contains_whitespace` | `newPassword` holds whitespace             |
| 400    | `password_unchanged`           | New equals current                         |
| 403    | `current_password_incorrect`   | Current password doesn't match             |
| 409    | `password_not_set`             | Google-only account, no password to change |

```json
{
  "code": "current_password_incorrect",
  "message": "current password incorrect: expected the current password to match the stored one",
  "details": null
}
```

---

## POST /add-password

- ### Description:

  Add a password to the signed-in user's Google-only account. Afterwards the account signs in with either [POST /signin](#post-signin) (email + password) or [POST /google-signin](#post-google-signin); the Google connection stays.

- ### Constraints:

  - `password`: 8–64 chars, no whitespace anywhere. Taken verbatim (not trimmed).
  - Account must be signed up with Google and have no password yet — once added, use [POST /change-password](#post-change-password) to replace it.
  - Existing sessions (refresh tokens) stay valid.

- ### Authorization:

  Bearer access token.

- ### Request:

```typescript
interface AddPasswordRequest {
  password: string; // 8–64 chars, no whitespace
}
```

Example:

```json
{
  "password": "correct-horse-battery"
}
```

- ### Response:

  `200 OK`, empty body.

- ### Errors:

| Status | Code                           | When                                  |
| ------ | ------------------------------ | ------------------------------------- |
| 400    | `password_too_short`           | `password` empty or under 8 chars     |
| 400    | `password_too_long`            | `password` over 64 chars              |
| 400    | `password_contains_whitespace` | `password` holds whitespace           |
| 409    | `password_already_set`         | Account already has a password        |
| 409    | `google_account_not_connected` | Account was not signed up with Google |

```json
{
  "code": "password_already_set",
  "message": "user 6f1c2a8e-3b4d-4e5f-9a0b-1c2d3e4f5a6b already has a password: expected an account that has no password yet",
  "details": null
}
```

- ### Side effects:

  - Dispatches `user.password_added`.

---

## POST /reset-password

- ### Description:

  Finish a forgotten-password reset. The frontend page at `/reset-password` reads `token` from its own URL (the link mailed by [POST /messages/password-reset-email](./messages.md#post-password-reset-email)), asks for the new password and sends both here. No sign-in needed — send the user to [POST /signin](#post-signin) afterwards.

- ### Constraints:

  - `token`: trimmed; required.
  - `newPassword`: 8–64 chars, no whitespace anywhere. Taken verbatim (not trimmed).
  - Link valid **15 minutes**, single use.
  - Link is rejected once the account's email changed since it was mailed, or the account was deleted or has no password.
  - Every link rejection answers the same `password_reset_link_invalid`, so it never reveals account state.

- ### Authorization:

  None.

- ### Request:

```typescript
interface ResetPasswordRequest {
  token: string; // from the reset link's ?token=
  newPassword: string; // 8–64 chars, no whitespace
}
```

Example:

```json
{
  "token": "Xk2b9QeR...",
  "newPassword": "correct-horse-battery"
}
```

- ### Response:

  `200 OK`, empty body.

- ### Errors:

| Status | Code                           | When                                                                            |
| ------ | ------------------------------ | ------------------------------------------------------------------------------- |
| 400    | `password_reset_token_missing` | `token` empty or blank                                                          |
| 400    | `password_too_short`           | `newPassword` empty or under 8 chars                                            |
| 400    | `password_too_long`            | `newPassword` over 64 chars                                                     |
| 400    | `password_contains_whitespace` | `newPassword` holds whitespace                                                  |
| 400    | `password_reset_link_invalid`  | Link used, expired, unknown, for another purpose, or no longer fits the account |

```json
{
  "code": "password_reset_link_invalid",
  "message": "this password reset link is not valid or has expired",
  "details": null
}
```

- ### Side effects:

  - Link spent.
  - Every session ends: all the user's refresh tokens are deleted, so other devices must sign in again (access tokens already issued live out their 1 hour).
  - Email not verified yet → marked verified (the link reached the inbox) and `user.email_verified` dispatched.
  - Dispatches `user.password_reset`.

---

## POST /add-google

- ### Description:

  Connect a Google account to the signed-in user's email + password account. Afterwards the account signs in with either [POST /signin](#post-signin) (email + password) or [POST /google-signin](#post-google-signin); the password stays. The frontend renders Google's button and posts the `credential` it gets back as `idToken`, as for [POST /google-signup](#post-google-signup).

- ### Constraints:

  - `idToken`: same rules as [POST /google-signup](#post-google-signup).
  - Google must report the email as verified.
  - Google's email must equal the account's email (case-insensitive) — a Google account under another address can't be connected.
  - Account must have no Google account connected yet.
  - The Google account (matched by Google's stable account id) must not be connected to any other account.
  - If the account's email was still unverified, it becomes verified — Google vouched for the same address.
  - Existing sessions (refresh tokens) stay valid.

- ### Authorization:

  Bearer access token.

- ### Request:

```typescript
interface AddGoogleRequest {
  idToken: string; // the `credential` from Google's button callback
}
```

Example:

```json
{
  "idToken": "eyJhbGciOiJSUzI1NiIsImtpZCI6Ij..."
}
```

- ### Response:

  `200 OK`, empty body. Re-fetch [GET /me](#get-me) for the updated `signInMethods` and `emailVerifiedAt`.

- ### Errors:

| Status | Code                               | When                                                        |
| ------ | ---------------------------------- | ----------------------------------------------------------- |
| 400    | `google_id_token_missing`          | `idToken` empty or blank                                    |
| 401    | `google_id_token_invalid`          | Token malformed, forged, expired, or issued for another app |
| 403    | `google_email_unverified`          | Google has not verified the Google account's email          |
| 409    | `google_account_already_connected` | Account already has a Google account connected              |
| 409    | `google_email_mismatch`            | Google account's email differs from the account's email     |
| 409    | `google_account_taken`             | Google account is already connected to another account      |

```json
{
  "code": "google_email_mismatch",
  "message": "google email \"jane@gmail.com\" does not match user 6f1c2a8e-3b4d-4e5f-9a0b-1c2d3e4f5a6b: expected a google account with the same email as the account",
  "details": null
}
```

- ### Side effects:

  - Dispatches `user.google_account_added`.

---

## GET /me

- ### Description:

  Get the signed-in user's own account, with profile names when a profile exists.

- ### Constraints:

  - Returns the account the access token belongs to.

- ### Authorization:

  Bearer access token.

- ### Request:

  No body.

- ### Response:

  `200 OK`

```typescript
interface GetCurrentUserResponse {
  id: string; // uuid
  email: string;
  createdAt: string; // ISO 8601
  emailVerifiedAt: string | null; // null → email not verified yet
  signinAt: string | null; // last sign-in; null → never signed in
  firstName: string | null; // null → no profile yet
  lastName: string | null; // null → no profile yet
  signInMethods: SignInMethod[]; // see SignInMethod in POST /signin
}
```

```json
{
  "id": "6f1c2a8e-3b4d-4e5f-9a0b-1c2d3e4f5a6b",
  "email": "jane@example.com",
  "createdAt": "2026-10-01T09:15:00Z",
  "emailVerifiedAt": null,
  "signinAt": "2026-10-17T08:30:12Z",
  "firstName": null,
  "lastName": null,
  "signInMethods": ["email_password", "google"]
}
```

- ### Errors:

| Status | Code             | When                                   |
| ------ | ---------------- | -------------------------------------- |
| 404    | `user_not_found` | Account deleted after token was issued |

```json
{
  "code": "user_not_found",
  "message": "user not found",
  "details": null
}
```

---

## DELETE /me

- ### Description:

  Delete the signed-in user's own account. Soft delete: the account and profile are anonymized; the email is freed for a new signup.

- ### Constraints:

  - Deletes the account the access token belongs to.
  - After success the access token stops working (the user no longer exists) — clear local session.

- ### Authorization:

  Bearer access token.

- ### Request:

  No body.

- ### Response:

  `200 OK`, empty body.

- ### Errors:

| Status | Code             | When                 |
| ------ | ---------------- | -------------------- |
| 404    | `user_not_found` | Account already gone |

```json
{
  "code": "user_not_found",
  "message": "user not found",
  "details": null
}
```
