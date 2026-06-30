-- name: CreateTool :one
INSERT INTO tools (id, task_id, tool) VALUES (?, ?, ?) RETURNING *;

-- name: GetToolsByTask :many
SELECT * FROM tools WHERE task_id = ?;