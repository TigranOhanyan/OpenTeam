-- name: CreateArticulation :one
INSERT INTO articulations (id, turn_id, from_member_name, to_member_name, tool_call_id, message) 
VALUES (?, ?, ?, ?, ?, ?) RETURNING *;