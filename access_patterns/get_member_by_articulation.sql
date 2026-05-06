-- name: GetMemberByArticulation :one
SELECT m.* FROM members m
JOIN articulations a ON a.from_member_name = m.name
WHERE a.turn_id = ?
LIMIT 1;