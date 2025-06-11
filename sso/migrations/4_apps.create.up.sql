CREATE EXTENSION IF NOT EXISTS "pgcrypto";

INSERT INTO apps (id, name, secret)
VALUES
    (1, 'sso', encode(gen_random_bytes(32), 'base64')),
    (2, 'profile', encode(gen_random_bytes(32), 'base64')),
    (3, 'admin', encode(gen_random_bytes(32), 'base64')),
    (4, 'orders', encode(gen_random_bytes(32), 'base64')),
    (5, 'bucket', encode(gen_random_bytes(32), 'base64')),
    (6, 'cart', encode(gen_random_bytes(32), 'base64')),
    (7, 'subscription', encode(gen_random_bytes(32), 'base64')),
    (8, 'toys', encode(gen_random_bytes(32), 'base64'))

ON CONFLICT DO NOTHING;