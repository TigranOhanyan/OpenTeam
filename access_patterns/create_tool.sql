-- name: CreateTool :one
INSERT INTO tools (id, duty_id, name, description, parameters) VALUES (?, ?, ?, ?, ?) RETURNING *;