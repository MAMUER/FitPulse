#!/usr/bin/env python3
"""
Training script for GAN-based Training Plan Generator - Keras 3 compatible
Uses real exercise data from datasets/processed/training_plans_exercises.csv

USAGE:
    # Train the model
    python cmd/ml-generator/train_gan.py
    
    # Generate training plans using the trained model
    from train_gan import TrainingPlanGAN
    gan = TrainingPlanGAN(latent_dim=64, plan_dim=19)
    gan.generator = gan.generator.__class__.from_config(
        gan.generator.get_config()
    )  # Or load from saved model
    plan = gan.generate_plan(seed=42)
    
    # Or directly with Keras:
    import tensorflow as tf
    import numpy as np
    model = tf.keras.models.load_model('models/generator.keras')
    plan = model.predict(np.random.randn(1, 64), verbose=0)[0]
    
    # Plan features (19 dimensions):
    # [duration, intensity, rest_ratio, weekly_freq, equip_tools, warmup, 
    #  cooldown, progression, age_factor, fitness_factor, health_factor,
    #  goal_strength, goal_endurance, goal_flexibility, goal_weight_loss,
    #  goal_muscle_gain, goal_rehabilitation, goal_sport_specific]
"""
import os
import json
import numpy as np
import pandas as pd
from datetime import datetime
from pathlib import Path
import keras
from keras import layers, models
# === MLflow setup ===
import mlflow
mlflow.set_tracking_uri("sqlite:///mlflow.db")
mlflow.set_experiment("fitpulse-generator")
# === конец ===
os.environ['KERAS_BACKEND'] = 'tensorflow'
os.environ['TF_CPP_MIN_LOG_LEVEL'] = '2'

import tensorflow as tf
tf.get_logger().setLevel('ERROR')

SCRIPT_DIR = Path(__file__).parent.parent.parent
TRAINING_DATA_PATH = SCRIPT_DIR / "datasets" / "processed" / "training_plans_exercises.csv"
PLAN_DIM = 19

class TrainingPlanGAN:
    def __init__(self, latent_dim=64, plan_dim=PLAN_DIM):
        self.latent_dim = latent_dim
        self.plan_dim = plan_dim
        self.generator = self.build_generator()
        self.discriminator = self.build_discriminator()
    
    def build_generator(self):
        input_latent = layers.Input(shape=(self.latent_dim,), name='latent_input')
        x = layers.Dense(256, activation='relu')(input_latent)
        x = layers.BatchNormalization()(x)
        x = layers.Dropout(0.2)(x)
        x = layers.Dense(512, activation='relu')(x)
        x = layers.BatchNormalization()(x)
        x = layers.Dropout(0.2)(x)
        x = layers.Dense(256, activation='relu')(x)
        x = layers.BatchNormalization()(x)
        output = layers.Dense(self.plan_dim, activation='sigmoid', name='plan_output')(x)
        model = models.Model(inputs=input_latent, outputs=output, name='generator')
        model.compile(
            optimizer=keras.optimizers.Adam(learning_rate=0.0002, beta_1=0.5),
            loss='mse'
        )
        return model
    
    def build_discriminator(self):
        input_plan = layers.Input(shape=(self.plan_dim,), name='plan_input')
        x = layers.Dense(512, activation='relu')(input_plan)
        x = layers.Dropout(0.3)(x)
        x = layers.Dense(256, activation='relu')(x)
        x = layers.Dropout(0.3)(x)
        x = layers.Dense(128, activation='relu')(x)
        output = layers.Dense(1, activation='sigmoid', name='discriminator_output')(x)
        model = models.Model(inputs=input_plan, outputs=output, name='discriminator')
        model.compile(
            optimizer=keras.optimizers.Adam(learning_rate=0.0002, beta_1=0.5),
            loss='binary_crossentropy',
            metrics=['accuracy']
        )
        return model
    
    def load_real_data(self):
        if not TRAINING_DATA_PATH.exists():
            raise FileNotFoundError(f"Training data not found: {TRAINING_DATA_PATH}")
        df = pd.read_csv(TRAINING_DATA_PATH)
        if 'plan_vector' in df.columns:
            import ast
            plans = np.array([ast.literal_eval(x) for x in df['plan_vector']], dtype=np.float32)
        else:
            plans = df.iloc[:, :self.plan_dim].values.astype(np.float32)
        return plans
    
    def train(self, epochs=500, batch_size=64, save_interval=50):
        with mlflow.start_run(run_name=f"gan_v1_{datetime.now().strftime('%Y%m%d_%H%M')}") as run:
            mlflow.log_params({
                "latent_dim": self.latent_dim,
                "plan_dim": self.plan_dim,
                "epochs": epochs,
                "batch_size": batch_size,
                "optimizer": "Adam",
                "learning_rate": 0.0002,
                "beta_1": 0.5,
            })
            # === конец setup ===
            
            print("=" * 60)
            print("STARTING GAN TRAINING")
            print("=" * 60)
            
            print("\n[1/4] Loading training data...")
            real_plans = self.load_real_data()
            print(f"Loaded {len(real_plans)} training plans with {real_plans.shape[1]} features")
            
            mlflow.log_param("training_samples", len(real_plans))
            mlflow.log_param("feature_dim", real_plans.shape[1])
            
            valid = np.ones((batch_size, 1))
            fake = np.zeros((batch_size, 1))
            history = {'d_loss': [], 'g_loss': [], 'd_acc': []}
            
            print("\n[2/4] Training GAN...")
            for epoch in range(epochs):
                idx = np.random.randint(0, real_plans.shape[0], batch_size)
                real_batch = real_plans[idx]
                noise = np.random.normal(0, 1, (batch_size, self.latent_dim))
                generated = self.generator.predict(noise, verbose=0)
                
                d_loss_real = self.discriminator.train_on_batch(real_batch, valid)
                d_loss_fake = self.discriminator.train_on_batch(generated, fake)
                d_loss = 0.5 * (d_loss_real[0] + d_loss_fake[0])
                d_acc = 0.5 * (d_loss_real[1] + d_loss_fake[1])
                
                self.discriminator.trainable = False
                noise = np.random.normal(0, 1, (batch_size, self.latent_dim))
                g_loss = self.discriminator.train_on_batch(self.generator.predict(noise, verbose=0), valid)
                self.discriminator.trainable = True
                
                history['d_loss'].append(float(d_loss))
                history['g_loss'].append(float(g_loss[0]) if isinstance(g_loss, list) else float(g_loss))
                history['d_acc'].append(float(d_acc))
                
                # Логируем метрики каждые 10 эпох
                if epoch % 10 == 0:
                    mlflow.log_metrics({
                        "d_loss": d_loss,
                        "g_loss": history['g_loss'][-1],
                        "d_accuracy": d_acc,
                    }, step=epoch)
                
                if epoch % save_interval == 0:
                    print(f"Epoch {epoch}: D loss: {history['d_loss'][-1]:.4f}, "
                          f"G loss: {history['g_loss'][-1]:.4f}, D acc: {history['d_acc'][-1]:.4f}")
            
            print("\n[3/4] Saving models...")
            model_dir = SCRIPT_DIR / "models"
            os.makedirs(model_dir, exist_ok=True)
            
            # Сохраняем в MLflow
            mlflow.keras.log_model(self.generator, artifact_path="generator")
            
            # Локально тоже
            self.generator.save(str(model_dir / "generator.keras"))
            mlflow.log_artifact(str(model_dir / "generator.keras"), "models")
            
            history['timestamp'] = datetime.now().isoformat()
            history['run_id'] = run.info.run_id
            
            history_path = model_dir / "gan_training_history.json"
            with open(history_path, 'w', encoding='utf-8') as f:
                json.dump(history, f, indent=2, ensure_ascii=False)
            mlflow.log_artifact(str(history_path), "metadata")
            
            print("\n[4/4] Training Complete!")
            print("=" * 60)
            print(f"MLflow UI: {os.environ.get('MLFLOW_TRACKING_URI', 'http://localhost:5000')}")
            
            return self.generator, history
    
    def generate_plan(self, seed=None):
        """
        Generate a training plan vector.
        
        Returns:
            numpy.ndarray: 19-dimensional plan vector with values in [0, 1]
        """
        if seed is not None:
            np.random.seed(seed)
        noise = np.random.normal(0, 1, (1, self.latent_dim))
        plan = self.generator.predict(noise, verbose=0)[0]
        return plan
    
    def generate_plan_dict(self, seed=None):
        """
        Generate a training plan as a dictionary with feature names.
        
        Returns:
            dict: Plan features with descriptive names
        """
        plan = self.generate_plan(seed)
        feature_names = [
            'duration_minutes', 'intensity_level', 'rest_ratio', 'weekly_frequency',
            'equipment_tools', 'warmup_ratio', 'cooldown_ratio', 'progression_rate',
            'age_factor', 'fitness_factor', 'health_factor',
            'goal_strength', 'goal_endurance', 'goal_flexibility',
            'goal_weight_loss', 'goal_muscle_gain', 'goal_rehabilitation',
            'goal_sport_specific'
        ]
        return dict(zip(feature_names, plan))


def train_and_save():
    gan = TrainingPlanGAN(latent_dim=64, plan_dim=PLAN_DIM)
    generator, history = gan.train(epochs=500, batch_size=64, save_interval=50)
    
    print("\n" + "=" * 60)
    print("Testing Plan Generation")
    print("=" * 60)
    
    sample_plan = gan.generate_plan(seed=42)
    print(f"\nSample generated plan: {sample_plan[:5]}...")
    
    sample_dict = gan.generate_plan_dict(seed=42)
    print(f"\nPlan as dict: duration={sample_dict['duration_minutes']:.2f}, "
          f"intensity={sample_dict['intensity_level']:.2f}, "
          f"frequency={sample_dict['weekly_frequency']:.2f}")
    return generator


if __name__ == '__main__':
    train_and_save()