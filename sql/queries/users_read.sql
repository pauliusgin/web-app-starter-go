-- name: FindUserReadModelByID :one
SELECT
  u.id,
  u.email,
  u.created_at,
  u.email_verified_at,
  u.signin_at,
  p.first_name,
  p.last_name
FROM
  users u
  LEFT JOIN profiles p ON p.id = u.id
  AND p.deleted_at IS NULL
WHERE
  u.id = $1
  AND u.deleted_at IS NULL;
