-- +goose Up
CREATE TABLE turn_links (
    prev_id TEXT NOT NULL,
    next_id TEXT NOT NULL,
    linked_at DATETIME,
    PRIMARY KEY (prev_id, next_id),
    FOREIGN KEY (prev_id) REFERENCES turns(id),
    FOREIGN KEY (next_id) REFERENCES turns(id)
);