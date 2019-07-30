-- +goose Up
CREATE TABLE words (
    id SERIAL PRIMARY KEY,
    word text NOT NULL,
    stem text NOT NULL,
    lang integer REFERENCES languages,
    UNIQUE (word, stem, lang)
);

-- +goose Down
DROP TABLE words;