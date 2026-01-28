-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, role, display_name, avatar_url, is_active, data_root)
VALUES (
  sqlc.arg(username),
  sqlc.arg(email),
  sqlc.arg(password_hash),
  sqlc.arg(role),
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
  sqlc.arg(role),
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

-- name: CountUsers :one
SELECT COUNT(*)::bigint AS count FROM users;
