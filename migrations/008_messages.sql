-- +goose Up
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    step_id TEXT NOT NULL,
    channel_name TEXT NOT NULL,
    role_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    visibility TEXT NOT NULL CHECK (visibility IN ('channel', 'role', 'task')),
    openai_message JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (step_id) REFERENCES steps(id),
    FOREIGN KEY (channel_name) REFERENCES channels(name),
    FOREIGN KEY (role_id) REFERENCES roles(id),
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);