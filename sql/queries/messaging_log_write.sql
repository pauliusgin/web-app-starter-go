-- name: SaveMessage :one
INSERT INTO
  messaging_log (
    id,
    recipient_id,
    recipient_email,
    data,
    type,
    method,
    status,
    reference_id,
    errors,
    created_at
  )
VALUES
  ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
RETURNING
  *;
