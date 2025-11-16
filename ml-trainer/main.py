from ml_core import preprocess, Autoencoder, save_artifacts
from src.data_loader import load_data
from src.trainer import train_autoencoder
import yaml
from loguru import logger

def load_config(path="./config.yml"):
    with open(path, "r") as f:
        return yaml.safe_load(f)

def main():
    config = load_config()

    df = load_data(config["database"]["url"])
    logger.info(f"Loaded {len(df)} events from database")

    X, scaler = preprocess(df, fit=True, save_dir="../models")
    logger.info(f"Preprocessed data shape: {X.shape}")

    model, metrics = train_autoencoder(X, config)
    logger.success(f"Training finished. test_loss={metrics['test_loss']:.6f}")

    save_artifacts(model, scaler, metrics, config["training"]["model_dir"])
    logger.success("Model, scaler, and metrics saved successfully!")

if __name__ == "__main__":
    main()
