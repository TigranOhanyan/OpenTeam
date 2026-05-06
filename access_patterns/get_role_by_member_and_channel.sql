-- name: GetRoleByMemberAndChannel :one
SELECT * FROM roles WHERE member_name = ? AND channel_name = ? LIMIT 1;