CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS profiles (
                                        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                        user_id BIGINT NOT NULL UNIQUE,
                                        name VARCHAR(255) NOT NULL,
                                        phone VARCHAR(50),
                                        address TEXT,
                                        email VARCHAR(255) NOT NULL
);

CREATE INDEX idx_profiles_user_id ON profiles(user_id);
CREATE INDEX idx_profiles_email ON profiles(email);

