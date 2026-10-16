CREATE TABLE IF NOT EXISTS urls (
    id         SERIAL PRIMARY KEY,
    short_url  TEXT        NOT NULL,
    long_url   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT urls_short_url_key UNIQUE (short_url)
);