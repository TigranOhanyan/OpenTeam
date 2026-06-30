-- name: CreateAction :one
INSERT INTO actions (execution_id, task_id, tool_call, llm_response_id) VALUES (?, ?, ?, ?) RETURNING *;

-- name: GetAction :one
SELECT * FROM actions WHERE execution_id = ? LIMIT 1;

-- name: UpdateActionMessages :one
UPDATE actions 
SET tool_requirement_message_id = ?, tool_result_message_id = ? 
WHERE execution_id = ? RETURNING *;