-- +goose Up
CREATE TABLE languages (
    id SERIAL PRIMARY KEY,
    code text,
    english_name text,
    localized_name text,
    UNIQUE (code, english_name, localized_name)
);

INSERT INTO languages (code, english_name, localized_name)
VALUES ('de', 'German', 'Deutsch');

INSERT INTO languages (code, english_name, localized_name)
VALUES ('en', 'English', 'English');

INSERT INTO languages (code, english_name, localized_name)
VALUES ('ru', 'Russian', 'Русский');

-- +goose Down
DROP TABLE languages;