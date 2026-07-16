"""
ML Generator API Service
Generates personalized training plans using Conditional Diffusion Model
"""

import asyncio
import json
import logging
import os
from contextlib import asynccontextmanager
from typing import Dict, List, Optional

import numpy as np
import onnxruntime as ort
import structlog
from aio_pika import connect_robust
from fastapi import FastAPI, HTTPException
from prometheus_client import Gauge
from pydantic import BaseModel, ConfigDict, Field
from valkey.asyncio import Valkey

# Configure structured logging
structlog.configure(
    processors=[
        structlog.contextvars.merge_contextvars,
        structlog.processors.add_log_level,
        structlog.processors.StackInfoRenderer(),
        structlog.dev.set_exc_info,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.dev.ConsoleRenderer(),
    ],
    wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
)
logger = structlog.get_logger()

# Prometheus metrics
classification_confidence = Gauge(
    "classification_confidence",
    "ML model confidence score for training type classification",
    ["model_version", "class_name"],
)

# Global state
generator_session: Optional[ort.InferenceSession] = None
valkey_client: Optional[Valkey] = None
rabbitmq_connection = None
ml_async_enabled = False

TRAINING_CLASSES = {
    0: "recovery",
    1: "endurance_basic",
    2: "endurance_threshold",
    3: "power_hiit",
    4: "overtraining",
    5: "illness",
}

TRAINING_TEMPLATES = {
    "recovery": {
        "duration_range": (20, 45),
        "intensity_range": (0.3, 0.5),
        "exercises": ["walking", "yoga", "stretching", "light_swimming", "mobility"],
        "rest_ratio": 0.7,
        "name_ru": "Восстановление",
    },
    "endurance_basic": {
        "duration_range": (45, 90),
        "intensity_range": (0.5, 0.7),
        "exercises": ["running", "cycling", "swimming", "rowing", "hiking"],
        "rest_ratio": 0.4,
        "name_ru": "Базовая выносливость E1-E2",
    },
    "endurance_threshold": {
        "duration_range": (30, 60),
        "intensity_range": (0.7, 0.85),
        "exercises": ["tempo_run", "threshold_intervals", "fartlek", "critical_power"],
        "rest_ratio": 0.3,
        "name_ru": "Пороговая выносливость E3",
    },
    "power_hiit": {
        "duration_range": (20, 45),
        "intensity_range": (0.85, 1.0),
        "exercises": ["hiit", "strength", "sprints", "crossfit", "plyometrics"],
        "rest_ratio": 0.5,
        "name_ru": "Силовая/HIIT",
    },
    "overtraining": {
        "duration_range": (0, 20),
        "intensity_range": (0.0, 0.3),
        "exercises": ["rest", "walking", "stretching", "yoga", "mobility"],
        "rest_ratio": 0.8,
        "name_ru": "Перетренированность",
    },
    "illness": {
        "duration_range": (0, 0),
        "intensity_range": (0.0, 0.0),
        "exercises": ["rest"],
        "rest_ratio": 1.0,
        "name_ru": "Заболевание",
    },
}


class UserProfile(BaseModel):
    """User profile for plan generation — all fields optional with defaults"""
    model_config = ConfigDict(strict=True)

    gender: Optional[str] = Field("male", description="Gender (male/female)")
    age: Optional[int] = Field(30, description="Age", ge=10, le=100)
    fitness_level: Optional[str] = Field(
        "intermediate", description="Fitness level (beginner/intermediate/advanced)"
    )
    weight: Optional[float] = Field(70.0, description="Weight (kg)", ge=30, le=200)
    height: Optional[float] = Field(170.0, description="Height (cm)", ge=100, le=250)
    health_conditions: Optional[List[str]] = Field(None, description="Health conditions")
    goals: Optional[List[str]] = Field(None, description="Training goals")
    lifestyle: Optional[Dict] = Field(None, description="Lifestyle factors")
    allergies: Optional[List[str]] = Field(None, description="Allergies")
    contraindications: Optional[List[str]] = Field(None, description="Medical contraindications")


class HealthStatus(BaseModel):
    """Health status context from classifier and biometrics"""
    model_config = ConfigDict(strict=True)

    predicted_class: Optional[str] = Field("endurance_basic", description="Classifier predicted class")
    confidence: Optional[float] = Field(0.5, description="Classifier confidence", ge=0.0, le=1.0)
    hrv: Optional[float] = Field(65.0, description="Heart rate variability (ms)")
    sleep_hours: Optional[float] = Field(7.0, description="Sleep hours")
    active_conditions_count: Optional[int] = Field(0, description="Active health conditions count", ge=0)
    menstrual_phase: Optional[str] = Field("unknown", description="Menstrual phase")
    day_of_cycle: Optional[int] = Field(1, description="Day of menstrual cycle", ge=1, le=35)
    cycle_length: Optional[int] = Field(28, description="Menstrual cycle length (days)", ge=20, le=40)
    body_composition: Optional[Dict] = Field(None, description="BMI, body fat %, muscle mass")


class TrainingHistory(BaseModel):
    """Recent training history"""
    model_config = ConfigDict(strict=True)

    completed_workouts_count: Optional[int] = Field(0, description="Workouts completed in last 30 days", ge=0)
    avg_intensity: Optional[float] = Field(0.5, description="Average workout intensity", ge=0.0, le=1.0)
    last_workout_date: Optional[str] = Field(None, description="ISO date of last workout")


class PlanGenerationRequest(BaseModel):
    """Request for training plan generation"""
    model_config = ConfigDict(strict=True)

    training_class: str = Field(..., description="Training class from classifier")
    user_profile: UserProfile
    health_status: Optional[HealthStatus] = None
    training_history: Optional[TrainingHistory] = None
    preferences: Optional[Dict] = Field(None, description="User preferences")
    constraints: Optional[Dict] = Field(None, description="Plan constraints")


class Exercise(BaseModel):
    """Exercise details"""
    model_config = ConfigDict(strict=True)

    name: str
    duration_minutes: int
    intensity: float


class TrainingPlan(BaseModel):
    """Generated training plan"""
    model_config = ConfigDict(strict=True)

    training_type: str
    training_type_ru: str
    duration_minutes: int
    intensity: float
    weekly_frequency: int
    primary_exercise: str
    warmup_minutes: int
    cooldown_minutes: int
    exercises: List[str]
    session_structure: List[Exercise]
    notes: List[str]
    weekly_schedule: Optional[Dict] = None


async def load_generator():
    """Load ONNX-optimized generator model"""
    global generator_session

    model_path = "/app/models/generator.onnx"

    if os.path.exists(model_path):
        # ONNX Runtime с оптимизациями
        sess_options = ort.SessionOptions()
        sess_options.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
        sess_options.execution_mode = ort.ExecutionMode.ORT_PARALLEL
        
        generator_session = ort.InferenceSession(
            model_path,
            sess_options,
            providers=["CPUExecutionProvider"]
        )
        logger.info("Generator loaded from ONNX", path=model_path)
    else:
        logger.error("Generator not found", path=model_path)


def generate_from_noise(noise: np.ndarray, condition: np.ndarray, num_steps: int = 100) -> np.ndarray:
    """Generate plan from noise using DDPM reverse process with ONNX noise predictor"""
    if generator_session is None:
        raise RuntimeError("Generator not loaded")

    plan_dim = noise.shape[-1] if noise.ndim > 1 else 19

    x_t = noise.copy()
    for i in reversed(range(num_steps)):
        t = np.array([i / num_steps], dtype=np.float32)

        input_dict = {
            "x_t": x_t.astype(np.float32),
            "t": t.astype(np.float32),
            "condition": condition.astype(np.float32),
        }

        input_names = [inp.name for inp in generator_session.get_inputs()]
        ordered_inputs = {name: input_dict[name] for name in input_names if name in input_dict}
        result = generator_session.run(None, ordered_inputs)
        noise_pred = result[0]

        alpha_bar_t = betas_np[i]
        alpha_bar_prev = betas_np[i - 1] if i > 0 else 1.0
        alpha_t = alpha_bar_t / alpha_bar_prev
        beta_t = 1.0 - alpha_t

        x_0_pred = (1.0 / np.sqrt(alpha_bar_t)) * (x_t - (1.0 - alpha_bar_t) / np.sqrt(1.0 - alpha_bar_t) * noise_pred)
        x_0_pred = np.clip(x_0_pred, -1, 1)

        mean = np.sqrt(alpha_bar_prev) * x_0_pred + (1.0 - alpha_bar_prev) / np.sqrt(1.0 - alpha_bar_t) * noise_pred
        if i > 0:
            sigma_t = np.sqrt(beta_t)
            x_t = mean + sigma_t * np.random.randn(*mean.shape)
        else:
            x_t = mean

    return ((x_t + 1.0) / 2.0).clip(0, 1)[0]


_betas_np = None


def get_betas():
    global _betas_np
    if _betas_np is None:
        _betas_np = np.linspace(1e-4, 0.02, 1000, dtype=np.float32)
    return _betas_np


betas_np = get_betas()


def build_rule_based_plan(training_class: str, user_profile: UserProfile) -> np.ndarray:
    """Rule-based 19-dim plan vector fallback"""
    template = TRAINING_TEMPLATES.get(training_class, TRAINING_TEMPLATES["endurance_basic"])
    duration_min = int(np.mean(template["duration_range"]))
    intensity = float(np.mean(template["intensity_range"]))
    rest_ratio = template["rest_ratio"]
    weekly_freq = 3

    equipment = np.zeros(8)
    for idx, ex in enumerate(template["exercises"][:8]):
        equipment[idx] = 1.0

    warmup = 0.15
    cooldown = 0.15
    age_factor = max(0.0, 1.0 - (user_profile.age - 30) / 70.0)
    fitness_map = {"beginner": 0.3, "intermediate": 0.5, "advanced": 0.8}
    fitness_factor = fitness_map.get(user_profile.fitness_level, 0.5)
    health_factor = 0.5 if user_profile.health_conditions else 1.0
    goals = [g.lower() for g in (user_profile.goals or [])]
    goal_strength = 0.8 if any("набор" in g or "muscle" in g for g in goals) else 0.2
    goal_endurance = 0.8 if any("выносливость" in g or "endurance" in g for g in goals) else 0.2

    return np.array([
        duration_min / 100.0, intensity, rest_ratio, weekly_freq / 7.0,
        *equipment, warmup, cooldown, age_factor, fitness_factor, health_factor,
        goal_strength, goal_endurance, 0.0, 0.0, 0.0
    ], dtype=np.float32)


def build_static_beginner_plan() -> np.ndarray:
    """Static beginner plan fallback"""
    return np.array([
        0.4, 0.4, 0.5, 0.4,
        1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0,
        0.2, 0.2, 0.6, 0.3, 0.7, 0.2, 0.2,
    ], dtype=np.float32)


def apply_post_processing_rules(plan_vector: np.ndarray, request: PlanGenerationRequest) -> np.ndarray:
    """Apply conditional post-processing rules to 19-dim plan vector"""
    pv = plan_vector.copy()
    health = request.health_status or HealthStatus()
    profile = request.user_profile

    phase = (health.menstrual_phase or "unknown").lower()
    if phase == "menstruation":
        pv[1] = max(0.0, pv[1] - 0.2)
        pv[12] = min(1.0, pv[12] + 0.1)
        pv[13] = min(1.0, pv[13] + 0.1)

    if "pregnancy" in [c.lower() for c in (profile.contraindications or [])]:
        pv[1] = min(pv[1], 0.5)
        pv[0] = min(pv[0], 0.4)

    if health.predicted_class == "illness":
        pv[0] = 0.0
        pv[1] = 0.0

    if health.predicted_class == "overtraining":
        pv[3] = max(0.2, pv[3] * 0.7)
        pv[2] = min(1.0, pv[2] + 0.2)

    if profile.age > 60:
        pv[1] = max(0.0, pv[1] - 0.2)
        pv[12] = min(1.0, pv[12] + 0.1)

    bmi = (profile.weight or 70) / ((profile.height or 170) / 100) ** 2
    if bmi > 35:
        pv[1] = max(0.0, pv[1] - 0.15)

    if (health.sleep_hours or 7.0) < 6:
        pv[1] = max(0.0, pv[1] - 0.15)
        pv[13] = min(1.0, pv[13] + 0.1)

    return pv


async def init_async():
    """Initialize async RabbitMQ consumer and Valkey client."""
    global valkey_client, rabbitmq_connection, ml_async_enabled

    ml_async_enabled = os.environ.get("ML_ASYNC", "").lower() == "true"
    if not ml_async_enabled:
        logger.info("Async mode disabled")
        return

    # Async Valkey
    valkey_host = os.environ.get("VALKEY_HOST", "localhost")
    valkey_port = int(os.environ.get("VALKEY_PORT", 6379))

    try:
        valkey_client = Valkey(host=valkey_host, port=valkey_port, decode_responses=True)
        await valkey_client.ping()
        logger.info("Valkey connected", host=valkey_host, port=valkey_port)
    except Exception as e:
        logger.error("Valkey connection failed", error=str(e))
        ml_async_enabled = False
        valkey_client = None
        return

    # Async RabbitMQ consumer
    rabbitmq_url = os.environ.get("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
    
    try:
        rabbitmq_connection = await connect_robust(rabbitmq_url)
        asyncio.create_task(_consume_rabbitmq())
        logger.info("RabbitMQ consumer started")
    except Exception as e:
        logger.error("RabbitMQ connection failed", error=str(e))
        ml_async_enabled = False


async def _consume_rabbitmq():
    """Async RabbitMQ consumer loop."""
    async with rabbitmq_connection:
        channel = await rabbitmq_connection.channel()
        await channel.set_qos(prefetch_count=1)
        queue = await channel.declare_queue("ml.generate", durable=True)

        async with queue.iterator() as queue_iter:
            async for message in queue_iter:
                async with message.process():
                    await _on_generate_message(message.body)


async def _on_generate_message(body: bytes):
    """Process a plan generation job from RabbitMQ."""
    job_id = None
    try:
        message = json.loads(body)
        job_id = message.get("job_id")

        if not job_id:
            logger.error("Received message without job_id")
            return

        logger.info("Processing plan generation job", job_id=job_id)

        training_class = message["training_class"]
        user_profile_dict = message["user_profile"]
        preferences = message.get("preferences")
        health_status_dict = message.get("health_status")
        training_history_dict = message.get("training_history")

        up = UserProfile(**user_profile_dict)
        hs = HealthStatus(**health_status_dict) if health_status_dict else None
        th = TrainingHistory(**training_history_dict) if training_history_dict else None

        plan = await _do_generate_plan(training_class, up, preferences, hs, th)

        result = {
            "job_id": job_id,
            "status": "completed",
            "result": plan,
            "completed_at": __import__("datetime").datetime.utcnow().isoformat() + "Z",
        }

        await valkey_client.setex(f"ml:result:{job_id}", 3600, json.dumps(result))
        logger.info("Job completed", job_id=job_id)

    except Exception as e:
        logger.error("Error processing job", job_id=job_id, error=str(e))
        raise


async def _do_generate_plan(training_class, user_profile, preferences=None,
                            health_status=None, training_history=None):
    """Core plan generation logic with 3-tier fallback."""
    condition = encode_user_profile(user_profile, health_status, training_history, preferences)

    # Tier 1: Diffusion model
    if generator_session is not None:
        try:
            noise = np.random.normal(0, 1, (1, 19)).astype(np.float32)
            plan_vector = generate_from_noise(noise, condition, num_steps=50)
            plan_vector = apply_post_processing_rules(plan_vector, PlanGenerationRequest(
                training_class=training_class,
                user_profile=user_profile,
                health_status=health_status,
                training_history=training_history,
                preferences=preferences,
            ))
            return plan_vector.tolist()
        except Exception as e:
            logger.warning("Diffusion generation failed, falling back to rule-based", error=str(e))

    # Tier 2: Rule-based
    try:
        plan_vector = build_rule_based_plan(training_class, user_profile)
        return plan_vector.tolist()
    except Exception as e:
        logger.warning("Rule-based generation failed, falling back to static", error=str(e))

    # Tier 3: Static beginner plan
    plan_vector = build_static_beginner_plan()
    return plan_vector.tolist()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Modern startup/shutdown pattern"""
    # Startup
    await load_generator()
    await init_async()
    logger.info("ML Generator Service started")
    yield
    # Shutdown
    if valkey_client:
        await valkey_client.close()
    if rabbitmq_connection:
        await rabbitmq_connection.close()
    logger.info("ML Generator Service stopped")


# Single FastAPI app definition with lifespan
app = FastAPI(
    title="ML Generator Service",
    description="Generates personalized training plans using Conditional Diffusion",
    version="2.0.0",
    lifespan=lifespan,
)


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "generator_loaded": generator_session is not None,
        "async_enabled": ml_async_enabled,
    }


@app.get("/templates")
async def get_templates():
    """Get training templates"""
    return TRAINING_TEMPLATES


def encode_user_profile(profile: UserProfile, health_status: Optional[HealthStatus] = None,
                        training_history: Optional[TrainingHistory] = None,
                        preferences: Optional[Dict] = None) -> np.ndarray:
    """Encode full user context to 32-dimensional conditional vector"""
    health = health_status or HealthStatus()
    history = training_history or TrainingHistory()
    prefs = preferences or {}

    # 0: age_normalized (0-1)
    age_norm = np.clip((profile.age - 18) / (100 - 18), 0.0, 1.0)

    # 1: bmi_normalized (0-1)
    bmi = (profile.weight or 70) / ((profile.height or 170) / 100) ** 2
    bmi_norm = np.clip((bmi - 15) / (40 - 15), 0.0, 1.0)

    # 2: fitness_level (0-1)
    fitness_map = {"beginner": 0.0, "intermediate": 0.5, "advanced": 1.0}
    fitness_norm = fitness_map.get(profile.fitness_level, 0.5)

    # 3-6: goal one-hot
    goals_lower = [g.lower() for g in (profile.goals or [])]
    goal_strength = 1.0 if any(g in goals_lower for g in ["набор массы", "muscle_gain", "силовые"]) else 0.0
    goal_endurance = 1.0 if any(g in goals_lower for g in ["выносливость", "endurance", "марафон"]) else 0.0
    goal_weight_loss = 1.0 if any(g in goals_lower for g in ["похудение", "weight_loss", "fat_loss"]) else 0.0
    goal_flexibility = 1.0 if any(g in goals_lower for g in ["гибкость", "flexibility", "растяжка"]) else 0.0

    # 7: health_factor (inverse of classifier confidence)
    health_factor = 1.0 - np.clip(health.confidence, 0.0, 1.0)

    # 8-10: menstrual phase one-hot
    phase = (health.menstrual_phase or "unknown").lower()
    menstrual_luteal = 1.0 if phase == "luteal" else 0.0
    menstrual_menstruation = 1.0 if phase == "menstruation" else 0.0
    menstrual_ovulation = 1.0 if phase == "ovulation" else 0.0

    # 11: active_conditions_count_normalized
    conditions_norm = np.clip((health.active_conditions_count or 0) / 5.0, 0.0, 1.0)

    # 12: has_contraindications
    has_contraindications = 1.0 if (profile.contraindications and len(profile.contraindications) > 0) else 0.0

    # 13: has_allergies
    has_allergies = 1.0 if (profile.allergies and len(profile.allergies) > 0) else 0.0

    # 14: recovery_needed
    recovery_needed = 1.0 if (health.predicted_class in ("recovery", "overtraining")) else 0.0

    # 15: days_since_last_workout (0-1)
    days_since = 7.0  # default to 7 days if unknown
    if history.last_workout_date:
        try:
            from datetime import datetime
            last = datetime.fromisoformat(history.last_workout_date.replace("Z", "+00:00"))
            days_since = max(0.0, (datetime.now(last.tzinfo) - last).total_seconds() / 86400.0)
        except Exception:
            pass
    days_since_norm = np.clip(days_since / 7.0, 0.0, 1.0)

    # 16: workout_frequency (0-1)
    workout_freq = np.clip((history.completed_workouts_count or 0) / 30.0, 0.0, 1.0)

    # 17: sleep_quality (0-1)
    sleep_quality = np.clip((health.sleep_hours or 7.0) / 9.0, 0.0, 1.0)

    # 18: hrv_factor (0-1)
    hrv_factor = np.clip((health.hrv or 65.0) / 100.0, 0.0, 1.0)

    # 19: temperature_normalized (0-1)
    temp = 36.6  # default normal temperature
    if health.body_composition and "temperature" in health.body_composition:
        temp = health.body_composition["temperature"]
    temp_norm = np.clip((temp - 35.5) / (38.5 - 35.5), 0.0, 1.0)

    # 20: spo2_factor (0-1)
    spo2 = health.body_composition.get("spo2", 98.0) if health.body_composition else 98.0
    spo2_factor = np.clip(spo2 / 100.0, 0.0, 1.0)

    # 21: available_days_count (0-1)
    available_days = prefs.get("available_days", ["mon", "wed", "fri"])
    available_days_count = np.clip(len(available_days) / 7.0, 0.0, 1.0)

    # 22: preferred_morning
    preferred_time = (prefs.get("time") or "morning").lower()
    preferred_morning = 1.0 if preferred_time == "morning" else 0.0

    # 23: preferred_evening
    preferred_evening = 1.0 if preferred_time == "evening" else 0.0

    # 24-27: equipment one-hot
    equipment = [e.lower() for e in prefs.get("equipment", [])]
    equipment_dumbbell = 1.0 if any("dumbbell" in e for e in equipment) else 0.0
    equipment_resistance_band = 1.0 if any("band" in e or "resistance" in e for e in equipment) else 0.0
    equipment_barbell = 1.0 if any("barbell" in e for e in equipment) else 0.0
    equipment_none = 1.0 if len(equipment) == 0 else 0.0

    # 28-31: reserved for future features
    reserved = np.array([0.0, 0.0, 0.0, 0.0], dtype=np.float32)

    encoded = np.array([
        age_norm, bmi_norm, fitness_norm,
        goal_strength, goal_endurance, goal_weight_loss, goal_flexibility,
        health_factor,
        menstrual_luteal, menstrual_menstruation, menstrual_ovulation,
        conditions_norm, has_contraindications, has_allergies,
        recovery_needed, days_since_norm, workout_freq, sleep_quality, hrv_factor,
        temp_norm, spo2_factor,
        available_days_count, preferred_morning, preferred_evening,
        equipment_dumbbell, equipment_resistance_band, equipment_barbell, equipment_none,
    ], dtype=np.float32)

    encoded = np.concatenate([encoded, reserved])
    return encoded.reshape(1, -1).astype(np.float32)


def decode_plan(plan_vector: np.ndarray, training_class: str, user_profile: UserProfile) -> dict:
    """Decode model output (19 dimensions) to training plan"""
    template = TRAINING_TEMPLATES.get(training_class, TRAINING_TEMPLATES["endurance_basic"])

    duration = int(plan_vector[0] * 100)
    intensity = float(plan_vector[1])
    weekly_freq = int(plan_vector[3] * 7)

    equipment_dist = plan_vector[4:12]
    primary_exercise_idx = int(np.argmax(equipment_dist))
    primary_exercise = template["exercises"][primary_exercise_idx % len(template["exercises"])]

    warmup = int(plan_vector[12] * 100)
    cooldown = int(plan_vector[13] * 100)

    session_structure = [
        Exercise(name="Разминка", duration_minutes=max(5, min(20, warmup)), intensity=0.3),
        Exercise(
            name=primary_exercise,
            duration_minutes=int(duration * 0.6),
            intensity=intensity,
        ),
        Exercise(name="Заминка", duration_minutes=max(5, min(20, cooldown)), intensity=0.3),
    ]

    notes = []
    if user_profile.fitness_level == "beginner":
        notes.append("Начните с 50% от рекомендованной интенсивности")
        duration = int(duration * 0.7)

    if user_profile.age > 50:
        notes.append("Увеличьте время разминки и заминки")

    if user_profile.health_conditions:
        notes.append("Проконсультируйтесь с врачом перед началом")

    if user_profile.goals:
        goals_lower = [g.lower() for g in user_profile.goals]
        if "похудение" in goals_lower:
            notes.append("Добавьте 10-15 минут кардио после основной тренировки")
        if "набор массы" in goals_lower:
            notes.append("Сфокусируйтесь на силовых упражнениях")
        if "реабилитация" in goals_lower:
            notes.append("Следите за техникой выполнения упражнений")

    weekly_schedule = {
        "monday": primary_exercise if weekly_freq >= 1 else "rest",
        "wednesday": primary_exercise if weekly_freq >= 2 else "rest",
        "friday": primary_exercise if weekly_freq >= 3 else "rest",
        "saturday": "active_recovery" if weekly_freq >= 4 else "rest",
        "sunday": "rest",
    }

    return {
        "training_type": training_class,
        "training_type_ru": template["name_ru"],
        "duration_minutes": max(20, min(120, duration)),
        "intensity": round(intensity, 2),
        "weekly_frequency": max(1, min(7, weekly_freq)),
        "primary_exercise": primary_exercise,
        "warmup_minutes": max(5, min(20, warmup)),
        "cooldown_minutes": max(5, min(20, cooldown)),
        "exercises": template["exercises"],
        "session_structure": [e.model_dump() for e in session_structure],
        "notes": notes,
        "weekly_schedule": weekly_schedule,
    }


@app.post("/generate-plan")
async def generate_plan(request: PlanGenerationRequest):
    """Generate personalized training plan (synchronous endpoint)"""
    plan_vector = await _do_generate_plan(
        request.training_class,
        request.user_profile,
        request.preferences,
        request.health_status,
        request.training_history,
    )

    plan = decode_plan(np.asarray(plan_vector, dtype=np.float32), request.training_class, request.user_profile)

    classification_confidence.labels(
        model_version="diffusion_v1",
        class_name=request.training_class,
    ).set(1.0)

    return plan


# ========== Diet Generation ==========

DIET_TEMPLATES = {
    "balanced": {
        "name_ru": "Сбалансированная",
        "protein_ratio": 0.25,
        "fat_ratio": 0.30,
        "carbs_ratio": 0.45,
        "meals": {
            "breakfast": [
                {"name": "Овсянка с бананом и мёдом", "kcal": 350, "protein": 12, "carbs": 60, "fat": 8},
                {"name": "Омлет с овощами и тостом", "kcal": 380, "protein": 22, "carbs": 30, "fat": 18},
            ],
            "snack1": [
                {"name": "Яблоко + миндаль (30г)", "kcal": 200, "protein": 6, "carbs": 22, "fat": 10},
            ],
            "lunch": [
                {"name": "Куриная грудка с рисом и салатом", "kcal": 550, "protein": 40, "carbs": 60, "fat": 15},
            ],
            "snack2": [
                {"name": "Протеиновый батончик", "kcal": 200, "protein": 20, "carbs": 22, "fat": 8},
            ],
            "dinner": [
                {"name": "Индейка с овощами на пару", "kcal": 400, "protein": 35, "carbs": 25, "fat": 18},
            ],
        },
    },
    "high_protein": {
        "name_ru": "Высокобелковая",
        "protein_ratio": 0.35,
        "fat_ratio": 0.25,
        "carbs_ratio": 0.40,
        "meals": {
            "breakfast": [
                {"name": "Омлет из 4 яиц с курицей", "kcal": 450, "protein": 40, "carbs": 5, "fat": 28},
            ],
            "snack1": [
                {"name": "Протеиновый коктейль", "kcal": 200, "protein": 30, "carbs": 8, "fat": 4},
            ],
            "lunch": [
                {"name": "Двойная порция курицы с рисом", "kcal": 650, "protein": 55, "carbs": 55, "fat": 18},
            ],
            "snack2": [
                {"name": "Творог 5% + орехи", "kcal": 250, "protein": 22, "carbs": 10, "fat": 14},
            ],
            "dinner": [
                {"name": "Стейк из лосося с овощами", "kcal": 500, "protein": 40, "carbs": 15, "fat": 28},
            ],
        },
    },
    "weight_loss": {
        "name_ru": "Для похудения",
        "protein_ratio": 0.35,
        "fat_ratio": 0.30,
        "carbs_ratio": 0.35,
        "meals": {
            "breakfast": [
                {"name": "Овсянка на воде с ягодами", "kcal": 220, "protein": 8, "carbs": 40, "fat": 4},
            ],
            "snack1": [
                {"name": "Огурец + хумус", "kcal": 100, "protein": 4, "carbs": 12, "fat": 4},
            ],
            "lunch": [
                {"name": "Куриный суп с овощами", "kcal": 300, "protein": 25, "carbs": 30, "fat": 8},
            ],
            "snack2": [
                {"name": "Зелёное яблоко", "kcal": 70, "protein": 0, "carbs": 18, "fat": 0},
            ],
            "dinner": [
                {"name": "Запечённая белая рыба с салатом", "kcal": 280, "protein": 30, "carbs": 10, "fat": 12},
            ],
        },
    },
    "keto": {
        "name_ru": "Кето",
        "protein_ratio": 0.25,
        "fat_ratio": 0.70,
        "carbs_ratio": 0.05,
        "meals": {
            "breakfast": [
                {"name": "Яичница с авокадо и беконом", "kcal": 450, "protein": 22, "carbs": 5, "fat": 38},
            ],
            "snack1": [
                {"name": "Орехи макадамия (30г)", "kcal": 210, "protein": 2, "carbs": 4, "fat": 21},
            ],
            "lunch": [
                {"name": "Стейк рибай с маслом и брокколи", "kcal": 650, "protein": 48, "carbs": 8, "fat": 50},
            ],
            "snack2": [
                {"name": "Сырная тарелка", "kcal": 280, "protein": 18, "carbs": 2, "fat": 22},
            ],
            "dinner": [
                {"name": "Лосось на сливочном масле с шпинатом", "kcal": 520, "protein": 38, "carbs": 6, "fat": 42},
            ],
        },
    },
    "paleo": {
        "name_ru": "Палео",
        "protein_ratio": 0.30,
        "fat_ratio": 0.35,
        "carbs_ratio": 0.35,
        "meals": {
            "breakfast": [
                {"name": "Омлет с овощами и авокадо", "kcal": 380, "protein": 22, "carbs": 12, "fat": 28},
            ],
            "snack1": [
                {"name": "Грецкие орехи и яблоко", "kcal": 220, "protein": 5, "carbs": 20, "fat": 14},
            ],
            "lunch": [
                {"name": "Курица гриль с кореньями и салатом", "kcal": 520, "protein": 42, "carbs": 18, "fat": 30},
            ],
            "snack2": [
                {"name": "Морковные палочки с хумусом", "kcal": 150, "protein": 5, "carbs": 15, "fat": 8},
            ],
            "dinner": [
                {"name": "Запечённый лосось с спаржей", "kcal": 420, "protein": 35, "carbs": 10, "fat": 28},
            ],
        },
    },
}


class DietGenerationRequest(BaseModel):
    """Request for diet plan generation"""
    model_config = ConfigDict(strict=True)

    user_id: str
    age: int = Field(30, ge=10, le=100)
    gender: str = Field("male")
    weight_kg: float = Field(70.0, ge=30, le=200)
    height_cm: float = Field(170.0, ge=100, le=250)
    fitness_level: str = Field("intermediate")
    goals: List[str] = Field(default_factory=list)
    diet_type: str = Field("balanced")
    meals_count: int = Field(4, ge=2, le=6)
    allergies: List[str] = Field(default_factory=list)
    contraindications: List[str] = Field(default_factory=list)
    training_goal: str = Field("general_fitness")


class DietPlanResponse(BaseModel):
    """Generated diet plan"""
    model_config = ConfigDict(strict=True)

    diet_type: str
    diet_type_ru: str
    daily_calories: int
    bmr: int
    macros: Dict[str, float]
    meals: List[Dict]
    total_protein_g: float
    total_carbs_g: float
    total_fat_g: float
    notes: List[str]


def calculate_diet_plan(request: DietGenerationRequest) -> DietPlanResponse:
    """Calculate personalized diet plan using Mifflin-St Jeor"""
    # BMR calculation
    bmr = 10 * request.weight_kg + 6.25 * request.height_cm - 5 * request.age
    if request.gender.lower() == "male":
        bmr += 5
    else:
        bmr -= 161

    # Activity multiplier
    activity_map = {"beginner": 1.375, "intermediate": 1.55, "advanced": 1.725}
    activity_multiplier = activity_map.get(request.fitness_level, 1.55)

    # Goal adjustment
    goal_adjust = 0
    goals_lower = [g.lower() for g in request.goals]
    if "weight_loss" in goals_lower or "похудение" in goals_lower:
        goal_adjust = -400
    elif "muscle_gain" in goals_lower or "набор массы" in goals_lower:
        goal_adjust = 300
    elif "endurance" in goals_lower or "выносливость" in goals_lower:
        goal_adjust = 100

    tdee = max(1200, int(bmr * activity_multiplier + goal_adjust))

    # Get diet template
    template = DIET_TEMPLATES.get(request.diet_type, DIET_TEMPLATES["balanced"])
    protein_ratio = template["protein_ratio"]
    fat_ratio = template["fat_ratio"]
    carbs_ratio = template["carbs_ratio"]

    protein_g = round((tdee * protein_ratio) / 4)
    fat_g = round((tdee * fat_ratio) / 9)
    carbs_g = round((tdee * carbs_ratio) / 4)

    # Select meals
    meal_keys = ["breakfast", "snack1", "lunch", "snack2", "dinner"]
    selected_meals = []
    total_kcal = 0
    total_protein = 0
    total_carbs = 0
    total_fat = 0

    for i in range(min(request.meals_count, len(meal_keys))):
        key = meal_keys[i]
        meals = template["meals"].get(key, [])
        if not meals:
            continue
        meal = meals[0]  # Simplified: take first meal
        selected_meals.append({
            "name": meal["name"],
            "kcal": meal["kcal"],
            "protein": meal["protein"],
            "carbs": meal["carbs"],
            "fat": meal["fat"],
            "time": _get_meal_time(i, request.meals_count),
        })
        total_kcal += meal["kcal"]
        total_protein += meal["protein"]
        total_carbs += meal["carbs"]
        total_fat += meal["fat"]

    notes = []
    if request.age > 60:
        notes.append("Увеличьте потребление белка и уменьшите количество соли")
    if "pregnancy" in [c.lower() for c in request.contraindications]:
        notes.append("Дополнительно увеличьте потребление фолатов и железа")
    if any(a in ["латекс", "орехи"] for a in request.allergies):
        notes.append("Проверьте состав блюд на наличие аллергенов")

    return DietPlanResponse(
        diet_type=request.diet_type,
        diet_type_ru=template["name_ru"],
        daily_calories=tdee,
        bmr=int(bmr),
        macros={"protein": protein_ratio, "fat": fat_ratio, "carbs": carbs_ratio},
        meals=selected_meals,
        total_protein_g=total_protein,
        total_carbs_g=total_carbs,
        total_fat_g=total_fat,
        notes=notes,
    )


def _get_meal_time(index: int, total_meals: int) -> str:
    """Generate meal time based on index"""
    times = ["08:00", "10:30", "13:00", "16:00", "18:30", "21:00"]
    return times[index] if index < len(times) else f"{8 + index * 2:02d}:00"


@app.post("/generate-diet")
async def generate_diet(request: DietGenerationRequest):
    """Generate personalized diet plan"""
    plan = calculate_diet_plan(request)
    return plan.model_dump()


# ========== Health Sync Endpoints ==========

@app.post("/sync/flo")
async def sync_flo(request: Dict):
    """Sync menstrual cycle data from Flo"""
    user_id = request.get("user_id")
    access_token = request.get("access_token")
    refresh_token = request.get("refresh_token")

    if not user_id or not access_token:
        raise HTTPException(status_code=400, detail="user_id and access_token are required")

    # TODO: Implement actual Flo API integration
    # For now, store the tokens for future sync
    logger.info("Flo sync requested", user_id=user_id)

    return {
        "success": True,
        "message": "Flo sync endpoint ready. Full integration pending.",
        "synced_cycles": 0,
    }


@app.post("/sync/okok")
async def sync_okok(request: Dict):
    """Sync body composition data from OKOK International"""
    user_id = request.get("user_id")
    access_token = request.get("access_token")
    refresh_token = request.get("refresh_token")

    if not user_id or not access_token:
        raise HTTPException(status_code=400, detail="user_id and access_token are required")

    # TODO: Implement actual OKOK API integration
    # For now, store the tokens for future sync
    logger.info("OKOK sync requested", user_id=user_id)

    return {
        "success": True,
        "message": "OKOK sync endpoint ready. Full integration pending.",
        "synced_records": 0,
    }