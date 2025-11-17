import json
import time
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any, Callable, Dict, Optional

from loguru import logger
from sqlalchemy import create_engine, text


@dataclass
class TrainingState:
    last_event_id: int = 0
    trained_at: Optional[str] = None
    total_events: int = 0
    total_sessions: int = 0
    metrics: Dict[str, Any] = field(default_factory=dict)


class Retrainer:
    def __init__(self, config: Dict[str, Any], train_fn: Callable[[Dict[str, Any]], Dict[str, Any]]):
        self.config = config
        self.train_fn = train_fn
        retrain_cfg = config.get("retraining", {})

        model_dir = Path(config["training"]["model_dir"])
        default_state_path = model_dir / "training_state.json"
        state_path_cfg = retrain_cfg.get("state_path")
        state_path = Path(state_path_cfg) if state_path_cfg else default_state_path

        self.state_path = state_path
        self.min_new_events = int(retrain_cfg.get("min_new_events", 1000))
        self.poll_interval = int(retrain_cfg.get("poll_interval_seconds", 300))
        max_hours_cfg = retrain_cfg.get("max_hours_between_retrains")
        self.max_hours_between_retrains = int(max_hours_cfg) if max_hours_cfg else None

        self.engine = create_engine(config["database"]["url"])
        self.state = self._load_state()

    def run(self):
        logger.info(
            "Starting retrainer loop (min_new_events=%s, poll_interval=%ss)",
            self.min_new_events,
            self.poll_interval,
        )
        try:
            while True:
                triggered, reason, stats = self._should_retrain()
                if triggered:
                    logger.info("Retraining triggered (%s)", reason)
                    self._execute_training()
                else:
                    logger.debug(
                        "New events since last training: %s (min=%s)",
                        stats["new_events"],
                        self.min_new_events,
                    )
                time.sleep(self.poll_interval)
        except KeyboardInterrupt:
            logger.info("Retrainer stopped by user.")

    def _execute_training(self):
        try:
            metadata = self.train_fn(self.config)
        except ValueError as exc:
            logger.warning("Retraining skipped: %s", exc)
            return
        self._save_state(metadata)
        logger.success(
            "Retraining finished at %s (events=%s sessions=%s)",
            metadata["trained_at"],
            metadata["total_events"],
            metadata["total_sessions"],
        )

    def _should_retrain(self):
        stats = self._collect_new_data_stats()
        if stats["new_events"] >= self.min_new_events:
            return True, f"{stats['new_events']} new events", stats
        if self._is_stale():
            return True, "max_hours_between_retrains reached", stats
        return False, "", stats

    def _collect_new_data_stats(self) -> Dict[str, int]:
        last_event_id = self.state.last_event_id or 0
        with self.engine.connect() as conn:
            result = conn.execute(
                text(
                    """
                    SELECT
                        COUNT(*) AS new_events,
                        COALESCE(MAX(id), :last_id) AS max_event_id
                    FROM events
                    WHERE id > :last_id
                    """
                ),
                {"last_id": last_event_id},
            ).mappings().first()
        return {
            "new_events": int(result["new_events"] or 0),
            "max_event_id": int(result["max_event_id"] or last_event_id),
        }

    def _is_stale(self) -> bool:
        if not self.max_hours_between_retrains:
            return False
        if not self.state.trained_at:
            return True
        try:
            last_trained = datetime.fromisoformat(self.state.trained_at)
        except ValueError:
            return True
        delta = datetime.now(timezone.utc) - last_trained
        return delta >= timedelta(hours=self.max_hours_between_retrains)

    def _load_state(self) -> TrainingState:
        if self.state_path.exists():
            try:
                with open(self.state_path, "r") as f:
                    payload = json.load(f)
                return TrainingState(
                    last_event_id=int(payload.get("last_event_id", 0) or 0),
                    trained_at=payload.get("trained_at"),
                    total_events=int(payload.get("total_events", 0) or 0),
                    total_sessions=int(payload.get("total_sessions", 0) or 0),
                    metrics=payload.get("metrics", {}),
                )
            except Exception as exc:
                logger.warning("Failed to read training state (%s). Starting from scratch.", exc)
        return TrainingState()

    def _save_state(self, metadata: Dict[str, Any]):
        last_event_id = metadata.get("last_event_id") or self.state.last_event_id
        state = TrainingState(
            last_event_id=int(last_event_id or 0),
            trained_at=metadata.get("trained_at"),
            total_events=int(metadata.get("total_events", 0)),
            total_sessions=int(metadata.get("total_sessions", 0)),
            metrics=metadata.get("metrics", {}),
        )
        self.state_path.parent.mkdir(parents=True, exist_ok=True)
        with open(self.state_path, "w") as f:
            json.dump(state.__dict__, f, indent=2, default=str)
        self.state = state
