-- name: CreateMessage :one
INSERT INTO messages (id, turn_id, channel_name, role_id, duty_id, visibility, openai_message) 
VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING *;