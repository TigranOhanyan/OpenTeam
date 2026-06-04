-- name: CreateAction :one
INSERT INTO actions (id, run_id, step_id, tool_call_id, tool_call) VALUES (?, ?, ?, ?, ?) RETURNING *;