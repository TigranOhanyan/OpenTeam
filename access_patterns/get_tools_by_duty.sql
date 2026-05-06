-- name: GetToolsByDuty :many
SELECT * FROM tools WHERE duty_id = ?;