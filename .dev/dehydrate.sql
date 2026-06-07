-- database: ../.dev/0.db
DELETE FROM llm_chunk_responses;

DELETE FROM llm_responses;

DELETE FROM mentions;

DELETE FROM actions;

DELETE FROM messages;

DELETE FROM run_links;

DELETE FROM steps;

DELETE FROM runs;

DELETE FROM tasks AS t
WHERE
  EXISTS (
    SELECT
      1
    FROM
      roles AS r
      JOIN members AS m ON r.member_name = m.name
    WHERE
      r.id = t.role_id
      AND m.kind = 'human'
  );

DELETE FROM roles AS r
WHERE
  EXISTS (
    SELECT
      1
    FROM
      members AS m
    WHERE
      r.member_name = m.name
      AND m.kind = 'human'
  );

DELETE FROM members AS m
WHERE
  m.kind = 'human';
