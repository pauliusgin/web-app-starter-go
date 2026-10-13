-- name: CreateAuthChallenge :one
INSERT INTO
  auth_challenges (
    id,
    user_id,
    target_type,
    target,
    purpose,
    token_hash,
    code,
    provider,
    reference_id,
    created_at,
    expires_at,
    used_at
  )
VALUES
  (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    NULL
  )
RETURNING
  *;

-- name: MarkAuthChallengeAsUsed :one
UPDATE auth_challenges
SET
  used_at = NOW()
WHERE
  auth_challenges.id = $1
  AND auth_challenges.used_at IS NULL
  AND auth_challenges.expires_at > NOW()
RETURNING
  *;
