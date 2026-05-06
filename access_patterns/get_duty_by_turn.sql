-- name: GetDutyByTurn :one
SELECT d.*
FROM duties d
WHERE d.id = COALESCE(
  (SELECT m.duty_id FROM messages m WHERE m.turn_id = sqlc.arg(turn_id) LIMIT 1),
  (SELECT r.duty_id FROM llm_responses r WHERE r.turn_id = sqlc.arg(turn_id) LIMIT 1),
  (SELECT c.duty_id FROM llm_chunk_responses c WHERE c.turn_id = sqlc.arg(turn_id) LIMIT 1)
)
LIMIT 1;