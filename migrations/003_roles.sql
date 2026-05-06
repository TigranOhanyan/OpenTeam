-- +goose Up
CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    member_name TEXT NOT NULL,
    channel_name TEXT NOT NULL,
    FOREIGN KEY (member_name) REFERENCES members(name),
    FOREIGN KEY (channel_name) REFERENCES channels(name),
    UNIQUE (member_name, channel_name)
);