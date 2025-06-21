-- +goose Up
-- +goose StatementBegin
-- Add subscription_type column to subscriptions table
ALTER TABLE subscriptions ADD COLUMN subscription_type VARCHAR(10) NOT NULL DEFAULT 'story';

-- Create post_parsers table
CREATE TABLE post_parsers (
    id SERIAL PRIMARY KEY,
    post_id VARCHAR NOT NULL,
    username VARCHAR NOT NULL,
    post_url VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Add unique constraint to post_id
ALTER TABLE post_parsers ADD CONSTRAINT unique_post_id UNIQUE(post_id);

-- Add index on username for faster lookups
CREATE INDEX idx_post_parsers_username ON post_parsers (username);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop the post_parsers table
DROP TABLE post_parsers;

-- Remove subscription_type column from subscriptions table
ALTER TABLE subscriptions DROP COLUMN subscription_type;
-- +goose StatementEnd 