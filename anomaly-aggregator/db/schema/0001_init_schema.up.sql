CREATE TABLE IF NOT EXISTS
    users (
        id serial4 NOT NULL,
        created_at timestamptz DEFAULT now () NULL,
        last_seen timestamptz NULL,
        CONSTRAINT users_pkey PRIMARY KEY (id)
    );

CREATE TABLE IF NOT EXISTS
    raw_events (
        id bigserial PRIMARY KEY,
        user_id int NOT NULL REFERENCES users (id),
        event_type text NOT NULL,
        "timestamp" timestamptz NOT NULL,
        ip text NULL,
        user_agent text NULL,
        country text NULL,
        metadata jsonb NULL,
        additional jsonb NULL,
        created_at timestamptz DEFAULT now () NULL,
        session_id text NULL
    );
