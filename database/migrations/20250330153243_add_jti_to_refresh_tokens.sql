-- +goose Up
-- +goose StatementBegin

-- Add jti column to refresh_tokens
ALTER TABLE refresh_tokens ADD COLUMN jti UUID NOT NULL UNIQUE;

-- Create index for jti
CREATE INDEX idx_refresh_tokens_jti ON refresh_tokens(jti);

-- Drop the old token_hash column and its index
DROP INDEX IF EXISTS idx_refresh_tokens_token;
ALTER TABLE refresh_tokens DROP COLUMN token_hash;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Recreate token_hash column
ALTER TABLE refresh_tokens ADD COLUMN token_hash VARCHAR(255) NOT NULL;
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token_hash);

-- Drop jti column and its index
DROP INDEX IF EXISTS idx_refresh_tokens_jti;
ALTER TABLE refresh_tokens DROP COLUMN jti;

-- +goose StatementEnd 