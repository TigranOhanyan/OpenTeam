-- +goose Up
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    role_id TEXT NOT NULL,
    prev_id TEXT,
    instruction TEXT NOT NULL,
    FOREIGN KEY (role_id) REFERENCES roles(id),
    FOREIGN KEY (prev_id) REFERENCES tasks(id)
);