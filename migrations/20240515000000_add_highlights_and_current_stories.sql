-- +goose Up
-- +goose StatementBegin
CREATE TABLE highlights (
    id SERIAL PRIMARY KEY,
    username VARCHAR NOT NULL,
    media_url VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE current_stories (
    id SERIAL PRIMARY KEY,
    username VARCHAR NOT NULL,
    media_url VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE highlights;
DROP TABLE current_stories;
-- +goose StatementEnd 