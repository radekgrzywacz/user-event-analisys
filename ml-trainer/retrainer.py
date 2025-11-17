from main import load_config, run_training
from src.retrainer import Retrainer


def main():
    config = load_config()
    retrainer = Retrainer(config, train_fn=run_training)
    retrainer.run()


if __name__ == "__main__":
    main()
