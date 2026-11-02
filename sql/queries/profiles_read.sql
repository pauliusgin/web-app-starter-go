-- name: GetProfileReadModelByID :one
SELECT
  p.id,
  p.first_name,
  p.last_name
FROM
  profiles p
WHERE
  p.id = $1
  AND p.deleted_at IS NULL;
