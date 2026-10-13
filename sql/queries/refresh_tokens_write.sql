-- name: CreateRefreshToken :one
INSERT INTO
  refresh_tokens (
    id,
    user_id,
    token_hash,
    created_at,
    expires_at,
    revoked_at
  )
VALUES
  ($1, $2, $3, $4, $5, NULL)
RETURNING
  *;

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens
SET
  revoked_at = NOW()
WHERE
  refresh_tokens.token_hash = $1
  AND refresh_tokens.revoked_at IS NULL
RETURNING
  *;

-- name: DeleteAllRefreshTokens :exec
DELETE FROM refresh_tokens rt
WHERE
  rt.user_id = $1;

-- name: DeleteUnusedRefreshTokens :execrows
DELETE FROM refresh_tokens
WHERE
  expires_at < NOW()
  OR revoked_at < NOW() - (sqlc.arg(days)::int * INTERVAL '1 day');