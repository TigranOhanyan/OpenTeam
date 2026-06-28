-- +goose Up
CREATE TABLE executions (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK(kind IN ('mention', 'agent', 'task', 'react', 'reason', 'act', 'tool')),
    status TEXT NOT NULL CHECK(status IN ('open', 'closed')) DEFAULT 'open',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);