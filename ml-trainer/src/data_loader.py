import pandas as pd
from sqlalchemy import create_engine
from loguru import logger

def load_data(pg_url: str):
    logger.info("Loading data from Postgres...")
    engine = create_engine(pg_url)

    query = """
        SELECT
            id,
            user_id,
            event_type,
            "timestamp",
            ip,
            user_agent,
            country,
            session_id
        FROM raw_events;
    """

    # parse timestamp column here
    df = pd.read_sql(query, engine, parse_dates=["timestamp"])

    # now safe
    df["hour"] = df["timestamp"].dt.hour
    return df
