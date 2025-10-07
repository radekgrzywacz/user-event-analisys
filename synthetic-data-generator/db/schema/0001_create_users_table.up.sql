CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    country TEXT NOT NULL,
    ip TEXT NOT NULL,
    user_agent TEXT NOT NULL
);