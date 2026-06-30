-- name: CreateLlmChunkResponses :exec
INSERT INTO llm_chunk_responses (id, sequence_number, openai_chunk_response) VALUES (?, ?, ?);

-- name: GetLlmChunkResponseByExecution :many
SELECT 
    response.execution_id AS execution_id, 
    chunk.id AS id,
    chunk.sequence_number AS sequence_number,
    chunk.openai_chunk_response AS openai_chunk_response
FROM llm_chunk_responses AS chunk
INNER JOIN llm_responses AS response ON chunk.id = response.id
WHERE response.execution_id = ? 
ORDER BY chunk.sequence_number ASC;
