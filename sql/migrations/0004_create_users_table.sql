-- +goose Up
CREATE TABLE users (
    id integer NOT NULL PRIMARY KEY,
    current_lang integer DEFAULT 2 REFERENCES languages,
    current_state integer DEFAULT 2
);

-- +goose Down
DROP TABLE users;