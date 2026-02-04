-- name: CreateBot :one
INSERT INTO bots (owner_user_id, type, display_name, avatar_url, is_active, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, owner_user_id, type, display_name, avatar_url, is_active, metadata, created_at, updated_at;

-- name: GetBotByID :one
SELECT id, owner_user_id, type, display_name, avatar_url, is_active, metadata, created_at, updated_at
FROM bots
WHERE id = $1;

-- name: ListBotsByOwner :many
SELECT id, owner_user_id, type, display_name, avatar_url, is_active, metadata, created_at, updated_at
FROM bots
WHERE owner_user_id = $1
ORDER BY created_at DESC;

-- name: ListBotsByMember :many
SELECT b.id, b.owner_user_id, b.type, b.display_name, b.avatar_url, b.is_active, b.metadata, b.created_at, b.updated_at
FROM bots b
JOIN bot_members m ON m.bot_id = b.id
WHERE m.user_id = $1
ORDER BY b.created_at DESC;

-- name: UpdateBotProfile :one
UPDATE bots
SET display_name = $2,
    avatar_url = $3,
    is_active = $4,
    metadata = $5,
    updated_at = now()
WHERE id = $1
RETURNING id, owner_user_id, type, display_name, avatar_url, is_active, metadata, created_at, updated_at;

-- name: UpdateBotOwner :one
UPDATE bots
SET owner_user_id = $2,
    updated_at = now()
WHERE id = $1
RETURNING id, owner_user_id, type, display_name, avatar_url, is_active, metadata, created_at, updated_at;

-- name: DeleteBotByID :exec
DELETE FROM bots WHERE id = $1;

-- name: UpsertBotMember :one
INSERT INTO bot_members (bot_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (bot_id, user_id) DO UPDATE SET
  role = EXCLUDED.role
RETURNING bot_id, user_id, role, created_at;

-- name: ListBotMembers :many
SELECT bot_id, user_id, role, created_at
FROM bot_members
WHERE bot_id = $1
ORDER BY created_at DESC;

-- name: GetBotMember :one
SELECT bot_id, user_id, role, created_at
FROM bot_members
WHERE bot_id = $1 AND user_id = $2
LIMIT 1;

-- name: DeleteBotMember :exec
DELETE FROM bot_members WHERE bot_id = $1 AND user_id = $2;
