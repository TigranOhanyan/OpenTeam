-- name: GetMentionByStep :one
SELECT a.* FROM mentions a
WHERE a.step_id = ?
LIMIT 1;