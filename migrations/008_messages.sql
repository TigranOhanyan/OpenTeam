-- +goose Up
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL,
    channel_name TEXT NOT NULL,
    role_id TEXT NOT NULL,
    task_id TEXT NULL,
    visibility TEXT NOT NULL CHECK (visibility IN ('channel', 'role', 'task')),
    openai_message JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_execution_id FOREIGN KEY (execution_id) REFERENCES executions(id),
    CONSTRAINT fk_channel_name FOREIGN KEY (channel_name) REFERENCES channels(name),
    CONSTRAINT fk_role_id FOREIGN KEY (role_id) REFERENCES roles(id),
    CONSTRAINT fk_task_id FOREIGN KEY (task_id) REFERENCES tasks(id),
    CONSTRAINT fk_id FOREIGN KEY (id) REFERENCES llm_responses(id)
);