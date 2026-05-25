-- name: GetTaskByStep :one
SELECT d.*
FROM tasks d
WHERE d.id = COALESCE(
  (SELECT m.task_id FROM messages m WHERE m.step_id = sqlc.arg(step_id) LIMIT 1),
  (SELECT r.task_id FROM llm_responses r WHERE r.step_id = sqlc.arg(step_id) LIMIT 1),
  (SELECT c.task_id FROM llm_chunk_responses c WHERE c.step_id = sqlc.arg(step_id) LIMIT 1)
)
LIMIT 1;