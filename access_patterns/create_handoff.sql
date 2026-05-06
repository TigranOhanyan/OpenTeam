-- name: CreateHandoff :one
INSERT INTO handoffs (id, turn_id, to_agent, tool_call_id) VALUES (?, ?, ?, ?) RETURNING *;