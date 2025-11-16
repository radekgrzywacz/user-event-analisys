import json
import torch
import pickle
import numpy as np
import pandas as pd
from loguru import logger
from collections import defaultdict
from quixstreams import Application
from ml_core import Autoencoder
import yaml

# === Konfiguracja ===
with open("config.yml", "r") as f:
    config = yaml.safe_load(f)

logger.info("Loading model and scaler...")

# Wczytanie modelu
model = Autoencoder(input_dim=8, hidden_dim=32)
model.load_state_dict(torch.load(config["model"]["model_path"], map_location="cpu"))
model.eval()

# Wczytanie scalera
with open(config["model"]["scaler_path"], "rb") as f:
    scaler = pickle.load(f)

# Wczytanie metryk
metrics = torch.load(config["model"]["metrics_path"], weights_only=False)
threshold = float(metrics["threshold"]) * 10

logger.success("Model and scaler loaded successfully!")

# === Konfiguracja Kafki ===
app = Application(
    broker_address=config["kafka"]["broker_address"],
    consumer_group=config["kafka"]["consumer_group"],
    auto_offset_reset="latest"
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
        timestamp = payload.get("timestamp") or event.get("timestamp")

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
