CREATE TABLE IF NOT EXISTS users
(
    id bigserial PRIMARY KEY,
    email citext UNIQUE NOT NULL,
    password_hash bytea NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_email ON users (email);

CREATE TABLE IF NOT EXISTS apps
(
    id serial PRIMARY KEY,
    name text UNIQUE NOT NULL,
    secret text UNIQUE NOT NULL
);