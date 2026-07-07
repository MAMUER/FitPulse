#!/usr/bin/env python3
"""
Training script for Conditional Diffusion Model - PyTorch 2.5+ with torch.compile()
Uses real exercise data from datasets/processed/training_plans_exercises.csv
"""

import json
import os
from datetime import datetime
from pathlib import Path

import lightning as L
import numpy as np
import pandas as pd
import torch
import torch.nn as nn
from torch.utils.data import DataLoader, TensorDataset

# Optional W&B integration (disabled by default for local development)
WANDB_ENABLED = os.environ.get("WANDB_ENABLED", "false").lower() == "true"
if WANDB_ENABLED:
    import wandb
    wandb.init(project="fitpulse-generator", name=f"diffusion_v1_{datetime.now().strftime('%Y%m%d_%H%M')}")
else:
    # Mock wandb for local development
    class MockWandb:
        def log(self, *args, **kwargs): pass
        def config(self): pass
        def finish(self): pass
    wandb = MockWandb()

os.environ["TF_CPP_MIN_LOG_LEVEL"] = "2"

SCRIPT_DIR = Path(__file__).parent.parent.parent
TRAINING_DATA_PATH = SCRIPT_DIR / "datasets" / "processed" / "training_plans_exercises.csv"
PLAN_DIM = 19
LATENT_DIM = 64


class ConditionalDiffusionModel(L.LightningModule):
    """Conditional Diffusion Model for training plan generation"""
    
    def __init__(self, latent_dim=LATENT_DIM, plan_dim=PLAN_DIM):
        super().__init__()
        self.latent_dim = latent_dim
        self.plan_dim = plan_dim
        
        # UNet-like architecture for denoising
        self.encoder = nn.Sequential(
            nn.Linear(plan_dim, 256),
            nn.LayerNorm(256),
            nn.GELU(),
            nn.Dropout(0.2),
            nn.Linear(256, 512),
            nn.LayerNorm(512),
            nn.GELU(),
        )
        
        self.time_emb = nn.Sequential(
            nn.Linear(1, 128),
            nn.GELU(),
            nn.Linear(128, 512),
        )
        
        self.decoder = nn.Sequential(
            nn.Linear(512 + 512, 512),
            nn.LayerNorm(512),
            nn.GELU(),
            nn.Dropout(0.2),
            nn.Linear(512, 256),
            nn.LayerNorm(256),
            nn.GELU(),
            nn.Linear(256, plan_dim),
        )
    
    def forward(self, x_t, t):
        """Denoise step"""
        x_emb = self.encoder(x_t)
        t_emb = self.time_emb(t.unsqueeze(-1))
        combined = torch.cat([x_emb, t_emb], dim=-1)
        noise_pred = self.decoder(combined)
        return noise_pred
    
    def training_step(self, batch, batch_idx):
        x_0, condition = batch
        
        # Add noise
        t = torch.randint(0, 1000, (x_0.shape[0],), device=self.device)
        noise = torch.randn_like(x_0)
        alpha_bar = self._get_alpha_bar(t).unsqueeze(-1)  # Fix: reshape to (batch_size, 1)
        x_t = alpha_bar * x_0 + torch.sqrt(1 - alpha_bar) * noise
        
        # Predict noise
        noise_pred = self(x_t, t.float() / 1000.0)
        
        # Loss
        loss = nn.functional.mse_loss(noise_pred, noise)
        
        self.log("train_loss", loss, prog_bar=True)
        if WANDB_ENABLED:
            wandb.log({"train_loss": loss.item(), "epoch": self.current_epoch})
        
        return loss
    
    def _get_alpha_bar(self, t):
        """Cosine schedule"""
        return torch.cos((t.float() / 1000.0 + 0.008) / 1.008 * np.pi / 2) ** 2
    
    def configure_optimizers(self):
        return torch.optim.AdamW(self.parameters(), lr=1e-4, weight_decay=1e-5)
    
    @torch.no_grad()
    def sample(self, condition=None, num_steps=100):
        """Generate sample using reverse diffusion"""
        x_t = torch.randn(1, self.plan_dim, device=self.device)
        
        for i in reversed(range(num_steps)):
            t = torch.tensor([i / num_steps], device=self.device)
            noise_pred = self(x_t, t)
            
            alpha_bar = self._get_alpha_bar(torch.tensor([i], device=self.device)).unsqueeze(-1)  # Fix
            alpha_bar_prev = self._get_alpha_bar(torch.tensor([i - 1], device=self.device)).unsqueeze(-1) if i > 0 else torch.tensor([[1.0]], device=self.device)  # Fix
            
            x_0_pred = (x_t - torch.sqrt(1 - alpha_bar) * noise_pred) / torch.sqrt(alpha_bar)
            x_t = alpha_bar_prev * x_0_pred + torch.sqrt(1 - alpha_bar_prev) * noise_pred
            
            if i > 0:
                x_t += torch.sqrt(1 - alpha_bar_prev) * torch.randn_like(x_t)
        
        return x_t.clamp(0, 1)


def load_real_data():
    """Load training data"""
    if not TRAINING_DATA_PATH.exists():
        raise FileNotFoundError(f"Training data not found: {TRAINING_DATA_PATH}")
    
    df = pd.read_csv(TRAINING_DATA_PATH)
    
    if "plan_vector" in df.columns:
        import ast
        plans = np.array([ast.literal_eval(x) for x in df["plan_vector"]], dtype=np.float32)
    else:
        plans = df.iloc[:, :PLAN_DIM].values.astype(np.float32)
    
    # Dummy conditions (can be extended)
    conditions = np.zeros((len(plans), 10), dtype=np.float32)
    
    return plans, conditions


def train_and_save():
    """Train and save model"""
    print("=" * 60)
    print("STARTING DIFFUSION MODEL TRAINING")
    print("=" * 60)
    
    print("\n[1/4] Loading training data...")
    plans, conditions = load_real_data()
    print(f"Loaded {len(plans)} training plans with {plans.shape[1]} features")
    
    if WANDB_ENABLED:
        wandb.config.update({
            "latent_dim": LATENT_DIM,
            "plan_dim": PLAN_DIM,
            "training_samples": len(plans),
            "feature_dim": plans.shape[1],
        })
    
    # Create dataset
    dataset = TensorDataset(torch.from_numpy(plans), torch.from_numpy(conditions))
    dataloader = DataLoader(dataset, batch_size=64, shuffle=True, num_workers=0)  # num_workers=0 for Windows compatibility
    
    print("\n[2/4] Training Diffusion Model...")
    model = ConditionalDiffusionModel()
    
    trainer = L.Trainer(
        max_epochs=500,
        accelerator="auto",
        devices=1,
        log_every_n_steps=10,
        enable_progress_bar=True,
    )
    
    trainer.fit(model, dataloader)
    
    print("\n[3/4] Saving models...")
    model_dir = SCRIPT_DIR / "models"
    os.makedirs(model_dir, exist_ok=True)
    
    # Save PyTorch model
    torch.save(model.state_dict(), model_dir / "generator.pt")
    
    # Export to ONNX for optimized inference
    model_uncompiled = ConditionalDiffusionModel()
    model_uncompiled.load_state_dict(model.state_dict())
    model_uncompiled.eval()
    
    dummy_input = torch.randn(1, PLAN_DIM)
    dummy_t = torch.tensor([0.5])
    
    torch.onnx.export(
        model_uncompiled,
        (dummy_input, dummy_t),
        model_dir / "generator.onnx",
        input_names=["x_t", "t"],
        output_names=["noise_pred"],
        dynamic_axes={"x_t": {0: "batch_size"}, "t": {0: "batch_size"}},
        opset_version=17,
    )
    
    print(f"Model saved to {model_dir / 'generator.onnx'}")
    
    print("\n[4/4] Testing generation...")
    sample = model_uncompiled.sample(num_steps=100)
    print(f"Sample generated: {sample.cpu().numpy()[0][:5]}...")
    
    if WANDB_ENABLED:
        wandb.finish()
    
    print("\n" + "=" * 60)
    print("TRAINING COMPLETE!")
    print("=" * 60)


if __name__ == "__main__":
    train_and_save()