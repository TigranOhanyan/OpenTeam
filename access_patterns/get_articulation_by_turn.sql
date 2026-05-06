-- name: GetArticulationByTurn :one
SELECT a.* FROM articulations a
WHERE a.turn_id = ?
LIMIT 1;