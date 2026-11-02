# Conventions

Everything here applies to every endpoint unless the endpoint's page says otherwise.

---

## Base URL

All resource routes live under `baseUrl/api/v1`. The only exception is the liveness probe at `baseUrl/api/health`.

Dev default: `http://localhost:3000`.

---

## Request bodies

- JSON endpoints require `Content-Type: application/json` (charset parameter allowed).
- Body must hold **exactly one** JSON object — trailing data after it is rejected.
- **Unknown fields are rejected** — sending a key the endpoint doesn't declare answers `400 request_body_invalid`. Don't send extra fields "just in case".
- Body limit: **64 KiB** for JSON endpoints. Uploads have their own limit (see [Uploads](./uploads.md)).
- String inputs are trimmed server-side; emails are also lowercased. Responses return the stored (trimmed) values.

Common decoding errors (every JSON endpoint can answer these — they are not repeated on each page):

| Status | Code                       | When                                                                       |
| ------ | -------------------------- | -------------------------------------------------------------------------- |
| 400    | `request_body_invalid`     | Empty body, malformed JSON, wrong field type, unknown field, trailing data |
| 413    | `request_body_too_large`   | Body over 64 KiB                                                           |
| 415    | `unsupported_content_type` | `Content-Type` is not `application/json`                                   |

---

## Authorization

Protected endpoints require an access token:

```
Authorization: Bearer <accessToken>
```

- Scheme is case-insensitive (`bearer` works); exactly one space; token must contain no whitespace.
- Access token (JWT) is valid for **1 hour**.
- The token must name a user that still exists — a deleted account's tokens stop working immediately.

Auth errors (every protected endpoint can answer these — not repeated per page):

| Status | Code                   | When                                                     | Client action                   |
| ------ | ---------------------- | -------------------------------------------------------- | ------------------------------- |
| 401    | `access_token_missing` | No `Authorization` header                                | Sign in                         |
| 401    | `access_token_expired` | Token past its expiry                                    | Refresh, then retry the request |
| 401    | `access_token_invalid` | Wrong scheme, malformed header, bad signature, user gone | Sign in                         |

401 responses also carry a `WWW-Authenticate` header (`Bearer`, `Bearer error="invalid_token"`, …).

```json
{
  "code": "access_token_expired",
  "message": "the access token expired",
  "details": null
}
```

The refresh/revoke endpoints don't use this header — they read the refresh token from the `refresh_token` httpOnly cookie. See [Tokens](./tokens.md).

---

## Error response

Every failed request (except unmatched routes, see below) answers JSON in this shape:

```typescript
interface ErrorResponse {
  code: ErrorCode; // stable, snake_case — branch on this
  message: string; // human-readable, for logs/debugging — may change, never branch on it
  details: Record<string, unknown> | null; // null unless the code documents details (e.g. verification_email_rate_limited)
}
```

```json
{
  "code": "email_taken",
  "message": "email \"jane@example.com\" is already taken: expected an email that does not already have an account",
  "details": null
}
```

Messages in these docs are illustrative — only `code` is a contract.

### 500

Any unexpected server failure answers `500` with `internal_error` and a generic message (the cause is logged server-side, never sent):

```json
{
  "code": "internal_error",
  "message": "could not create the profile",
  "details": null
}
```

Any endpoint can answer this; it isn't repeated per page.

### Unmatched routes

Unknown path → `404`, known path with wrong method → `405`. These are Go's **plain-text** responses (`404 page not found`), not JSON. Treat them as client bugs.

---

## `null` in responses

Every response field is always present. A missing value is an explicit `null` — never a dropped key, never `""`, never a zero date. Optional fields are typed `T | null` in these docs.

---

## PATCH semantics

No endpoint uses PATCH yet. When one is added (decode its body fields with `handlers.PatchField`), it tells three states apart per field:

| Request                 | Meaning                            |
| ----------------------- | ---------------------------------- |
| key absent              | leave the stored value as is       |
| `"field": null`         | clear the stored value             |
| `"field": "value"`      | set it (trimmed)                   |
| `"field": ""` / `"   "` | **rejected** — use `null` to clear |

For nested objects: `null` clears every field inside; an object applies the same rules to each inner key.

Fields marked required on that endpoint must be sent on every PATCH.

---

## Pagination

No list endpoint ships yet. List endpoints (built on `internal/pagination` and `handlers` page query params) take two optional query params:

| Param   | Default | Allowed             | Rejection           |
| ------- | ------- | ------------------- | ------------------- |
| `page`  | 1       | whole number 1–1000 | `400 page_invalid`  |
| `limit` | 20      | whole number 1–100  | `400 limit_invalid` |

Empty value (`?page=`) counts as absent. Paginated responses wrap items:

```typescript
interface PaginatedResponse<T> {
  data: T[]; // never null; [] past the last page
  metadata: PageMetadata;
}

interface PageMetadata {
  total: number; // items across all pages
  page: number; // echoes the page served
  limit: number; // echoes the page size served
  totalPages: number; // 0 when total is 0
}
```

---

## Path IDs

All IDs are UUIDs. A path id that doesn't parse answers `400 <resource>_id_malformed`. A well-formed id that names nothing **you own** answers `404 <resource>_not_found` — ownership failures and missing rows look the same on purpose.

---

## Error codes

Full list (source: `internal/api/http/handlers/errorcode.go`). Grouped by where they come from.

| Code                                | Status | Meaning                                                                                     |
| ----------------------------------- | ------ | ------------------------------------------------------------------------------------------- |
| **General**                         |        |                                                                                             |
| `internal_error`                    | 500    | Unexpected server failure                                                                   |
| `request_body_invalid`              | 400    | Body empty, malformed, wrong shape, unknown field; or unreadable multipart                  |
| `request_body_too_large`            | 413    | Body over the size limit                                                                    |
| `unsupported_content_type`          | 415    | Wrong `Content-Type`                                                                        |
| `page_invalid`                      | 400    | `page` query param not a whole number from 1 to 1000                                        |
| `limit_invalid`                     | 400    | `limit` query param not a whole number from 1 to 100                                        |
| **Auth**                            |        |                                                                                             |
| `access_token_missing`              | 401    | No `Authorization` header                                                                   |
| `access_token_expired`              | 401    | Access token expired                                                                        |
| `access_token_invalid`              | 401    | Any other access token rejection                                                            |
| `refresh_token_invalid`             | 401    | Any refresh token rejection                                                                 |
| `invalid_credentials`               | 401    | Sign-in email/password match no account (also: password sign-in to a Google-only account)   |
| **Google sign-in**                  |        |                                                                                             |
| `google_id_token_missing`           | 400    | Google signup/sign-in without an `idToken`                                                  |
| `google_id_token_invalid`           | 401    | Google ID token malformed, forged, expired, or issued for another app                       |
| `google_email_unverified`           | 403    | Google has not verified the account's email                                                 |
| `google_account_taken`              | 409    | Google signup or addition for a Google account another account already holds                |
| `google_account_not_found`          | 404    | Google sign-in for a Google account and email with no account                               |
| `google_account_not_linked`         | 409    | Google sign-in whose email belongs to a password account                                    |
| **Email / password input**          |        |                                                                                             |
| `email_invalid`                     | 400    | Not a plain address (`a@b.com`, no display name)                                            |
| `email_taken`                       | 409    | Email already belongs to an account                                                         |
| `email_unchanged`                   | 400    | Email change to the address the account already has                                         |
| `password_missing`                  | 400    | Password empty                                                                              |
| `password_too_short`                | 400    | Under 8 characters                                                                          |
| `password_too_long`                 | 400    | Over 64 characters                                                                          |
| `password_contains_whitespace`      | 400    | Holds a space, tab or line break                                                            |
| `password_unchanged`                | 400    | New password equals current                                                                 |
| `current_password_incorrect`        | 403    | Current password doesn't match                                                              |
| `password_not_set`                  | 409    | Password change for a Google-only account                                                   |
| `password_already_set`              | 409    | Password addition for an account that already has a password                                |
| `google_account_not_connected`      | 409    | Password addition for an account not signed up with Google                                  |
| `google_account_already_connected`  | 409    | Google account addition for an account that already has one                                 |
| `google_email_mismatch`             | 409    | Google account addition whose Google email differs from the account's                       |
| **Email verification**              |        |                                                                                             |
| `verification_token_missing`        | 400    | No `token` query param                                                                      |
| `verification_link_invalid`         | 400    | Link used, expired, or for another purpose                                                  |
| `verification_link_not_owned`       | 403    | Link issued to a different account                                                          |
| `email_already_verified`            | 409    | Verification email resend for an already verified email                                     |
| `verification_email_rate_limited`   | 429    | Verification email asked for too soon; `details.retryAfterSeconds` + `Retry-After` header   |
| **Password reset**                  |        |                                                                                             |
| `password_reset_token_missing`      | 400    | Password reset without a `token`                                                            |
| `password_reset_link_invalid`       | 400    | Reset link used, expired, unknown, for another purpose, or no longer fits the account       |
| **Users**                           |        |                                                                                             |
| `user_not_found`                    | 404    | User doesn't exist                                                                          |
| **Profiles**                        |        |                                                                                             |
| `first_name_missing`                | 400    | First name empty/blank                                                                      |
| `last_name_missing`                 | 400    | Last name empty/blank                                                                       |
| `first_name_invalid`                | 400    | First name not Latin/Cyrillic letters joined by single spaces/dashes, or over 50 characters |
| `last_name_invalid`                 | 400    | Last name not Latin/Cyrillic letters joined by single spaces/dashes, or over 50 characters  |
| `profile_exists`                    | 409    | Profile already created                                                                     |
| `profile_not_found`                 | 404    | No profile yet — create it first                                                            |
| `profile_picture_upload_invalid`    | 400    | Picture upload id malformed, unknown, not yours, or not a `profile_picture` upload          |
| **Uploads**                         |        |                                                                                             |
| `upload_purpose_invalid`            | 400    | `purpose` missing or unknown                                                                |
| `upload_files_missing`              | 400    | No file sent                                                                                |
| `upload_too_many_files`             | 400    | More files than the purpose allows                                                          |
| `upload_file_type_unsupported`      | 400    | Format not accepted at all, or not by this purpose                                          |
| `upload_file_too_large`             | 413    | File over 5 MiB                                                                             |
| `upload_image_dimensions_too_large` | 400    | Image wider or taller than 10000 px                                                         |
| `upload_file_unreadable`            | 400    | Image content can't be decoded                                                              |
