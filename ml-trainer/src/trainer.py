import numpy as np
import torch
from torch.utils.data import DataLoader, TensorDataset, random_split
from loguru import logger
from tqdm import tqdm
import os
from src.model import Autoencoder

def train_autoencoder(X, config):
    X_tensor = torch.tensor(X.values, dtype=torch.float32)
    dataset = TensorDataset(X_tensor)

    train_size = int((1 - config["training"]["test_split"]) * len(dataset))
    test_size = len(dataset) - train_size
    train_ds, test_ds = random_split(dataset, [train_size, test_size])

    train_dl = DataLoader(train_ds, batch_size=config["training"]["batch_size"], shuffle=True)
    test_dl = DataLoader(test_ds, batch_size=config["training"]["batch_size"], shuffle=False)

    model = Autoencoder(input_dim=X.shape[1], hidden_dim=config["training"]["hidden_dim"])
    criterion = torch.nn.MSELoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=config["training"]["learning_rate"])
    
    logger.info("Starting training...")
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
        logger.info(f"Epoch {epoch+1}: loss={total_loss/len(train_dl):.6f}")
    
    model.eval()
    test_losses = []
    with torch.no_grad():
        for(batch, ) in test_dl:
            outputs = model(batch)
            loss = criterion(outputs, batch)
            test_losses.append(loss.item())
    
    test_loss = np.mean(test_losses)
    logger.info(f"Final test_loss={test_loss:.6f}")

    os.makedirs(config["training"]["model_dir"], exist_ok=True)
    torch.save(model.state_dict(), os.path.join(config["training"]["model_dir"], "model.pt"))
    logger.success("[SUCCESS] Model saved successfully")

    threshold = np.percentile(test_losses, 95)
    metrics = {
        "train_loss": avg_loss,
        "test_loss": test_loss,
        "threshold": threshold
    }

    torch.save(metrics, os.path.join(config["training"]["model_dir"], "metrics.pt"))
    logger.info(f"Metrics saved with threshold={threshold:.6f}")

    return model