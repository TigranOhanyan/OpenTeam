-- name: GetContextMessages :many
SELECT * FROM messages 
WHERE 
    (visibility = 'duty' AND duty_id = ?) OR
    (visibility = 'role' AND role_id = ?) OR
    (visibility = 'channel' AND channel_name = ?)
ORDER BY created_at ASC;