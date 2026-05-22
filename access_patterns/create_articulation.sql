-- name: CreateAddressing :one
INSERT INTO addressings (id, turn_id, from_member_duty_id, to_member_name, tool_call_id, message) 
VALUES (?, ?, ?, ?, ?, ?) RETURNING *;