import os
import torch
import pickle

def save_artifacts(model, scaler, metrics, save_dir="./saved"):
    os.makedirs(save_dir, exist_ok=True)
    torch.save(model.state_dict(), os.path.join(save_dir, "model.pt"))
    torch.save(metrics, os.path.join(save_dir, "metrics.pt"))
    with open(os.path.join(save_dir, "scaler.pkl"), "wb") as f:
        pickle.dump(scaler, f)

def load_artifacts(model_class, model_dir="./saved"):
    model_path = os.path.join(model_dir, "model.pt")
    metrics_path = os.path.join(model_dir, "metrics.pt")
    scaler_path = os.path.join(model_dir, "scaler.pkl")

    metrics = torch.load(metrics_path)
    with open(scaler_path, "rb") as f:
        scaler = pickle.load(f)

    model = model_class(input_dim=8)
    model.load_state_dict(torch.load(model_path))
    model.eval()

    threshold = metrics["threshold"]
    return model, scaler, threshold
