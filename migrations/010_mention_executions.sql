-- +goose Up
CREATE TABLE mentions (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL UNIQUE,
    message_id TEXT NOT NULL UNIQUE,
    from_member_role_id TEXT NOT NULL,
    to_member_name TEXT NOT NULL,
    message TEXT NOT NULL,
    FOREIGN KEY (execution_id) REFERENCES executions(id),
    FOREIGN KEY (message_id) REFERENCES messages(id),
    FOREIGN KEY (from_member_role_id) REFERENCES roles(id),
    FOREIGN KEY (to_member_name) REFERENCES members(name)
);