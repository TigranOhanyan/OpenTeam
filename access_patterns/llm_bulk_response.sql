-- name: CreateLlmBulkResponse :one
INSERT INTO llm_bulk_responses (id, openai_response) VALUES (?, ?) RETURNING *;

-- name: GetLlmBulkResponseByExecution :one
SELECT 
    response.execution_id AS execution_id, 
    bulk.id AS id,
    bulk.openai_response AS openai_response
FROM llm_bulk_responses AS bulk
INNER JOIN llm_responses AS response ON bulk.id = response.id
WHERE response.execution_id = ? 
LIMIT 1;    