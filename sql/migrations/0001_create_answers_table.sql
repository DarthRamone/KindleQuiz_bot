-- +goose Up
CREATE TABLE public.answers (
                                id SERIAL PRIMARY KEY,
                                word_id integer,
                                user_id integer,
                                correct boolean,
                                user_lang integer,
                                guess character varying(50)
);

-- +goose Down
DROP TABLE public.answers