-- name: UpdateActionMessages :one
UPDATE actions 
SET tool_requirement_message_id = ?, tool_result_message_id = ? 
WHERE id = ? RETURNING *;