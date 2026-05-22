-- +goose Up
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    turn_id TEXT NOT NULL,
    channel_name TEXT NOT NULL,
    role_id TEXT NOT NULL,
    duty_id TEXT NOT NULL,
    visibility TEXT NOT NULL CHECK (visibility IN ('channel', 'role', 'duty', 'hidden')),
    openai_message JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (turn_id) REFERENCES turns(id),
    FOREIGN KEY (channel_name) REFERENCES channels(name),
    FOREIGN KEY (role_id) REFERENCES roles(id),
    FOREIGN KEY (duty_id) REFERENCES duties(id)
);