-- name: GetProfileReadModelByID :one
SELECT
  p.id,
  p.first_name,
  p.last_name,
  p.picture_upload_id,
  up.preview_url AS picture_preview_url,
  up.original_url AS picture_original_url
FROM
  profiles p
  LEFT JOIN uploads up ON up.id = p.picture_upload_id
WHERE
  p.id = $1
  AND p.deleted_at IS NULL;
