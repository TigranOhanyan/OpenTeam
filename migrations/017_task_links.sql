-- +goose Up
CREATE TABLE task_links (
    parent_id TEXT NOT NULL,
    child_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (parent_id, child_id),
    FOREIGN KEY (parent_id) REFERENCES tasks(id),
    FOREIGN KEY (child_id) REFERENCES tasks(id)
);