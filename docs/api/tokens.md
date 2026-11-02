# baseUrl/api/v1/tokens

Session management. Both endpoints read the **refresh token** from an httpOnly cookie — no `Authorization` header, no body.

### Refresh token cookie

Set by signup, signin and `POST /refresh`; cleared by `POST /revoke` and any `refresh_token_invalid` answer.

```
Set-Cookie: refresh_token=<opaque>; Path=/api/v1/tokens; Max-Age=2592000; HttpOnly; Secure; SameSite=Strict
```

- `HttpOnly` → JS can't read it; don't try to store or send it yourself.
- `Path=/api/v1/tokens` → the browser sends it only to these two endpoints.
- `SameSite=Strict` + no CORS → the frontend must call the API **same-origin**. Vite dev: proxy `/api` → `http://localhost:3000` in `vite.config.ts`; plain `fetch` (default `credentials: "same-origin"`) then sends the cookie. Cross-origin calls are not supported.
- `Secure` → browsers still accept it on `http://localhost`.

| Token         | Lifetime | Used for                             |
| ------------- | -------- | ------------------------------------ |
| Access token  | 1 hour   | Every protected endpoint             |
| Refresh token | 30 days  | `POST /refresh`, `POST /revoke` only |

---

## POST /refresh

- ### Description:

  Exchange the refresh token cookie for a new access token, rotating the cookie. Call it on app load (to restore a session) and when a protected request answers `401 access_token_expired`, then retry that request.

- ### Constraints:

  - **Rotation:** the presented refresh token is spent; the response sets a replacement cookie. Replaying the old one answers `401 refresh_token_invalid`.
  - Several tabs/requests refreshing concurrently with the same token will race — only one wins. Serialize refreshes in the client (single in-flight refresh promise).

- ### Authorization:

  `refresh_token` cookie.

- ### Request:

  No body.

- ### Response:

  `200 OK`, plus `Set-Cookie` with the rotated refresh token.

```typescript
interface RefreshTokenResponse {
  accessToken: string; // JWT, 1 hour
}
```

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

- ### Errors:

| Status | Code                    | When                                                                                 |
| ------ | ----------------------- | ------------------------------------------------------------------------------------ |
| 401    | `refresh_token_invalid` | Cookie missing, token unknown, expired, already used or revoked, or the user is gone |

All rejections share one code on purpose and clear the cookie. On this error → drop the session and send the user to sign-in.

```json
{
  "code": "refresh_token_invalid",
  "message": "invalid refresh token",
  "details": null
}
```

---

## POST /revoke

- ### Description:

  Log out: revoke the refresh token the cookie carries and clear the cookie. The current access token keeps working until it expires (≤ 1 hour) — drop it client-side.

- ### Constraints:

  - Idempotent and forgiving: a missing cookie or an unknown/already-revoked token still answers `200` (and clears the cookie). Logout never fails from the client's point of view.

- ### Authorization:

  `refresh_token` cookie (optional — see above).

- ### Request:

  No body.

- ### Response:

  `200 OK`, empty body, `Set-Cookie` clearing `refresh_token`.

- ### Errors:

  Only `500 internal_error` on a server failure.
