-- +goose Up
CREATE TABLE mentions (
    id TEXT PRIMARY KEY,
    step_id TEXT NOT NULL,
    from_member_task_id TEXT NOT NULL,
    to_member_name TEXT NOT NULL,
    tool_call_id TEXT NOT NULL,
    message TEXT NOT NULL,
    FOREIGN KEY (step_id) REFERENCES steps(id),
    FOREIGN KEY (from_member_task_id) REFERENCES tasks(id),
    FOREIGN KEY (to_member_name) REFERENCES members(name)
);