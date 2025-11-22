CREATE TABLE
    IF NOT EXISTS users (
        id serial4 NOT NULL,
        created_at timestamptz DEFAULT now () NULL,
        last_seen timestamptz NULL,
        CONSTRAINT users_pkey PRIMARY KEY (id)
    );

CREATE TABLE
    IF NOT EXISTS raw_events (
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

CREATE TABLE
    IF NOT EXISTS aggregated_results (
        id SERIAL PRIMARY KEY,
        session_id TEXT NOT NULL,
        user_id INT NOT NULL,
        ml_anomaly BOOLEAN,
        ml_score DOUBLE PRECISION,
        ml_threshold DOUBLE PRECISION,
        stat_anomaly BOOLEAN,
        anomaly_type TEXT,
        event_count INT,
        unique_events INT,
        created_at TIMESTAMPTZ DEFAULT now ()
    );

CREATE TABLE
    ml_results (
        id SERIAL PRIMARY KEY,
        user_id INT NOT NULL,
        session_id TEXT NOT NULL,
        timestamp TIMESTAMPTZ NOT NULL,
        anomaly BOOLEAN NOT NULL,
        score DOUBLE PRECISION,
        threshold DOUBLE PRECISION,
        event_count INT,
        unique_events INT,
        source TEXT DEFAULT 'ml',
        created_at TIMESTAMPTZ DEFAULT now ()
    );

CREATE TABLE
    stat_results (
        id SERIAL PRIMARY KEY,
        user_id INT NOT NULL,
        session_id TEXT NOT NULL,
        event_type TEXT,
        anomaly BOOLEAN NOT NULL,
        anomaly_type TEXT,
        message TEXT,
        timestamp TIMESTAMPTZ NOT NULL,
        source TEXT DEFAULT 'stat',
        created_at TIMESTAMPTZ DEFAULT now ()
    );

CREATE INDEX IF NOT EXISTS idx_ml_results_session ON ml_results (session_id);

CREATE INDEX IF NOT EXISTS idx_stat_results_session ON stat_results (session_id);

CREATE INDEX IF NOT EXISTS idx_aggregated_results_user ON aggregated_results (user_id);