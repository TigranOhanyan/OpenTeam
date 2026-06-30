-- name: CreateMessage :one
INSERT INTO messages (id, execution_id, channel_name, role_id, task_id, visibility, openai_message) 
VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetContextMessages :many
SELECT * FROM messages 
WHERE 
    (visibility = 'task' AND task_id = ?) OR
    (visibility = 'role' AND role_id = ?) OR
    (visibility = 'channel' AND channel_name = ?)
ORDER BY id ASC;

-- name: GetMessage :one
SELECT * FROM messages WHERE id = ? LIMIT 1;

-- name: GetMessagesByExecution :many
SELECT * FROM messages WHERE execution_id = ? ORDER BY id ASC;

-- name: GetMessages :many
SELECT * FROM messages ORDER BY id ASC;