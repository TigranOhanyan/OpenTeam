-- name: CreateHandoff :one
INSERT INTO handoffs (id, step_id, to_agent, tool_call_id) VALUES (?, ?, ?, ?) RETURNING *;