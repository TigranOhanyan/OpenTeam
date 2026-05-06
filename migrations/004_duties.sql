-- +goose Up
CREATE TABLE duties (
    id TEXT PRIMARY KEY,
    role_id TEXT NOT NULL,
    prev_id TEXT,
    instruction TEXT NOT NULL,
    model TEXT NOT NULL,
    stream_mode BOOLEAN NOT NULL,
    FOREIGN KEY (role_id) REFERENCES roles(id),
    FOREIGN KEY (prev_id) REFERENCES duties(id)
);