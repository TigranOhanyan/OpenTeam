-- name: GetMember :one
SELECT * FROM members WHERE name = ? LIMIT 1;