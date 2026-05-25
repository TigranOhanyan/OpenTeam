-- name: GetContextMessages :many
SELECT * FROM messages 
WHERE 
    (visibility = 'task' AND task_id = ?) OR
    (visibility = 'role' AND role_id = ?) OR
    (visibility = 'channel' AND channel_name = ?)
ORDER BY created_at ASC;