-- +goose Up
CREATE TABLE channels (
    name TEXT PRIMARY KEY,
    description TEXT NOT NULL
);