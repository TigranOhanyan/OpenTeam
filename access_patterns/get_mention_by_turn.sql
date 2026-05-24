-- name: GetMentionByTurn :one
SELECT a.* FROM mentions a
WHERE a.turn_id = ?
LIMIT 1;