-- +goose Up
CREATE TABLE mentions (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    from_member_task_id TEXT NOT NULL,
    to_member_name TEXT NOT NULL,
    message TEXT NOT NULL,
    FOREIGN KEY (run_id) REFERENCES runs(id),
    FOREIGN KEY (from_member_task_id) REFERENCES tasks(id),
    FOREIGN KEY (to_member_name) REFERENCES members(name)
);