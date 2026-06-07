-- name: GetActionByStep :many
SELECT * FROM actions WHERE step_id = ?;