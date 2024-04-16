-- +goose Up
-- +goose StatementBegin
CREATE TABLE story_parsers (id SERIAL, story_id VARCHAR, result BOOLEAN, username VARCHAR, created_at TIMESTAMP WITH TIME ZONE);
ALTER TABLE story_parsers ADD CONSTRAINT unique_story_id UNIQUE(story_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE story_parsers;
-- +goose StatementEnd
