CREATE TABLE IF NOT EXISTS urls(
    short_id CHAR(10) PRIMARY KEY NOT NULL,
    full_url TEXT,
    user_id CHAR(72) NOt NULL
);