#!/usr/bin/env python3
"""
Training script for Conditional Diffusion Model - PyTorch 2.5+
FIXED v4: Correct DDPM reverse sampling + conditioning
"""

import json
import os
import warnings
from datetime import datetime
from pathlib import Path

os.environ["PYTHONIOENCODING"] = "utf-8"

import lightning as L
import numpy as np
import pandas as pd
import torch
import torch.nn as nn
from torch.utils.data import DataLoader, TensorDataset

warnings.filterwarnings("ignore", category=FutureWarning)
warnings.filterwarnings("ignore", category=UserWarning)

WANDB_ENABLED = os.environ.get("WANDB_ENABLED", "false").lower() == "true"
if WANDB_ENABLED:
    import wandb
    wandb.init(project="fitpulse-generator", name=f"diffusion_v4_{datetime.now().strftime('%Y%m%d_%H%M')}")
else:
    class MockWandb:
        def log(self, *args, **kwargs): pass
        class config:
            @staticmethod
            def update(*args, **kwargs): pass
        @staticmethod
        def finish(): pass
    wandb = MockWandb()

os.environ["TF_CPP_MIN_LOG_LEVEL"] = "2"
os.environ["LIGHTNING_VERBOSITY"] = "low"

SCRIPT_DIR = Path(__file__).parent.parent.parent
TRAINING_DATA_PATH = SCRIPT_DIR / "datasets" / "processed" / "training_data_with_conditions.csv"
FALLBACK_TRAINING_DATA_PATH = SCRIPT_DIR / "datasets" / "processed" / "training_plans_exercises.csv"
PLAN_DIM = 19
CONDITION_DIM = 32
LATENT_DIM = 64


class ConditionalDiffusionModel(L.LightningModule):
    """Conditional Diffusion Model with proper DDPM sampling"""
    
    def __init__(self, latent_dim=LATENT_DIM, plan_dim=PLAN_DIM, condition_dim=CONDITION_DIM):
        super().__init__()
        self.latent_dim = latent_dim
        self.plan_dim = plan_dim
        self.condition_dim = condition_dim
        
        # Improved architecture with conditioning
        self.noise_pred_net = nn.Sequential(
            nn.Linear(plan_dim + condition_dim + 1, 512),  # +1 for time embedding
            nn.LayerNorm(512),
            nn.GELU(),
            nn.Dropout(0.1),
            nn.Linear(512, 512),
            nn.LayerNorm(512),
            nn.GELU(),
            nn.Dropout(0.1),
            nn.Linear(512, plan_dim),
        )
        
        # Beta schedule (linear)
        self.register_buffer('betas', torch.linspace(1e-4, 0.02, 1000))
        self.register_buffer('alphas', 1.0 - self.betas)
        self.register_buffer('alpha_bar', torch.cumprod(self.alphas, dim=0))
    
    def forward(self, x_t, t, condition=None):
        """Predict noise with conditioning"""
        if condition is None:
            condition = torch.zeros(x_t.shape[0], self.condition_dim, device=x_t.device)
        
        t_emb = t.unsqueeze(-1)
        x_with_t_cond = torch.cat([x_t, condition, t_emb], dim=-1)
        noise_pred = self.noise_pred_net(x_with_t_cond)
        return noise_pred
    
    def training_step(self, batch, batch_idx):
        x_0, condition = batch
        
        t = torch.randint(0, 1000, (x_0.shape[0],), device=self.device)
        noise = torch.randn_like(x_0)
        
        alpha_bar_t = self.alpha_bar[t].unsqueeze(-1)
        x_t = torch.sqrt(alpha_bar_t) * x_0 + torch.sqrt(1 - alpha_bar_t) * noise
        
        noise_pred = self(x_t, t.float() / 1000.0, condition)
        loss = nn.functional.mse_loss(noise_pred, noise)
        
        self.log("train_loss", loss, prog_bar=True, logger=True)
        if WANDB_ENABLED:
            wandb.log({"train_loss": loss.item(), "epoch": self.current_epoch})
        
        return loss
    
    def validation_step(self, batch, batch_idx):
        x_0, condition = batch
        
        t = torch.randint(0, 1000, (x_0.shape[0],), device=self.device)
        noise = torch.randn_like(x_0)
        
        alpha_bar_t = self.alpha_bar[t].unsqueeze(-1)
        x_t = torch.sqrt(alpha_bar_t) * x_0 + torch.sqrt(1 - alpha_bar_t) * noise
        
        noise_pred = self(x_t, t.float() / 1000.0, condition)
        loss = nn.functional.mse_loss(noise_pred, noise)
        
        self.log("val_loss", loss, prog_bar=True, logger=True, sync_dist=True)
        return loss
    
    def configure_optimizers(self):
        return torch.optim.Adam(self.parameters(), lr=3e-4)
    
    @torch.no_grad()
    def sample(self, condition=None, num_steps=1000):
        """Generate sample using CORRECT DDPM reverse process"""
        if condition is None:
            condition = torch.zeros(1, self.condition_dim, device=self.device)
        
        x_t = torch.randn(1, self.plan_dim, device=self.device)
        
        for i in reversed(range(num_steps)):
            t = torch.tensor([i / num_steps], device=self.device)
            
            # Predict noise
            noise_pred = self(x_t, t, condition)
            
            # DDPM reverse step (CORRECT FORMULA)
            alpha_bar_t = self.alpha_bar[i]
            alpha_bar_prev = self.alpha_bar[i - 1] if i > 0 else torch.tensor(1.0, device=self.device)
            alpha_t = alpha_bar_t / alpha_bar_prev
            beta_t = 1 - alpha_t
            
            # Compute x_0 prediction
            x_0_pred = (1 / torch.sqrt(alpha_bar_t)) * (
                x_t - (1 - alpha_bar_t) / torch.sqrt(1 - alpha_bar_t) * noise_pred
            )
            x_0_pred = x_0_pred.clamp(-1, 1)
            
            # Compute mean of x_{t-1}
            mean = torch.sqrt(alpha_bar_prev) * x_0_pred + (1 - alpha_bar_prev) / torch.sqrt(1 - alpha_bar_t) * noise_pred
            
            # Add noise (except at t=0)
            if i > 0:
                sigma_t = torch.sqrt(beta_t)
                noise = torch.randn_like(x_t)
                x_t = mean + sigma_t * noise
            else:
                x_t = mean
        
        # Denormalize from [-1, 1] to [0, 1]
        return ((x_t + 1) / 2).clamp(0, 1)


def load_real_data():
    """Load and normalize training data with conditions"""
    if not TRAINING_DATA_PATH.exists():
        if FALLBACK_TRAINING_DATA_PATH.exists():
            print(f"Warning: {TRAINING_DATA_PATH} not found, using {FALLBACK_TRAINING_DATA_PATH} (no conditions)")
            df = pd.read_csv(FALLBACK_TRAINING_DATA_PATH)
            import ast
            plans = np.array([ast.literal_eval(x) for x in df["plan_vector"]], dtype=np.float32)
            plans = (plans - 0.5) * 2.0
            split_idx = int(len(plans) * 0.8)
            train_plans = plans[:split_idx]
            val_plans = plans[split_idx:]
            train_conditions = np.zeros((len(train_plans), CONDITION_DIM), dtype=np.float32)
            val_conditions = np.zeros((len(val_plans), CONDITION_DIM), dtype=np.float32)
            return train_plans, train_conditions, val_plans, val_conditions
        raise FileNotFoundError(f"Training data not found: {TRAINING_DATA_PATH}")
    
    df = pd.read_csv(TRAINING_DATA_PATH)
    
    if "plan_vector" in df.columns:
        import ast
        plans = np.array([ast.literal_eval(x) for x in df["plan_vector"]], dtype=np.float32)
    else:
        plans = df.iloc[:, :PLAN_DIM].values.astype(np.float32)
    
    # Normalize plans to [-1, 1]
    plans = (plans - 0.5) * 2.0
    
    # Load conditions if available
    if "condition_vector" in df.columns:
        conditions = np.array([ast.literal_eval(x) for x in df["condition_vector"]], dtype=np.float32)
        print(f"Loaded {len(conditions)} condition vectors")
    else:
        conditions = np.zeros((len(plans), CONDITION_DIM), dtype=np.float32)
        print("Warning: No condition_vector column found, using zero conditions")
    
    split_idx = int(len(plans) * 0.8)
    train_plans = plans[:split_idx]
    val_plans = plans[split_idx:]
    train_conditions = conditions[:split_idx]
    val_conditions = conditions[split_idx:]
    
    return train_plans, train_conditions, val_plans, val_conditions


def train_and_save():
    """Train and save model"""
    print("=" * 60)
    print("STARTING DIFFUSION MODEL TRAINING (FIXED v4)")
    print("=" * 60)
    
    print("\n[1/5] Loading training data...")
    train_plans, train_conditions, val_plans, val_conditions = load_real_data()
    print(f"Train: {len(train_plans)} samples, Val: {len(val_plans)} samples")
    print(f"Data range: [{train_plans.min():.2f}, {train_plans.max():.2f}]")
    
    if WANDB_ENABLED:
        wandb.config.update({
            "latent_dim": LATENT_DIM,
            "plan_dim": PLAN_DIM,
            "condition_dim": CONDITION_DIM,
            "train_samples": len(train_plans),
            "val_samples": len(val_plans),
        })
    
    train_dataset = TensorDataset(torch.from_numpy(train_plans), torch.from_numpy(train_conditions))
    val_dataset = TensorDataset(torch.from_numpy(val_plans), torch.from_numpy(val_conditions))
    
    train_loader = DataLoader(train_dataset, batch_size=256, shuffle=True, num_workers=0)
    val_loader = DataLoader(val_dataset, batch_size=256, shuffle=False, num_workers=0)
    
    print("\n[2/5] Training Diffusion Model...")
    model = ConditionalDiffusionModel()
    
    early_stopping = L.pytorch.callbacks.EarlyStopping(
        monitor="val_loss",
        patience=50,
        mode="min",
        min_delta=0.001,
    )
    
    checkpoint_callback = L.pytorch.callbacks.ModelCheckpoint(
        monitor="val_loss",
        dirpath=SCRIPT_DIR / "models" / "checkpoints",
        filename="best-model-{epoch:02d}-{val_loss:.4f}",
        save_top_k=1,
        mode="min",
    )
    
    trainer = L.Trainer(
        max_epochs=500,
        accelerator="auto",
        devices=1,
        log_every_n_steps=10,
        enable_progress_bar=True,
        enable_model_summary=False,
        logger=True,
        callbacks=[early_stopping, checkpoint_callback],
        val_check_interval=1.0,
    )
    
    trainer.fit(model, train_loader, val_loader)
    
    print("\n[3/5] Saving models...")
    model_dir = SCRIPT_DIR / "models"
    os.makedirs(model_dir, exist_ok=True)
    
    best_model_path = checkpoint_callback.best_model_path
    if best_model_path:
        print(f"Loading best model from {best_model_path}")
        model = ConditionalDiffusionModel.load_from_checkpoint(
            best_model_path,
            map_location="cpu",
            weights_only=False
        )
    
    torch.save(model.state_dict(), model_dir / "generator.pt")
    
    model.eval()
    dummy_input = torch.randn(1, PLAN_DIM)
    dummy_t = torch.tensor([0.5])
    dummy_condition = torch.zeros(1, CONDITION_DIM)
    
    onnx_path = model_dir / "generator.onnx"
    torch.onnx.export(
        model,
        (dummy_input, dummy_t, dummy_condition),
        onnx_path,
        input_names=["x_t", "t", "condition"],
        output_names=["noise_pred"],
        dynamic_shapes={
            "x_t": {0: "batch_size"},
            "t": {0: "batch_size"},
            "condition": {0: "batch_size"},
        },
        opset_version=18,
    )
    
    print(f"Model saved to {onnx_path}")
    
    print("\n[4/5] Testing generation...")
    sample = model.sample(num_steps=1000)
    print(f"Sample generated: {sample.cpu().numpy()[0][:5]}...")
    print(f"Min: {sample.min().item():.4f}, Max: {sample.max().item():.4f}")
    
    non_zero_ratio = (sample > 0.01).float().mean().item()
    print(f"Non-zero ratio: {non_zero_ratio:.2%}")
    
    # Проверка разнообразия
    samples = torch.stack([model.sample(num_steps=1000) for _ in range(10)])
    variance = samples.var(dim=0).mean().item()
    print(f"Sample variance: {variance:.4f}")
    
    print("\n[5/5] Quality metrics...")
    print(f"Final val_loss: {trainer.callback_metrics.get('val_loss', 'N/A')}")
    print(f"Best val_loss: {checkpoint_callback.best_model_score:.4f}" if checkpoint_callback.best_model_score else "N/A")
    
    if WANDB_ENABLED:
        wandb.finish()
    
    print("\n" + "=" * 60)
    print("TRAINING COMPLETE!")
    print("=" * 60)


if __name__ == "__main__":
    train_and_save()