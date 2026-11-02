-- name: SaveUpload :one
INSERT INTO
  uploads (
    id,
    user_id,
    purpose,
    kind,
    mime_type,
    size_bytes,
    width,
    height,
    original_path,
    original_url,
    preview_path,
    preview_url,
    created_at,
    updated_at,
    attached_at
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
    $12,
    NOW(),
    NOW(),
    NULL
  )
RETURNING
  *;

-- name: ListUnusedPendingUploads :many
SELECT
  u.id,
  u.user_id,
  u.purpose,
  u.kind,
  u.mime_type,
  u.size_bytes,
  u.width,
  u.height,
  u.original_path,
  u.original_url,
  u.preview_path,
  u.preview_url,
  u.created_at,
  u.updated_at,
  u.attached_at
FROM
  uploads u
WHERE
  u.attached_at IS NULL
  AND u.created_at < $1
ORDER BY
  u.created_at;

-- name: DeleteUploadByID :exec
DELETE FROM uploads u
WHERE
  u.id = $1;

-- name: AttachUpload :one
UPDATE uploads
SET
  attached_at = NOW(),
  updated_at = NOW()
WHERE
  uploads.id = $1
  AND uploads.user_id = $2
  AND uploads.purpose = $3
RETURNING
  *;

-- name: FindUnreferencedUploadByID :one
SELECT
  u.id,
  u.user_id,
  u.purpose,
  u.kind,
  u.mime_type,
  u.size_bytes,
  u.width,
  u.height,
  u.original_path,
  u.original_url,
  u.preview_path,
  u.preview_url,
  u.created_at,
  u.updated_at,
  u.attached_at
FROM
  uploads u
WHERE
  u.id = $1
  AND NOT EXISTS (
    SELECT
      1
    FROM
      upload_references r
    WHERE
      r.upload_id = u.id
  );

-- name: ListOrphanedAttachedUploads :many
SELECT
  u.id,
  u.user_id,
  u.purpose,
  u.kind,
  u.mime_type,
  u.size_bytes,
  u.width,
  u.height,
  u.original_path,
  u.original_url,
  u.preview_path,
  u.preview_url,
  u.created_at,
  u.updated_at,
  u.attached_at
FROM
  uploads u
WHERE
  u.attached_at IS NOT NULL
  AND u.attached_at < sqlc.arg(attached_before)::TIMESTAMPTZ
  AND NOT EXISTS (
    SELECT
      1
    FROM
      upload_references r
    WHERE
      r.upload_id = u.id
  )
ORDER BY
  u.attached_at;
