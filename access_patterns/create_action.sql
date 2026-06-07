-- name: CreateAction :one
INSERT INTO actions (id, run_id, tool_call) VALUES (?, ?, ?) RETURNING *;