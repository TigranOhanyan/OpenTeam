-- name: CreateMention :one
INSERT INTO mentions (id, run_id, message_id, from_member_role_id, to_member_name, message) 
VALUES (?, ?, ?, ?, ?, ?) RETURNING *;