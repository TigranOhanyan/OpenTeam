-- name: GetChannelByRole :one
SELECT c.* FROM channels c
JOIN roles r ON r.channel_name = c.name
WHERE r.id = ?
LIMIT 1;