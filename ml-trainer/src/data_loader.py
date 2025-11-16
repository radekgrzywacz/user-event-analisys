import pandas as pd
from sqlalchemy import create_engine
from loguru import logger

def load_data(pg_url: str):
    logger.info("Loading data from Postgres...")
    engine = create_engine(pg_url)
    query = "SELECT * FROM events;"
    df = pd.read_sql(query, engine)
    df["hour"] = df["timestamp"].dt.hour
    return df
