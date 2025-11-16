import pandas as pd
from sqlalchemy import create_engine
import yaml

def load_config(path="config.yml"):
    with open(path, "r") as f:
        return yaml.safe_load(f)

def load_events():
    config = load_config()
    db = config["database"]


    engine = create_engine(
        f"postgresql+psycopg2://{db['user']}:{db['password']}@{db['host']}:{db['port']}/{db['name']}"
    )

    query = """
    SELECT user_id, session_id, event_type, ip, country, user_agent, EXTRACT(HOUR FROM timestamp) AS HOUR
    FROM events
    WHERE timestamp > NOW() - INTERVAL '30 days';
    """

    df = pd.read_sql(query, engine)
    return df