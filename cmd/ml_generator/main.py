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
    1: "endurance_e1e2",
    2: "threshold_e3",
    3: "strength_hiit",
}

TRAINING_TEMPLATES = {
    "recovery": {
        "duration_range": (20, 45),
        "intensity_range": (0.3, 0.5),
        "exercises": ["walking", "yoga", "stretching", "light_swimming", "mobility"],
        "rest_ratio": 0.7,
        "name_ru": "Восстановление",
    },
    "endurance_e1e2": {
        "duration_range": (45, 90),
        "intensity_range": (0.5, 0.7),
        "exercises": ["running", "cycling", "swimming", "rowing", "hiking"],
        "rest_ratio": 0.4,
        "name_ru": "Базовая выносливость",
    },
    "threshold_e3": {
        "duration_range": (30, 60),
        "intensity_range": (0.7, 0.85),
        "exercises": ["tempo_run", "threshold_intervals", "fartlek", "critical_power"],
        "rest_ratio": 0.3,
        "name_ru": "Пороговая выносливость",
    },
    "strength_hiit": {
        "duration_range": (20, 45),
        "intensity_range": (0.85, 1.0),
        "exercises": ["hiit", "strength", "sprints", "crossfit", "plyometrics"],
        "rest_ratio": 0.5,
        "name_ru": "Силовая/HIIT",
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


class PlanGenerationRequest(BaseModel):
    """Request for training plan generation"""
    model_config = ConfigDict(strict=True)

    training_class: str = Field(..., description="Training class from classifier")
    user_profile: UserProfile
    preferences: Optional[Dict] = Field(None, description="User preferences")


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


def generate_from_noise(noise: np.ndarray) -> np.ndarray:
    """Generate plan from noise vector using ONNX model"""
    if generator_session is None:
        raise RuntimeError("Generator not loaded")
    
    input_name = generator_session.get_inputs()[0].name
    result = generator_session.run(None, {input_name: noise})
    return result[0][0]


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

        up = UserProfile(**user_profile_dict)
        plan = await _do_generate_plan(training_class, up, preferences)

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


async def _do_generate_plan(training_class, user_profile, preferences=None):
    """Core plan generation logic, shared between sync and async endpoints."""
    if generator_session is None:
        raise RuntimeError("Generator not loaded")

    noise = np.random.normal(0, 1, (1, 64)).astype(np.float32)
    plan_vector = generate_from_noise(noise)
    plan = decode_plan(plan_vector, training_class, user_profile)
    return plan


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


def encode_user_profile(profile: UserProfile) -> np.ndarray:
    """Encode user profile to model input (10 dimensions)"""
    age_norm = (profile.age - 10) / 90
    fitness_map = {"beginner": 0.3, "intermediate": 0.6, "advanced": 0.9}
    fitness_norm = fitness_map.get(profile.fitness_level, 0.5)

    weight_norm = (profile.weight or 70) / 200
    height_norm = (profile.height or 170) / 250

    goal_encoded = 0.5
    if profile.goals:
        goals_lower = [g.lower() for g in profile.goals]
        if "похудение" in goals_lower or "weight_loss" in goals_lower:
            goal_encoded = 0.2
        elif "набор массы" in goals_lower or "muscle_gain" in goals_lower:
            goal_encoded = 0.8
        elif "реабилитация" in goals_lower or "rehabilitation" in goals_lower:
            goal_encoded = 0.1

    health_flag = 1.0 if profile.health_conditions else 0.0
    gender_encoded = 1.0 if profile.gender.lower() == "male" else 0.0

    sleep_score = 0.5
    nutrition_score = 0.5
    if profile.lifestyle:
        sleep_score = profile.lifestyle.get("sleep_hours", 7) / 10
        nutrition_score = profile.lifestyle.get("nutrition_quality", 0.5)

    encoded = np.array(
        [
            age_norm,
            fitness_norm,
            weight_norm,
            height_norm,
            goal_encoded,
            health_flag,
            gender_encoded,
            sleep_score,
            nutrition_score,
            0.5,
        ],
        dtype=np.float32,
    )

    return encoded.reshape(1, -1)


def decode_plan(plan_vector: np.ndarray, training_class: str, user_profile: UserProfile) -> dict:
    """Decode model output (19 dimensions) to training plan"""
    template = TRAINING_TEMPLATES.get(training_class, TRAINING_TEMPLATES["endurance_e1e2"])

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


@app.post("/generate-plan", response_model=TrainingPlan)
async def generate_plan(request: PlanGenerationRequest):
    """Generate personalized training plan (synchronous endpoint)"""
    if generator_session is None:
        raise HTTPException(status_code=503, detail="Generator not loaded")

    try:
        plan = await _do_generate_plan(request.training_class, request.user_profile, request.preferences)
        classification_confidence.labels(
            model_version="diffusion_v1",
            class_name=request.training_class,
        ).set(1.0)

        return TrainingPlan(**plan)

    except RuntimeError as e:
        raise HTTPException(status_code=503, detail=str(e))
    except Exception as e:
        logger.error("Plan generation failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8002, loop="uvloop")