-- name: GetMentionsBySourceStep :many
SELECT m.* FROM mentions m
INNER JOIN runs r ON m.run_id = r.id
WHERE r.source_step_id = ?
ORDER BY m.id ASC;