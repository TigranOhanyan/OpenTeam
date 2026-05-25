-- name: GetLlmChunkResponseByStep :many
SELECT * FROM llm_chunk_responses WHERE step_id = ? ORDER BY sequence_number ASC;