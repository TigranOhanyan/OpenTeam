-- name: CreateTool :one
INSERT INTO tools (id, task_id, name, description, parameters) VALUES (?, ?, ?, ?, ?) RETURNING *;