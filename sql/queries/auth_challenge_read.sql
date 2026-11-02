-- name: FindAuthChallengeByTokenHash :one
SELECT
  ac.id,
  ac.user_id,
  ac.target_type,
  ac.target,
  ac.purpose,
  ac.token_hash,
  ac.code,
  ac.provider,
  ac.reference_id,
  ac.created_at,
  ac.expires_at,
  ac.used_at
FROM
  auth_challenges ac
WHERE
  ac.token_hash = $1
  AND ac.used_at IS NULL
  AND ac.expires_at > now();

-- name: ListAuthChallengeCreatedAtByTarget :many
SELECT
  ac.created_at
FROM
  auth_challenges ac
WHERE
  ac.user_id = $1
  AND ac.target = $2
  AND ac.purpose = $3
  AND ac.created_at > sqlc.arg(created_after)
ORDER BY
  ac.created_at;

-- name: ListAuthChallengeCreatedAtByUser :many
SELECT
  ac.created_at
FROM
  auth_challenges ac
WHERE
  ac.user_id = $1
  AND ac.purpose = $2
  AND ac.created_at > sqlc.arg(created_after)
ORDER BY
  ac.created_at;
