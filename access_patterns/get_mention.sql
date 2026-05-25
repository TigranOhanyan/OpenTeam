-- name: GetMention :one
SELECT a.* FROM mentions a
WHERE a.id = ?
LIMIT 1;