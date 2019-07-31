-- +goose Up
CREATE TABLE questions (
    user_id integer REFERENCES users PRIMARY KEY,
    word_id integer REFERENCES words
);

-- +goose Down
DROP TABLE questions;