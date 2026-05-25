-- name: CreateMention :one
INSERT INTO mentions (id, run_id, from_member_task_id, to_member_name, message) 
VALUES (?, ?, ?, ?, ?) RETURNING *;