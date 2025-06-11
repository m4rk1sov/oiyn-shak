CREATE TABLE IF NOT EXISTS email_verification_tokens (
                                           id SERIAL PRIMARY KEY,
                                           token VARCHAR(255) UNIQUE NOT NULL,
                                           user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                           expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
                                           created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
                                           UNIQUE(user_id)
);

CREATE INDEX idx_email_verification_tokens_token ON email_verification_tokens(token);
CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
