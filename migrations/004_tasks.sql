-- +goose Up
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    role_id TEXT NOT NULL,
    instruction TEXT NOT NULL,
    FOREIGN KEY (role_id) REFERENCES roles(id)
);