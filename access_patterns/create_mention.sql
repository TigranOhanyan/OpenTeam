-- name: CreateMention :one
INSERT INTO mentions (id, step_id, from_member_task_id, to_member_name, tool_call_id, message) 
VALUES (?, ?, ?, ?, ?, ?) RETURNING *;