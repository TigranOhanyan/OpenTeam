-- name: GetAddressingByTurn :one
SELECT a.* FROM addressings a
WHERE a.turn_id = ?
LIMIT 1;