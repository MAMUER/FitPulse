# ADR 0011: Training Plan Generation with GAN

## Status

Accepted

## Context

The ML Generator service needed to produce personalized training plans based on user profiles and biometric data. The initial implementation used a placeholder model with synthetic data. Real exercise data from `exercisedb_v1_sample` was available on D:\ drive.

Requirements:
- Generate 19-dimensional training plan vectors
- Support 10,000+ training samples
- Train in Docker container
- Use Keras 3 with TensorFlow backend

## Decision

Implemented a GAN-based training plan generator with the following architecture:

1. **Data Preprocessing**
   - Converted exercise data from `datasets/raw/exercisedb` to training plan vectors
   - Generated 10,000 training plans with 19 features
   - Features: duration, intensity, rest_ratio, weekly_freq, equipment (8 dims), warmup, cooldown, progression, age/fitness/health/goal factors

2. **Model Architecture**
   - Generator: 64-dim latent → 256 → 512 → 256 → 19 (sigmoid)
   - Discriminator: 19 → 512 → 256 → 128 → 1 (sigmoid)
   - Loss: MSE for generator, binary_crossentropy for discriminator
   - Optimizer: Adam (lr=0.0002, beta_1=0.5)

3. **Training Configuration**
   - 500 epochs, batch size 64
   - Docker-based training with TensorFlow backend
   - Model saved to `models/generator.keras`

## Consequences

- **Positive**: Real exercise data used for training
- **Positive**: Keras 3 compatibility ensures future-proof implementation
- **Positive**: 19-dimensional output supports rich training plan features
- **Neutral**: Training requires ~5 minutes on CPU
- **Neutral**: Model generates normalized vectors (0-1) requiring post-processing

## Implementation

- `cmd/ml-generator/preprocess_exercises.py` - Data preprocessing
- `cmd/ml-generator/train_gan.py` - GAN training script
- `cmd/ml-generator/main.py` - FastAPI service with simplified generation
- `models/generator.keras` - Trained model (1.2MB)

## Usage

```python
import tensorflow as tf
import numpy as np

model = tf.keras.models.load_model('models/generator.keras')
plan = model.predict(np.random.randn(1, 64), verbose=0)[0]
```