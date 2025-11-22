import json
import os
import pickle
from collections import defaultdict
from pathlib import Path
from typing import Any, Dict

import numpy as np
import pandas as pd
import torch
import yaml
from dotenv import load_dotenv
from loguru import logger
from quixstreams import Application

from ml_core import Autoencoder

BASE_DIR = Path(__file__).resolve().parent
ENV_PATH = BASE_DIR / ".env"

if not os.getenv("RUNNING_IN_DOCKER") and ENV_PATH.exists():
    load_dotenv(dotenv_path=ENV_PATH, override=False)


def load_app_config(path: Path) -> Dict[str, Any]:
    with open(path, "r") as f:
        config = yaml.safe_load(f)

    kafka_cfg = config.setdefault("kafka", {})
    kafka_cfg["broker_address"] = os.getenv("KAFKA_BROKER_ADDRESS", kafka_cfg.get("broker_address"))
    kafka_cfg["input_topic"] = os.getenv("KAFKA_INPUT_TOPIC", kafka_cfg.get("input_topic"))
    kafka_cfg["output_topic"] = os.getenv("KAFKA_OUTPUT_TOPIC", kafka_cfg.get("output_topic"))
    kafka_cfg["consumer_group"] = os.getenv("KAFKA_CONSUMER_GROUP", kafka_cfg.get("consumer_group"))

    model_cfg = config.setdefault("model", {})
    model_cfg["model_path"] = os.getenv("MODEL_PATH", model_cfg.get("model_path"))
    model_cfg["scaler_path"] = os.getenv("SCALER_PATH", model_cfg.get("scaler_path"))
    model_cfg["metrics_path"] = os.getenv("METRICS_PATH", model_cfg.get("metrics_path"))
    model_cfg["input_dim"] = int(os.getenv("MODEL_INPUT_DIM", model_cfg.get("input_dim", 8)))
    model_cfg["hidden_dim"] = int(os.getenv("MODEL_HIDDEN_DIM", model_cfg.get("hidden_dim", 32)))
    model_cfg["threshold_multiplier"] = float(
        os.getenv("THRESHOLD_MULTIPLIER", model_cfg.get("threshold_multiplier", 10.0))
    )

    state_default = config.get("state_dir", BASE_DIR / "state" / kafka_cfg["consumer_group"])
    config["state_dir"] = os.getenv("STATE_DIR", str(state_default))
    return config


config = load_app_config(BASE_DIR / "config.yml")
Path(config["state_dir"]).mkdir(parents=True, exist_ok=True)

logger.info("Loading model and scaler...")

# Wczytanie modelu
model_cfg = config["model"]
model = Autoencoder(input_dim=model_cfg["input_dim"], hidden_dim=model_cfg["hidden_dim"])
model.load_state_dict(torch.load(model_cfg["model_path"], map_location="cpu"))
model.eval()

# Wczytanie scalera
with open(model_cfg["scaler_path"], "rb") as f:
    scaler = pickle.load(f)

# Wczytanie metryk
metrics = torch.load(model_cfg["metrics_path"], weights_only=False)
threshold = float(metrics["threshold"]) * model_cfg["threshold_multiplier"]

logger.success("Model and scaler loaded successfully!")

# === Konfiguracja Kafki ===
app = Application(
    broker_address=config["kafka"]["broker_address"],
    consumer_group=config["kafka"]["consumer_group"],
    auto_offset_reset="latest",
    state_dir=config["state_dir"],
)

input_topic = app.topic(config["kafka"]["input_topic"], value_deserializer="json")
output_topic = app.topic(config["kafka"]["output_topic"], value_serializer="json")

sdf = app.dataframe(input_topic)

# === Buforowanie eventów po sesji ===
session_buffer = defaultdict(list)
SESSION_BATCH_SIZE = 5  # Ile eventów zbierać przed analizą

# === Główna funkcja przetwarzająca ===
def process_event(event: dict):
    try:
        payload = event.get("payload", {})
        metadata = payload.get("metadata", {})
        correlation = event.get("correlation", {})

        session_id = payload.get("session_id") or correlation.get("session_id")
        user_id = payload.get("user_id")
        timestamp = (
            payload.get("timestamp")
            or event.get("timestamp")
            or pd.Timestamp.utcnow().isoformat()
        )

        # Brak podstawowych danych? Pomijamy event
        if not session_id or not user_id:
            return None

        # Zapisz event do bufora
        session_buffer[session_id].append({
            "user_id": user_id,
            "type": payload.get("type"),
            "ip": metadata.get("ip"),
            "country": metadata.get("country"),
            "user_agent": metadata.get("user_agent"),
            "timestamp": timestamp
        })

        # Jeśli jeszcze nie ma pełnej sesji – czekamy
        if len(session_buffer[session_id]) < SESSION_BATCH_SIZE:
            return None

        # Stwórz DataFrame z eventów danej sesji
        df = pd.DataFrame(session_buffer[session_id])
        df["hour"] = pd.to_datetime(df["timestamp"]).dt.hour

        # Policz cechy takie jak w preprocess()
        features = pd.DataFrame({
            "unique_ips": [df["ip"].nunique()],
            "unique_countries": [df["country"].nunique()],
            "unique_agents": [df["user_agent"].nunique()],
            "event_count": [len(df)],
            "unique_events": [df["type"].nunique()],
            "min_hour": [df["hour"].min()],
            "max_hour": [df["hour"].max()],
            "avg_hour": [df["hour"].mean()]
        })

        # Wyczyść bufor po analizie
        session_buffer[session_id].clear()

        # Skalowanie + predykcja
        X_scaled = scaler.transform(features)
        X_tensor = torch.tensor(X_scaled, dtype=torch.float32)
        with torch.no_grad():
            reconstruction = model(X_tensor)
            loss = torch.nn.functional.mse_loss(reconstruction, X_tensor, reduction="mean").item()

        is_anomaly = bool(loss > threshold)

        result = {
            "user_id": int(user_id),
            "session_id": session_id,
            "timestamp": timestamp,
            "anomaly": is_anomaly,
            "score": float(loss),
            "threshold": float(threshold),
            "event_count": int(len(df)),
            "unique_events": int(df["type"].nunique()),
        }

        return result

    except Exception as e:
        logger.error(f"Error processing event: {e}")
        return None


# === Funkcja wywoływana przez QuixStreams ===
def detect_anomaly(row):
    try:
        result = process_event(row)
        if result:
            logger.info(
                f"[ML] user={result['user_id']} anomaly={result['anomaly']} score={result['score']:.6f}"
            )
            return result
    except Exception as e:
        logger.error(f"Error in detect_anomaly: {e}")
    return None


# === Strumieniowy pipeline ===
(
    sdf
    .apply(detect_anomaly)
    .filter(lambda r: r is not None)
    .to_topic(output_topic)
)

logger.info("Starting ML analyser...")
app.run()
