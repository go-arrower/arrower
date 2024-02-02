-- name: UpsertSetting :exec
INSERT INTO arrower.setting (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET updated_at = NOW(),
                                value      = $2;

-- name: GetSetting :one
SELECT value
FROM arrower.setting
WHERE key = $1
LIMIT 1;

-- name: GetSettings :many
SELECT key, value
FROM arrower.setting
WHERE key = ANY (sqlc.slice('composite_keys')::TEXT[]);

-- name: DeleteSetting :exec
DELETE
FROM arrower.setting
WHERE key = $1;