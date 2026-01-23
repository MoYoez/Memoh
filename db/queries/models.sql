-- name: CreateModel :one
INSERT INTO models (model_id, name, base_url, api_key, client_type, dimensions, type)
VALUES (
  sqlc.arg(model_id),
  sqlc.arg(name),
  sqlc.arg(base_url),
  sqlc.arg(api_key),
  sqlc.arg(client_type),
  sqlc.arg(dimensions),
  sqlc.arg(type)
)
RETURNING *;

-- name: GetModelByID :one
SELECT * FROM models WHERE id = sqlc.arg(id);

-- name: GetModelByModelID :one
SELECT * FROM models WHERE model_id = sqlc.arg(model_id);

-- name: ListModels :many
SELECT * FROM models
ORDER BY created_at DESC;

-- name: ListModelsByType :many
SELECT * FROM models
WHERE type = sqlc.arg(type)
ORDER BY created_at DESC;

-- name: ListModelsByClientType :many
SELECT * FROM models
WHERE client_type = sqlc.arg(client_type)
ORDER BY created_at DESC;

-- name: UpdateModel :one
UPDATE models
SET
  name = sqlc.arg(name),
  base_url = sqlc.arg(base_url),
  api_key = sqlc.arg(api_key),
  client_type = sqlc.arg(client_type),
  dimensions = sqlc.arg(dimensions),
  type = sqlc.arg(type),
  updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateModelByModelID :one
UPDATE models
SET
  name = sqlc.arg(name),
  base_url = sqlc.arg(base_url),
  api_key = sqlc.arg(api_key),
  client_type = sqlc.arg(client_type),
  dimensions = sqlc.arg(dimensions),
  type = sqlc.arg(type),
  updated_at = now()
WHERE model_id = sqlc.arg(model_id)
RETURNING *;

-- name: DeleteModel :exec
DELETE FROM models WHERE id = sqlc.arg(id);

-- name: DeleteModelByModelID :exec
DELETE FROM models WHERE model_id = sqlc.arg(model_id);

-- name: CountModels :one
SELECT COUNT(*) FROM models;

-- name: CountModelsByType :one
SELECT COUNT(*) FROM models WHERE type = sqlc.arg(type);

