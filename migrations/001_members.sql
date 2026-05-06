-- +goose Up
CREATE TABLE members (
    name TEXT PRIMARY KEY,
    kind TEXT NOT NULL
);