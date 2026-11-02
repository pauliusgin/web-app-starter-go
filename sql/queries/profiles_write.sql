-- name: SaveProfile :one
INSERT INTO
  profiles (
    id,
    first_name,
    last_name,
    picture_upload_id,
    created_at,
    updated_at
  )
VALUES
  ($1, $2, $3, $4, NOW(), NOW())
RETURNING
  *;

-- name: UpdateProfile :one
UPDATE profiles
SET
  first_name = $2,
  last_name = $3,
  picture_upload_id = $4,
  updated_at = NOW()
WHERE
  profiles.id = $1
  AND profiles.deleted_at IS NULL
RETURNING
  *;
