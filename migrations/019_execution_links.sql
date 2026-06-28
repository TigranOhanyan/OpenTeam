-- +goose Up
CREATE TABLE execution_links (
    parent_id TEXT NOT NULL,
    child_id TEXT NOT NULL,
    linked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (parent_id, child_id),
    FOREIGN KEY (parent_id) REFERENCES executions(id),
    FOREIGN KEY (child_id) REFERENCES executions(id)
);