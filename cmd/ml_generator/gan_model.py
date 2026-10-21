"""
Conditional GAN для генерации персонализированных планов тренировок.

Архитектура:
- Генератор G: (z, c) -> план тренировки, где z ~ N(0, I) (latent вектор),
  c = one_hot(класс функционального состояния) + признаки пользователя.
- Дискриминатор D: (план, c) -> оценка "реальный/сгенерированный".

Связка с бустингом:
1. Бустинг-классификатор F: X -> c (класс состояния: recovery, endurance, power, ...)
2. Условный GAN G: (z, c) -> персонализированный план тренировки
3. Итог: X -> F(X) = c -> G(z, c) = план, адаптированный под состояние пользователя.

Научная новизна: комбинация бустинг-классификации функционального состояния
с условной ГАН для генерации персонализированных планов нагрузки.
"""

import os
import json
import numpy as np
import torch
import torch.nn as nn
import torch.optim as optim
from torch.utils.data import Dataset, DataLoader
from pathlib import Path

# Размерность выходного плана (совпадает с PLAN_DIM из 실험ов)
PLAN_DIM = 19
# Число классов функционального состояния (из классификатора)
NUM_CLASSES = 7
# Размерность latent-вектора
LATENT_DIM = 100
# Размерность условия: one_hot(класс) + признаки пользователя (возраст, пол, вес, рост, цель)
CONDITION_DIM = NUM_CLASSES + 5


class TrainingPlanDataset(Dataset):
    """Датасет планов тренировок с метками классов состояния."""

    def __init__(self, plans, conditions):
        """
        plans: np.array shape (N, PLAN_DIM) - параметры плана
        conditions: np.array shape (N, CONDITION_DIM) - one_hot класс + пользовательские признаки
        """
        self.plans = torch.FloatTensor(plans)
        self.conditions = torch.FloatTensor(conditions)

    def __len__(self):
        return len(self.plans)

    def __getitem__(self, idx):
        return self.plans[idx], self.conditions[idx]


class Generator(nn.Module):
    """Генератор планов тренировок."""

    def __init__(self, latent_dim=LATENT_DIM, condition_dim=CONDITION_DIM, plan_dim=PLAN_DIM):
        super().__init__()
        self.latent_dim = latent_dim
        self.condition_dim = condition_dim
        self.plan_dim = plan_dim

        self.model = nn.Sequential(
            nn.Linear(latent_dim + condition_dim, 512),
            nn.LeakyReLU(0.2),
            nn.BatchNorm1d(512),
            nn.Linear(512, 512),
            nn.LeakyReLU(0.2),
            nn.BatchNorm1d(512),
            nn.Linear(512, 256),
            nn.LeakyReLU(0.2),
            nn.BatchNorm1d(256),
            nn.Linear(256, plan_dim),
            nn.Tanh(),  # выводим в диапазоне [-1, 1], затем денармализуем
        )

    def forward(self, z, condition):
        x = torch.cat([z, condition], dim=1)
        return self.model(x)


class Discriminator(nn.Module):
    """Дискриминатор: отличает реальный план от сгенерированного."""

    def __init__(self, condition_dim=CONDITION_DIM, plan_dim=PLAN_DIM):
        super().__init__()
        self.model = nn.Sequential(
            nn.Linear(plan_dim + condition_dim, 512),
            nn.LeakyReLU(0.2),
            nn.Dropout(0.3),
            nn.Linear(512, 256),
            nn.LeakyReLU(0.2),
            nn.Dropout(0.3),
            nn.Linear(256, 1),
            nn.Sigmoid(),
        )

    def forward(self, plan, condition):
        x = torch.cat([plan, condition], dim=1)
        return self.model(x)


class ConditionalGAN:
    """Обёртка для обучения и инференса условной ГАН."""

    def __init__(self, device=None, lr=2e-4, beta1=0.5):
        self.device = device or torch.device("cuda" if torch.cuda.is_available() else "cpu")
        self.generator = Generator().to(self.device)
        self.discriminator = Discriminator().to(self.device)

        self.opt_g = optim.Adam(self.generator.parameters(), lr=lr, betas=(beta1, 0.999), weight_decay=1e-5)
        self.opt_d = optim.Adam(self.discriminator.parameters(), lr=lr, betas=(beta1, 0.999), weight_decay=1e-5)
        self.criterion = nn.BCELoss()

        self.latent_dim = LATENT_DIM
        self.condition_dim = CONDITION_DIM

    def train_step(self, real_plans, real_conditions):
        """Один шаг обучения ГАН."""
        batch_size = real_plans.size(0)
        real_plans = real_plans.to(self.device)
        real_conditions = real_conditions.to(self.device)

        real_labels = torch.ones(batch_size, 1, device=self.device)
        fake_labels = torch.zeros(batch_size, 1, device=self.device)

        # ----- Обучаем дискриминатор -----
        self.opt_d.zero_grad()

        d_real = self.discriminator(real_plans, real_conditions)
        d_real_loss = self.criterion(d_real, real_labels)

        z = torch.randn(batch_size, self.latent_dim, device=self.device)
        fake_plans = self.generator(z, real_conditions)
        d_fake = self.discriminator(fake_plans.detach(), real_conditions)
        d_fake_loss = self.criterion(d_fake, fake_labels)

        d_loss = d_real_loss + d_fake_loss
        d_loss.backward()
        self.opt_d.step()

        # ----- Обучаем генератор -----
        self.opt_g.zero_grad()

        z = torch.randn(batch_size, self.latent_dim, device=self.device)
        fake_plans = self.generator(z, real_conditions)
        d_fake = self.discriminator(fake_plans, real_conditions)
        g_loss = self.criterion(d_fake, real_labels)  # хотим, чтобы дискриминатор сказал "реально"

        g_loss.backward()
        self.opt_g.step()

        return {"d_loss": d_loss.item(), "g_loss": g_loss.item()}

    def generate(self, condition, num_samples=1):
        """Генерация планов для данного условия."""
        self.generator.eval()
        with torch.no_grad():
            if isinstance(condition, np.ndarray):
                condition = torch.FloatTensor(condition).to(self.device)
            if condition.dim() == 1:
                condition = condition.unsqueeze(0)
            condition = condition.repeat(num_samples, 1)

            z = torch.randn(num_samples, self.latent_dim, device=self.device)
            plans = self.generator(z, condition)
        return plans.cpu().numpy()

    def save(self, path):
        os.makedirs(os.path.dirname(path), exist_ok=True)
        torch.save({
            "generator": self.generator.state_dict(),
            "discriminator": self.discriminator.state_dict(),
            "opt_g": self.opt_g.state_dict(),
            "opt_d": self.opt_d.state_dict(),
        }, path)

    def load(self, path):
        checkpoint = torch.load(path, map_location=self.device, weights_only=True)
        self.generator.load_state_dict(checkpoint["generator"])
        self.discriminator.load_state_dict(checkpoint["discriminator"])
        self.opt_g.load_state_dict(checkpoint["opt_g"])
        self.opt_d.load_state_dict(checkpoint["opt_d"])


def build_condition(class_index, user_features):
    """
    Построение вектора условия для ГАН.
    class_index: int (0-6) класс функционального состояния от бустинга
    user_features: np.array shape (5,) - возраст, пол, вес, рост, цель (кодированные)
    Возвращает np.array shape (CONDITION_DIM,)
    """
    one_hot = np.zeros(NUM_CLASSES, dtype=np.float32)
    one_hot[class_index] = 1.0
    return np.concatenate([one_hot, user_features.astype(np.float32)])


def normalize_plan(plan, plan_min, plan_max):
    """Нормализация плана в диапазон [-1, 1] для Tanh выхода генератора."""
    return 2.0 * (plan - plan_min) / (plan_max - plan_min) - 1.0


def denormalize_plan(plan_norm, plan_min, plan_max):
    """Денармализация плана из [-1, 1] в исходный диапазон."""
    return (plan_norm + 1.0) / 2.0 * (plan_max - plan_min) + plan_min


if __name__ == "__main__":
    # Быстрая проверка архитектуры
    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    print(f"Device: {device}")

    gan = ConditionalGAN(device=device)

    # Тестовый проход
    batch_size = 4
    real_plans = torch.randn(batch_size, PLAN_DIM)
    real_conditions = torch.randn(batch_size, CONDITION_DIM)

    metrics = gan.train_step(real_plans, real_conditions)
    print(f"Train step: {metrics}")

    # Тест генерации
    condition = build_condition(0, np.array([30, 1, 75.0, 180.0, 1.0]))
    plans = gan.generate(condition, num_samples=2)
    print(f"Generated plans shape: {plans.shape}")
    print(f"GAN ready. Plan dim: {PLAN_DIM}, Classes: {NUM_CLASSES}, Condition dim: {CONDITION_DIM}")
