import os
import pickle
from typing import List

import pandas as pd
from sklearn.preprocessing import StandardScaler

SESSION_FEATURE_COLUMNS: List[str] = [
    "unique_ips",
    "unique_countries",
    "unique_agents",
    "event_count",
    "unique_events",
    "min_hour",
    "max_hour",
    "avg_hour",
]


def build_session_features(df: pd.DataFrame) -> pd.DataFrame:
    """
    Aggregate raw events into a single feature row per session.
    """
    if "session_id" not in df.columns:
        raise ValueError("session_id column is required to build session features")

    work_df = df.copy()
    if "hour" not in work_df.columns:
        if "timestamp" not in work_df.columns:
            raise ValueError("timestamp column is required to derive hour")
        work_df["hour"] = pd.to_datetime(work_df["timestamp"]).dt.hour

    grouped = (
        work_df.groupby("session_id")
        .agg(
            user_id=("user_id", "first"),
            event_sequence=("event_type", lambda x: list(x)),
            unique_ips=("ip", pd.Series.nunique),
            unique_countries=("country", pd.Series.nunique),
            unique_agents=("user_agent", pd.Series.nunique),
            min_hour=("hour", "min"),
            max_hour=("hour", "max"),
            avg_hour=("hour", "mean"),
        )
        .reset_index()
    )

    grouped["event_count"] = grouped["event_sequence"].apply(len)
    grouped["unique_events"] = grouped["event_sequence"].apply(lambda x: len(set(x)))

    grouped = grouped.sort_values("session_id").reset_index(drop=True)
    return grouped.drop(columns=["event_sequence"])


def preprocess(df_sessions: pd.DataFrame, fit: bool = True, scaler=None, save_dir: str = "./saved"):

    if df_sessions.empty:
        return pd.DataFrame(columns=["session_id", *SESSION_FEATURE_COLUMNS]), scaler

    # ensure deterministic order (important for consistency)
    df_sessions = df_sessions.sort_values("session_id").reset_index(drop=True)

    missing = [c for c in SESSION_FEATURE_COLUMNS if c not in df_sessions.columns]
    if missing:
        raise ValueError(f"Missing session feature columns: {missing}")

    session_ids = df_sessions["session_id"].reset_index(drop=True)
    features = df_sessions[SESSION_FEATURE_COLUMNS]

    if fit:
        scaler = StandardScaler()
        X = scaler.fit_transform(features)
        os.makedirs(save_dir, exist_ok=True)
        with open(os.path.join(save_dir, "scaler.pkl"), "wb") as f:
            pickle.dump(scaler, f)
    else:
        if scaler is None:
            raise ValueError("Scaler must be provided when fit=False")
        X = scaler.transform(features)

    X_df = pd.DataFrame(X, columns=SESSION_FEATURE_COLUMNS)
    X_df.insert(0, "session_id", session_ids.values)
    return X_df, scaler
