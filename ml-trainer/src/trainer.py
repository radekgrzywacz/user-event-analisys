import numpy as np
import torch
from torch.utils.data import DataLoader, TensorDataset, random_split
from loguru import logger
from tqdm import tqdm

from ml_core import Autoencoder

def _robust_threshold(errors: np.ndarray) -> float:
    median = np.median(errors)
    mad = np.median(np.abs(errors - median))
    if mad > 0:
        return float(median + 3 * mad)

    q1 = np.percentile(errors, 25)
    q3 = np.percentile(errors, 75)
    iqr = q3 - q1
    if iqr > 0:
        return float(q3 + 1.5 * iqr)

    return float(errors.max() if errors.size else 0.0)


def _batch_reconstruction_errors(model: Autoencoder, data_loader: DataLoader) -> np.ndarray:
    errors = []
    with torch.no_grad():
        for (batch,) in data_loader:
            outputs = model(batch)
            batch_errors = torch.mean((outputs - batch) ** 2, dim=1)
            errors.extend(batch_errors.cpu().numpy())
    return np.array(errors)


def train_autoencoder(X, config):
    X_tensor = torch.tensor(X.values, dtype=torch.float32)
    dataset = TensorDataset(X_tensor)

    train_size = max(1, int((1 - config["training"]["test_split"]) * len(dataset)))
    test_size = len(dataset) - train_size
    train_ds, test_ds = random_split(dataset, [train_size, test_size])

    train_dl = DataLoader(train_ds, batch_size=config["training"]["batch_size"], shuffle=True)
    test_dl = DataLoader(test_ds, batch_size=config["training"]["batch_size"], shuffle=False)

    model = Autoencoder(input_dim=X.shape[1], hidden_dim=config["training"]["hidden_dim"])
    criterion = torch.nn.MSELoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=config["training"]["learning_rate"])

    logger.info("Starting training loop...")
    for epoch in range(config["training"]["epochs"]):
        model.train()
        total_loss = 0
        for (batch,) in tqdm(train_dl, desc=f"Epoch {epoch+1}/{config['training']['epochs']}"):
            optimizer.zero_grad()
            outputs = model(batch)
            loss = criterion(outputs, batch)
            loss.backward()
            optimizer.step()
            total_loss += loss.item()
        avg_loss = total_loss / len(train_dl)
        logger.info(f"Epoch {epoch+1}: train_loss={avg_loss:.6f}")

    # Ewaluacja
    model.eval()
    test_errors = _batch_reconstruction_errors(model, test_dl)

    test_loss = float(test_errors.mean()) if test_errors.size else 0.0
    threshold = _robust_threshold(test_errors)

    metrics = {
        "train_loss": avg_loss,
        "test_loss": test_loss,
        "threshold": threshold,
        "input_dim": X.shape[1],
        "hidden_dim": config["training"]["hidden_dim"],
    }

    logger.info(f"Final test_loss={test_loss:.6f}, threshold={threshold:.6f}")
    return model, metrics


def compute_reconstruction_errors(model: Autoencoder, X):
    X_tensor = torch.tensor(X.values, dtype=torch.float32)
    with torch.no_grad():
        outputs = model(X_tensor)
        errors = torch.mean((outputs - X_tensor) ** 2, dim=1)
    return errors.cpu().numpy()
