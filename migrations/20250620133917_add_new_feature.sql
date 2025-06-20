-- +goose Up
-- +goose StatementBegin
CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    instagram_username VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE UNIQUE INDEX idx_unique_subscription ON subscriptions (chat_id, instagram_username);


CREATE INDEX idx_instagram_username ON subscriptions (instagram_username);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE subscriptions;
-- +goose StatementEnd