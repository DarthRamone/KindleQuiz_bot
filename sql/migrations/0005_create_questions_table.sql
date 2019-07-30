-- +goose Up
CREATE TABLE questions (
    user_id integer REFERENCES users,
    word_id integer REFERENCES words,
    PRIMARY KEY (user_id, word_id)
);

-- +goose Down
DROP TABLE public.questions;