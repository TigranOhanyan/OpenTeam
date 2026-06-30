-- name: CreateMention :one
INSERT INTO mentions (id, execution_id, message_id, from_member_role_id, to_member_name, message) 
VALUES (?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetMentionByExecution :one
SELECT a.* FROM mentions a
WHERE a.execution_id = ?
LIMIT 1;

-- name: GetMention :one
SELECT a.* FROM mentions a
WHERE a.id = ?
LIMIT 1;

-- name: GetMentionsByMessageId :one
SELECT m.* FROM mentions m WHERE m.message_id = ?;