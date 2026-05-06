-- +goose Up
CREATE TABLE articulations (
    id TEXT PRIMARY KEY,
    turn_id TEXT NOT NULL,
    from_member_name TEXT NOT NULL,
    to_member_name TEXT NOT NULL,
    tool_call_id TEXT NOT NULL,
    message TEXT NOT NULL,
    FOREIGN KEY (turn_id) REFERENCES turns(id),
    FOREIGN KEY (from_member_name) REFERENCES members(name),
    FOREIGN KEY (to_member_name) REFERENCES members(name)
);