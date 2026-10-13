-- name: RecordEvent :one
INSERT INTO
  event_log (id, user_id, event_label, event_data, occurred_at)
VALUES
  ($1, $2, $3, $4, NOW())
RETURNING
  *;