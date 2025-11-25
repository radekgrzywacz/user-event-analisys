import os
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, Optional

import pandas as pd
import yaml
from dotenv import load_dotenv
from loguru import logger
import psutil

from ml_core import preprocess, save_artifacts, build_session_features
from src.data_loader import load_data
from src.trainer import train_autoencoder, compute_reconstruction_errors

BASE_DIR = Path(__file__).resolve().parent
ENV_PATH = BASE_DIR / ".env"

if not os.getenv("RUNNING_IN_DOCKER") and ENV_PATH.exists():
    load_dotenv(dotenv_path=ENV_PATH, override=False)


def load_config(path: str = "./config.yml") -> Dict[str, Any]:
    with open(path, "r") as f:
        config = yaml.safe_load(f)

    database_url = os.getenv("DATABASE_URL")
    if database_url:
        config.setdefault("database", {})["url"] = database_url

    model_dir = os.getenv("MODEL_DIR")
    if model_dir:
        config.setdefault("training", {})["model_dir"] = model_dir

    retrain_cfg = config.setdefault("retraining", {})

    state_path = os.getenv("TRAINING_STATE_PATH")
    if state_path:
        retrain_cfg["state_path"] = state_path

    overrides = [
        ("min_new_events", "RETRAIN_MIN_NEW_EVENTS"),
        ("poll_interval_seconds", "RETRAIN_POLL_INTERVAL_SECONDS"),
        ("max_hours_between_retrains", "RETRAIN_MAX_HOURS_BETWEEN_RETRAINS"),
    ]
    for key, env_var in overrides:
        value = os.getenv(env_var)
        if value:
            retrain_cfg[key] = int(value)

    return config


def run_training(config: Dict[str, Any], df: Optional[pd.DataFrame] = None) -> Dict[str, Any]:
    """
    Execute the full training pipeline and return metadata about the run.
    When df is provided it will be used instead of loading data from the DB.
    """
    start_time = time.time()
    mem_before = _get_process_memory_mb()
    logger.info(f"Memory before training: {mem_before:.2f} MB")

    if df is None:
        df = load_data(config["database"]["url"])

    if df.empty:
        raise ValueError("No events available for training.")

    logger.info(f"Loaded {len(df)} events from database")
    df_sessions = build_session_features(df)
    if df_sessions.empty:
        raise ValueError("No sessions available for training.")

    logger.info(f"Built {len(df_sessions)} session feature vectors")

    if df_sessions["event_count"].median() < 2:
        logger.warning("Sessions have too few events, anomaly detector may be weak.")

    scaled_sessions, scaler = preprocess(df_sessions, fit=True, save_dir=config["training"]["model_dir"])
    feature_columns = [c for c in scaled_sessions.columns if c != "session_id"]
    X = scaled_sessions[feature_columns]
    logger.info(f"Preprocessed session-level data shape: {X.shape}")

    model, metrics = train_autoencoder(X, config)
    logger.success(f"Training finished. test_loss={metrics['test_loss']:.6f}")

    logger.info(f"Saving model to {config['training']['model_dir']}")
    save_artifacts(model, scaler, metrics, config["training"]["model_dir"])
    logger.success("Model, scaler, and metrics saved successfully!")

    session_errors = compute_reconstruction_errors(model, X)
    df_sessions = df_sessions.reset_index(drop=True)
    df_sessions["reconstruction_error"] = session_errors
    top_anomalies = df_sessions.nlargest(5, "reconstruction_error")
    for _, row in top_anomalies.iterrows():
        logger.warning(
            "Top session anomaly session_id=%s error=%.6f events=%s unique_ips=%s unique_countries=%s",
            row["session_id"],
            row["reconstruction_error"],
            row.get("event_count"),
            row.get("unique_ips"),
            row.get("unique_countries"),
        )

    flattened_metrics: Dict[str, Any] = {}
    for k, v in metrics.items():
        if isinstance(v, (int, float)):
            flattened_metrics[k] = float(v)
        else:
            flattened_metrics[k] = v

    metadata: Dict[str, Any] = {
        "trained_at": datetime.now(timezone.utc).isoformat(),
        "total_events": int(len(df)),
        "total_sessions": int(df_sessions.shape[0]),
        "last_event_id": _safe_max(df, "id"),
        "last_timestamp": _safe_max(df, "timestamp"),
        "metrics": flattened_metrics,
    }

    duration = time.time() - start_time
    mem_after = _get_process_memory_mb()
    logger.info(
        "Training finished in %.2fs (memory %.2f -> %.2f MB)",
        duration,
        mem_before,
        mem_after,
    )
    return metadata


def _safe_max(df: pd.DataFrame, column: str) -> Optional[Any]:
    if column not in df.columns or df[column].isna().all():
        return None
    value = df[column].max()
    if hasattr(value, "to_pydatetime"):
        return value.to_pydatetime()
    if hasattr(value, "item"):
        try:
            return value.item()
        except Exception:
            return value
    return value


def _get_process_memory_mb() -> float:
    try:
        return psutil.Process().memory_info().rss / 1024 ** 2
    except Exception:
        return 0.0


def main():
    config = load_config()
    metadata = run_training(config)
    logger.info(
        "Training metadata: last_event_id=%s total_events=%s sessions=%s",
        metadata.get("last_event_id"),
        metadata["total_events"],
        metadata["total_sessions"],
    )


if __name__ == "__main__":
    main()
