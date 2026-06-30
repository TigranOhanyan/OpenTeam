-- +goose Up
CREATE TABLE react_loops (
    execution_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'reason', 'act')),
    CONSTRAINT fk_execution_id FOREIGN KEY (execution_id) REFERENCES executions(id),
    CONSTRAINT fk_task_id FOREIGN KEY (task_id) REFERENCES tasks(id)
);