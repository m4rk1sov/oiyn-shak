CREATE TABLE IF NOT EXISTS password_reset_tokens (
                                                     id SERIAL PRIMARY KEY,
                                                     token VARCHAR(255) UNIQUE NOT NULL,
                                                     user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                                     expires_at TIMESTAMP NOT NULL,
                                                     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                                     used_at TIMESTAMP
);

CREATE INDEX idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);
