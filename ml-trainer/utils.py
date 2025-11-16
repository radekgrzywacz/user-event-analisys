import os
import yaml
from loguru import logger
import torch

def load_config(path="./config.yml"):
    if not os.path.exists(path):
        raise FileNotFoundError(f"Config file not found at {path}")
    with open(path, "r") as f:
        return yaml.safe_load(f)
    
def save_model(model, save_dir="./saved", filename="model.pt"):
    os.makedirs(save_dir, exist_ok=True)
    save_path = os.path.join(save_dir, filename)
    torch.save(model.state_dict(), save_path)
    logger.info(f"Model saved at {save_path}")
    return save_path

def ensure_dir(path: str): 
    os.makedirs(path, exist_ok=True)
    return path

def log_section(title: str):
    logger.info("\n" + "=" * 80)
    logger.info(title.upper())
    logger.info("=" * 80 + "\n")
