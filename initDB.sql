CREATE TABLE IF NOT EXISTS urls(
    short_id CHAR(50) PRIMARY KEY NOT NULL,
    full_url TEXT UNIQUE,
    user_id CHAR(72) NOt NULL
);