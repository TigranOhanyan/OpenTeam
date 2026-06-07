-- name: GetAllRuns :many
SELECT * FROM runs r
ORDER BY id ASC;