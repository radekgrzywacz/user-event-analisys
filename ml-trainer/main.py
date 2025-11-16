from src.data_loader import load_events
from src.feature_engineering import preprocess
from src.trainer import train_autoencoder
from utils import load_config

def main():
    config = load_config()
    print("Loading data from postgres...")
    df = load_events()

    print("Preprocessing...")
    X = preprocess(df, config["training"]["model_dir"])

    print("Training...")
    train_autoencoder(X, config)

    print("Training completed!")

if __name__ == "__main__":
    main()