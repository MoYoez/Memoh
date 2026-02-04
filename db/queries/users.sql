-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, role, display_name, avatar_url, is_active, data_root)
VALUES (
  sqlc.arg(username),
  sqlc.arg(email),
  sqlc.arg(password_hash),
  sqlc.arg(role)::user_role,
  sqlc.arg(display_name),
  sqlc.arg(avatar_url),
  sqlc.arg(is_active),
  sqlc.arg(data_root)
)
RETURNING *;

-- name: UpsertUserByUsername :one
INSERT INTO users (username, email, password_hash, role, display_name, avatar_url, is_active, data_root)
VALUES (
  sqlc.arg(username),
  sqlc.arg(email),
  sqlc.arg(password_hash),
  sqlc.arg(role)::user_role,
  sqlc.arg(display_name),
  sqlc.arg(avatar_url),
  sqlc.arg(is_active),
  sqlc.arg(data_root)
)
ON CONFLICT (username) DO UPDATE SET
  email = EXCLUDED.email,
  password_hash = EXCLUDED.password_hash,
  role = EXCLUDED.role,
  display_name = EXCLUDED.display_name,
  avatar_url = EXCLUDED.avatar_url,
  is_active = EXCLUDED.is_active,
  data_root = EXCLUDED.data_root,
  updated_at = now()
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = sqlc.arg(username);

-- name: GetUserByIdentity :one
SELECT * FROM users WHERE username = sqlc.arg(identity) OR email = sqlc.arg(identity);

-- name: GetUserByID :one
SELECT * FROM users WHERE id = sqlc.arg(id);

-- name: CreateUserWithID :one
INSERT INTO users (id, username, email, password_hash, role, display_name, avatar_url, is_active, data_root)
VALUES (
  sqlc.arg(id),
  sqlc.arg(username),
  sqlc.arg(email),
  sqlc.arg(password_hash),
  sqlc.arg(role)::user_role,
  sqlc.arg(display_name),
  sqlc.arg(avatar_url),
  sqlc.arg(is_active),
  sqlc.arg(data_root)
)
RETURNING *;

-- name: CountUsers :one
SELECT COUNT(*)::bigint AS count FROM users;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC;


-- name: UpdateUserProfile :one
UPDATE users
SET display_name = $2,
    avatar_url = $3,
    is_active = $4,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateUserAdmin :one
UPDATE users
SET role = sqlc.arg(role)::user_role,
    display_name = sqlc.arg(display_name),
    avatar_url = sqlc.arg(avatar_url),
    is_active = sqlc.arg(is_active),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateUserLastLogin :one
UPDATE users
SET last_login_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING *;

