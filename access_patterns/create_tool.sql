-- name: CreateTool :one
INSERT INTO tools (id, task_id, tool) VALUES (?, ?, ?) RETURNING *;