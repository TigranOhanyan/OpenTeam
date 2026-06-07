-- +goose Up
CREATE TABLE run_links (
    parent_run_id TEXT NOT NULL,
    child_run_id TEXT NOT NULL,
    spawning_step_id TEXT NOT NULL,
    linked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (parent_run_id, child_run_id),
    FOREIGN KEY (parent_run_id) REFERENCES runs(id),
    FOREIGN KEY (child_run_id) REFERENCES runs(id),
    FOREIGN KEY (spawning_step_id) REFERENCES steps(id)
);