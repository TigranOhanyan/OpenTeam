-- name: CreateAction :one
INSERT INTO actions (id, step_id, tool_call_id, name, arguments) VALUES (?, ?, ?, ?, ?) RETURNING *;