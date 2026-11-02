-- name: GetUploadReadModelByID :one
SELECT
  u.id,
  u.user_id,
  u.purpose,
  u.kind,
  u.mime_type,
  u.original_url,
  u.preview_url
FROM
  uploads u
WHERE
  u.id = $1
  AND u.user_id = $2;
