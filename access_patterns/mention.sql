-- name: CreateMention :one
INSERT INTO mentions (execution_id, message_id, from_member_role_id, to_member_name, message) 
VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: GetMention :one
SELECT * FROM mentions WHERE execution_id = ? LIMIT 1;
