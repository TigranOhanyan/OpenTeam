-- +goose Up
CREATE TABLE tools (
    id TEXT PRIMARY KEY,
    duty_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    parameters JSON NOT NULL,
    FOREIGN KEY (duty_id) REFERENCES duties(id)
);