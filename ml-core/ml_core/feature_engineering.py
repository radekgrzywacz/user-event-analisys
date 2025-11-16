import pandas as pd
from sklearn.preprocessing import StandardScaler
import pickle
import os

def preprocess(df, fit=True, scaler=None, save_dir="./saved"):
    grouped = df.groupby("session_id").agg({
        "user_id": "first",
        "event_type": lambda x: list(x),
        "ip": pd.Series.nunique,
        "country": pd.Series.nunique,
        "user_agent": pd.Series.nunique,
        "hour": ["min", "max", "mean"]
    }).reset_index()

    grouped.columns = [
        "session_id", "user_id", "event_sequence",
        "unique_ips", "unique_countries", "unique_agents",
        "min_hour", "max_hour", "avg_hour"
    ]

    grouped["event_count"] = grouped["event_sequence"].apply(len)
    grouped["unique_events"] = grouped["event_sequence"].apply(lambda x: len(set(x)))

    num_features = [
        "unique_ips", "unique_countries", "unique_agents",
        "event_count", "unique_events",
        "min_hour", "max_hour", "avg_hour"
    ]

    if fit:
        scaler = StandardScaler()
        X = scaler.fit_transform(grouped[num_features])
        os.makedirs(save_dir, exist_ok=True)
        with open(os.path.join(save_dir, "scaler.pkl"), "wb") as f:
            pickle.dump(scaler, f)
    else:
        if scaler is None:
            raise ValueError("Scaler must be provided when fit=False")
        X = scaler.transform(grouped[num_features])

    return pd.DataFrame(X, columns=num_features), scaler
