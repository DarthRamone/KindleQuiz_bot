-- +goose Up
CREATE TABLE public.answers (
    id SERIAL PRIMARY KEY,
    word_id integer REFERENCES words,
    user_id integer REFERENCES users,
    correct boolean,
    user_lang integer REFERENCES languages,
    guess character varying(50)
);

-- +goose Down
DROP TABLE public.answers