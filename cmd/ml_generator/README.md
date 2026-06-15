# ML Generator - Training Plan Generator

## Overview
Generates personalized training plans using a GAN trained on exercise data.

## Model
- **Input:** 64-dimensional noise vector
- **Output:** 19-dimensional plan vector (normalized 0-1)
- **Location:** `models/generator.keras`

## Installation
```bash
pip install -r cmd/ml_generator/requirements.txt
```

## Usage

### Direct Generation
```python
import tensorflow as tf
import numpy as np

model = tf.keras.models.load_model('models/generator.keras')
plan = model.predict(np.random.randn(1, 64), verbose=0)[0]
```

### Via TrainingScript
```python
from cmd.ml_generator.train_gan import TrainingPlanGAN

gan = TrainingPlanGAN(latent_dim=64, plan_dim=19)
plan = gan.generate_plan(seed=42)
plan_dict = gan.generate_plan_dict(seed=42)
```

### Via API
```bash
uvicorn cmd.ml_generator.main:app --host 0.0.0.0 --port 8002
```

POST /generate-plan with:
```json
{
  "training_class": "endurance_e3",
  "user_profile": {
    "age": 30,
    "fitness_level": "intermediate"
  }
}
```

## Plan Features (19 dimensions)

| Index | Feature | Range |
| ------- | --------- | ------- |
| 0 | duration_minutes | 0-100 |
| 1 | intensity_level | 0-1 |
| 2 | rest_ratio | 0-1 |
| 3 | weekly_frequency | 0-7 |
| 4-11 | equipment_tools | 8 values |
| 12 | warmup_ratio | 0-1 |
| 13 | cooldown_ratio | 0-1 |
| 14 | age_factor | 0-1 |
| 15 | fitness_factor | 0-1 |
| 16 | health_factor | 0-1 |
| 17 | goal_strength | 0-1 |
| 18 | goal_endurance | 0-1 |