-- name: CreateMessage :one
INSERT INTO messages (id, step_id, channel_name, role_id, task_id, visibility, openai_message) 
VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING *;