-- name: CreateAction :one
INSERT INTO actions (id, turn_id, tool_call_id, name, arguments) VALUES (?, ?, ?, ?, ?) RETURNING *;