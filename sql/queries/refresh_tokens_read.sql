-- name: FindRefreshTokenByTokenHash :one
SELECT
  rt.id,
  rt.user_id,
  rt.token_hash,
  rt.created_at,
  rt.expires_at,
  rt.revoked_at
FROM
  refresh_tokens rt
WHERE
  rt.token_hash = $1
  AND rt.revoked_at IS NULL
  AND rt.expires_at > now();
