-- +goose Up
CREATE TABLE user_words (
    user_id integer REFERENCES users,
    word_id integer NOT NULL references words,
    correct_answers integer DEFAULT 0,
    incorrect_answers integer DEFAULT 0,
    PRIMARY KEY (user_id, word_id)
);

-- +goose Down
DROP TABLE user_words;