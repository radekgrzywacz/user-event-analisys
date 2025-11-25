import numpy as np
import torch
from torch.utils.data import DataLoader, TensorDataset, random_split
from loguru import logger
from tqdm import tqdm

from ml_core import Autoencoder

def _robust_threshold(errors: np.ndarray) -> float:
    if errors.size == 0:
        return 0.0

    cap = np.percentile(errors, 99.9)
    clipped = np.clip(errors, None, cap)
    median = np.median(clipped)
    mad = np.median(np.abs(clipped - median))
    percentile_995 = np.percentile(clipped, 99.5)

    candidates = [percentile_995]
    if mad > 0:
        candidates.append(median + 3 * mad)
    else:
        q1 = np.percentile(clipped, 25)
        q3 = np.percentile(clipped, 75)
        iqr = q3 - q1
        if iqr > 0:
            candidates.append(q3 + 1.5 * iqr)

    return float(max(candidates))


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

    total = len(dataset)
    if total == 1:
        train_size, test_size = 1, 0
    else:
        proposed_train = int((1 - config["training"]["test_split"]) * total)
        train_size = min(max(proposed_train, 1), total - 1)
        test_size = total - train_size

    train_ds, test_ds = random_split(dataset, [train_size, test_size])

    train_dl = DataLoader(train_ds, batch_size=config["training"]["batch_size"], shuffle=True)
    test_dl = DataLoader(test_ds, batch_size=config["training"]["batch_size"], shuffle=False) if test_size > 0 else None

    model = Autoencoder(input_dim=X.shape[1], hidden_dim=config["training"]["hidden_dim"])
    criterion = torch.nn.MSELoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=config["training"]["learning_rate"])

    logger.info("Starting training loop...")
    train_loss_history = []
    test_loss_history = []
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
        train_loss_history.append(avg_loss)

        test_epoch_loss = None
        if test_dl is not None and len(test_dl) > 0:
            model.eval()
            test_errors_epoch = _batch_reconstruction_errors(model, test_dl)
            test_epoch_loss = float(test_errors_epoch.mean()) if test_errors_epoch.size else 0.0
            test_loss_history.append(test_epoch_loss)
            logger.info(f"Epoch {epoch+1}: train_loss={avg_loss:.6f} val_loss={test_epoch_loss:.6f}")
        else:
            logger.info(f"Epoch {epoch+1}: train_loss={avg_loss:.6f} (no val set)")

    # Ewaluacja
    model.eval()
    test_errors = _batch_reconstruction_errors(model, test_dl) if test_dl is not None else np.array([])
    train_errors = _batch_reconstruction_errors(model, train_dl)

    test_loss = float(test_errors.mean()) if test_errors.size else (float(train_loss_history[-1]) if train_loss_history else 0.0)
    threshold = _robust_threshold(test_errors if test_errors.size else train_errors)

    metrics = {
        "train_loss": avg_loss,
        "test_loss": test_loss,
        "threshold": threshold,
        "input_dim": X.shape[1],
        "hidden_dim": config["training"]["hidden_dim"],
        "train_loss_history": train_loss_history,
        "val_loss_history": test_loss_history,
    }

    logger.info(f"Final test_loss={test_loss:.6f}, threshold={threshold:.6f}")
    return model, metrics


def compute_reconstruction_errors(model: Autoencoder, X):
    X_tensor = torch.tensor(X.values, dtype=torch.float32)
    with torch.no_grad():
        outputs = model(X_tensor)
        errors = torch.mean((outputs - X_tensor) ** 2, dim=1)
    return errors.cpu().numpy()
