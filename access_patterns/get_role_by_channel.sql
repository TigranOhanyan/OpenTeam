-- name: GetRoleByChannel :many
SELECT * FROM roles WHERE channel_name = ?;