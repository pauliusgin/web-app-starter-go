-- name: CreateUser :one
INSERT INTO
  users (
    id,
    email,
    password_hash,
    created_at,
    updated_at,
    signin_at
  )
VALUES
  ($1, $2, $3, NOW(), NOW(), NOW())
RETURNING
  id,
  email,
  created_at,
  updated_at,
  signin_at;

-- name: FindUserByID :one
SELECT
  u.id,
  u.email,
  u.created_at,
  u.updated_at,
  u.email_verified_at,
  u.signin_at
FROM
  users u
WHERE
  u.id = $1
  AND u.deleted_at IS NULL;

-- name: FindUserByEmail :one
SELECT
  u.id,
  u.email,
  u.created_at,
  u.updated_at,
  u.email_verified_at,
  u.signin_at
FROM
  users u
WHERE
  u.email = $1
  AND u.deleted_at IS NULL;

-- name: MarkUserEmailAsVerified :one
UPDATE users
SET
  email_verified_at = NOW(),
  updated_at = NOW()
WHERE
  users.id = $1
RETURNING
  id,
  email,
  created_at,
  updated_at,
  email_verified_at,
  signin_at;

-- name: FindUserCredentialsByEmail :one
SELECT
  u.id,
  u.email,
  u.password_hash
FROM
  users u
WHERE
  u.email = $1
  AND u.deleted_at IS NULL;

-- name: FindUserCredentialsByID :one
SELECT
  u.id,
  u.email,
  u.password_hash
FROM
  users u
WHERE
  u.id = $1
  AND u.deleted_at IS NULL;

-- name: MarkUserSignedIn :one
UPDATE users
SET
  signin_at = NOW(),
  updated_at = NOW()
WHERE
  users.id = $1
  AND users.deleted_at IS NULL
RETURNING
  id,
  email,
  created_at,
  updated_at,
  email_verified_at,
  signin_at;

-- name: SoftDeleteUser :one
WITH
  deleted_user AS (
    UPDATE users
    SET
      email = $2,
      deleted_at = NOW(),
      updated_at = NOW()
    WHERE
      users.id = $1
      AND users.deleted_at IS NULL
    RETURNING
      users.id
  ),
  anonymized_profile AS (
    UPDATE profiles
    SET
      first_name = $3,
      last_name = $4,
      deleted_at = NOW(),
      updated_at = NOW()
    WHERE
      profiles.id IN (
        SELECT
          deleted_user.id
        FROM
          deleted_user
      )
      AND profiles.deleted_at IS NULL
  )
SELECT
  deleted_user.id
FROM
  deleted_user;


-- name: ChangeUserEmail :one
UPDATE users
SET
  email = $2,
  email_verified_at = NOW(),
  updated_at = NOW()
WHERE
  users.id = $1
  AND users.deleted_at IS NULL
RETURNING
  id,
  email,
  created_at,
  updated_at,
  email_verified_at,
  signin_at;

-- name: ChangeUserPassword :one
UPDATE users
SET
  password_hash = $2,
  updated_at = NOW()
WHERE
  users.id = $1
  AND users.deleted_at IS NULL
RETURNING
  id;
