# baseUrl/api/v1/messages

Common decoding, auth and 500 errors are listed in [Conventions](./conventions.md) and not repeated here.

---

## POST /verification-email

- ### Description:

  Resend the email verification link to the caller's **current** email — for when the previous link expired (links live **5 minutes**) or never arrived. The recipient is read from the access token's user; nothing is sent in the body.

- ### Constraints:

  - Current email must still be unverified.
  - Rate limits are per user and count every verification email that user triggered, including the one mailed at signup and those from [POST /users/change-email](./users.md#post-change-email):
    - Same address: at least **30 seconds** between emails.
    - Same address: at most **5** emails in any rolling **24 hours**.
    - Same user: at most **10** emails in any rolling **24 hours** (across all addresses).

- ### Authorization:

  Bearer access token.

- ### Request:

  No body.

- ### Response:

  `200 OK`, empty body.

- ### Side effects:

  Verification email to the current address, link valid **5 minutes**. Earlier links stay valid until they expire.

- ### Errors:

| Status | Code                              | When                                                |
| ------ | --------------------------------- | --------------------------------------------------- |
| 404    | `user_not_found`                  | Account no longer exists                            |
| 409    | `email_already_verified`          | Current email is already verified                   |
| 429    | `verification_email_rate_limited` | A rate limit above is hit — see `retryAfterSeconds` |

On `429`, `details.retryAfterSeconds` (integer, rounded up, ≥ 1) says when the next email is allowed; the `Retry-After` header carries the same number.

```typescript
interface VerificationEmailRateLimitedDetails {
  retryAfterSeconds: number;
}
```

```json
{
  "code": "verification_email_rate_limited",
  "message": "too many verification emails were requested, try again later",
  "details": {
    "retryAfterSeconds": 21
  }
}
```

```json
{
  "code": "email_already_verified",
  "message": "email \"jane@example.com\" is already verified: expected an email that is not verified yet",
  "details": null
}
```

---

## POST /password-reset-email

- ### Description:

  Start a forgotten-password reset: mail a reset link to the account behind `email`. The link opens `<FRONTEND_BASE_URL>/reset-password?token=<token>`; that page collects the new password and calls [POST /users/reset-password](./users.md#post-reset-password).

- ### Constraints:

  - Answers `200` whether or not the email has an account — the response never reveals it. No email goes out when:
    - no account holds the email;
    - the account has no password (Google-only — use Google sign-in instead);
    - a rate limit below is hit.
  - Every answer past input validation takes **at least 1 second**, so timing does not reveal it either.
  - `email` trimmed and lowercased.
  - Rate limits are per account and count only password reset emails:
    - at least **30 seconds** between emails;
    - at most **5** emails in any rolling **24 hours**.

- ### Authorization:

  None.

- ### Request:

```typescript
interface PasswordResetEmailRequest {
  email: string;
}
```

Example:

```json
{
  "email": "jane@example.com"
}
```

- ### Response:

  `200 OK`, empty body.

- ### Side effects:

  Password reset email to the account's email, link valid **15 minutes**, single use. Earlier links stay valid until they expire or one of them is used.

- ### Errors:

| Status | Code            | When                        |
| ------ | --------------- | --------------------------- |
| 400    | `email_invalid` | `email` not a plain address |

```json
{
  "code": "email_invalid",
  "message": "invalid email \"jane\": expected a plain address such as accountant@example.com, without a display name",
  "details": null
}
```
