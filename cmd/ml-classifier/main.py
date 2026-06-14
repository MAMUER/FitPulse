"""
ML Classifier API Service
Classifies training type based on physiological parameters
"""

import json
import logging
import os
import shutil
import threading
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional

import joblib
import keras  # standalone Keras 3 (via KERAS_BACKEND=tensorflow)
import numpy as np
from fastapi import FastAPI, HTTPException
from fastapi.responses import Response
from prometheus_client import Gauge
from pydantic import BaseModel, Field

classification_confidence = Gauge(
    "classification_confidence",
    "ML model confidence score for training type classification",
    ["model_version", "class_name"],
)

# Async imports (loaded conditionally)
try:
    import pika  # type: ignore[import]
    import redis  # type: ignore[import]

    ASYNC_DEPS_AVAILABLE = True
except ImportError:
    ASYNC_DEPS_AVAILABLE = False

app = FastAPI(
    title="ML Classifier Service",
    description="Classifies training types based on physiological parameters",
    version="1.0.0",
)

# Global variables
model = None
scaler = None
redis_client = None
rabbitmq_url = None
ml_async_enabled = False

TRAINING_CLASSES = {
    0: {
        "name": "recovery",
        "name_ru": "Восстановление",
        "description": "Низкая нагрузка + высокий HRV + хорошее восстановление",
        "hr_range": "50-65% HRmax",
        "hrv": "Высокий",
        "spo2": "96-99%",
        "recommendations": [
            "Лёгкая активность (ходьба, йога)",
            "Растяжка и мобилизация",
            "Плавание в лёгком темпе",
            "Велопрогулка без напряжения",
        ],
    },
    1: {
        "name": "endurance_e1e2",
        "name_ru": "Базовая выносливость (E1-E2)",
        "description": "Работа ниже лактатного порога, устойчивая кардиореспираторная система",
        "hr_range": "65-80% HRmax",
        "hrv": "Умеренный",
        "spo2": "95-98%",
        "recommendations": [
            "Бег в аэробной зоне",
            "Велосипед (средняя интенсивность)",
            "Плавание (дистанция)",
            "Лыжи/беговые лыжи",
        ],
    },
    2: {
        "name": "threshold_e3",
        "name_ru": "Пороговая выносливость (E3)",
        "description": "Нагрузка вблизи анаэробного порога, баланс лактата",
        "hr_range": "80-90% HRmax",
        "hrv": "Сниженный",
        "spo2": "93-96%",
        "recommendations": [
            "Темповый бег",
            "Интервалы на пороге",
            "Fartlek тренировки",
            "Критическая мощность (велосипед)",
        ],
    },
    3: {
        "name": "strength_hiit",
        "name_ru": "Силовая/HIIT",
        "description": (
            "Высокая вариабельность пульса + постнагрузочная гипертензия + стресс-реакция"
        ),
        "hr_range": "90-100% HRmax",
        "hrv": "Резкое падение",
        "spo2": "90-94%",
        "recommendations": [
            "HIIT интервалы",
            "Силовые тренировки",
            "Спринты",
            "CrossFit/WOD",
        ],
    },
}


class PhysiologicalData(BaseModel):
    """Input physiological parameters"""

    heart_rate: float = Field(..., description="Heart rate (bpm)", ge=40, le=220)
    heart_rate_variability: Optional[float] = Field(
        None, description="HRV (ms)", ge=0, le=200
    )
    spo2: Optional[float] = Field(
        None, description="Blood oxygen saturation (%)", ge=80, le=100
    )
    temperature: Optional[float] = Field(
        None, description="Body temperature (°C)", ge=35.0, le=42.0
    )
    blood_pressure_systolic: Optional[float] = Field(
        None, description="Systolic blood pressure (mmHg)", ge=80, le=250
    )
    blood_pressure_diastolic: Optional[float] = Field(
        None, description="Diastolic blood pressure (mmHg)", ge=50, le=150
    )
    sleep_hours: Optional[float] = Field(None, description="Sleep hours", ge=0, le=24)


class UserProfile(BaseModel):
    """User profile for personalized recommendations"""

    gender: str = Field(..., description="Gender (male/female)")
    age: int = Field(..., description="Age", ge=10, le=100)
    fitness_level: str = Field(
        ..., description="Fitness level (beginner/intermediate/advanced)"
    )
    weight: Optional[float] = Field(None, description="Weight (kg)", ge=30, le=200)
    height: Optional[float] = Field(None, description="Height (cm)", ge=100, le=250)
    health_conditions: Optional[List[str]] = Field(
        None, description="Health conditions/limitations"
    )
    goals: Optional[List[str]] = Field(None, description="Training goals")


class ClassificationRequest(BaseModel):
    """Request for classification"""

    physiological_data: PhysiologicalData
    user_profile: Optional[UserProfile] = None


class ClassificationResponse(BaseModel):
    """Response with classification result"""

    predicted_class: str
    predicted_class_ru: str
    confidence: float
    probabilities: Dict[str, float]
    description: str
    hr_range: str
    recommendations: List[str]
    personalized_notes: Optional[str] = None


def load_models():
    """Load trained models"""
    global model, scaler

    models_dir = Path("/app/models")
    scaler_path = models_dir / "scaler.pkl"

    # === Smart model selection + auto-sync ===
    timestamped_models = sorted(models_dir.glob("classifier_*.keras"))

    if timestamped_models:
        latest_model = timestamped_models[-1]  # newest by filename

        # Always keep classifier.keras in sync with the latest timestamped model
        stable_model = models_dir / "classifier.keras"
        if latest_model != stable_model:
            try:
                shutil.copy2(latest_model, stable_model)
                print(
                    f"🔄 Updated stable model: classifier.keras ← {latest_model.name}"
                )
            except Exception as e:
                print(f" Could not sync classifier.keras: {e}")

        # Clean up older timestamped models (keep only the newest one)
        for old_model in timestamped_models[:-1]:
            try:
                old_model.unlink()
                print(f"🗑️  Removed old model: {old_model.name}")
            except Exception as e:
                print(f" Could not remove {old_model.name}: {e}")

        model_path = str(stable_model)
    else:
        model_path = str(models_dir / "classifier.keras")
    # === end smart selection + auto-sync ===

    if Path(model_path).exists():
        model = keras.models.load_model(model_path)
        print(f"Model loaded from {model_path}")
    else:
        print(f"Model not found at {model_path}")

    if scaler_path.exists():
        scaler = joblib.load(scaler_path)
        print(f"Scaler loaded from {scaler_path}")
    else:
        print(f"Scaler not found at {scaler_path}")


def init_async():
    """Initialize RabbitMQ consumer and Redis client for async mode."""
    global redis_client, rabbitmq_url, ml_async_enabled

    if not ASYNC_DEPS_AVAILABLE:
        print(
            "Async mode requested but pika/redis not installed. Running in sync mode."
        )
        return

    ml_async_enabled = os.environ.get("ML_ASYNC", "").lower() == "true"
    if not ml_async_enabled:
        return

    rabbitmq_url = os.environ.get("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
    redis_host = os.environ.get("REDIS_HOST", "localhost")
    redis_port = int(os.environ.get("REDIS_PORT", 6379))

    try:
        redis_client = redis.Redis(
            host=redis_host, port=redis_port, decode_responses=True
        )
        redis_client.ping()
        print(f"Redis connected at {redis_host}:{redis_port}")
    except Exception as e:
        print(f"Redis connection failed: {e}. Async mode disabled.")
        ml_async_enabled = False
        redis_client = None
        return

    # Start multiple RabbitMQ consumer threads for better concurrency
    num_consumers = int(
        os.environ.get("ML_CONSUMER_THREADS", 3)
    )  # Default to 3 threads
    for i in range(num_consumers):
        consumer_thread = threading.Thread(
            target=_run_rabbitmq_consumer,
            daemon=True,
            name=f"ml-classify-consumer-{i+1}",
        )
        consumer_thread.start()
        print(f"RabbitMQ consumer thread {i+1} started for ml.classify queue")

    print(f"Started {num_consumers} RabbitMQ consumer threads")


def _run_rabbitmq_consumer():
    """Blocking RabbitMQ consumer loop for ml.classify queue."""
    logger = logging.getLogger("ml.classify.consumer")
    while True:
        try:
            credentials = pika.URLParameters(rabbitmq_url)
            connection = pika.BlockingConnection(credentials)
            channel = connection.channel()
            channel.queue_declare(queue="ml.classify", durable=True)
            channel.basic_qos(
                prefetch_count=10
            )  # Increased prefetch from 1 to 10 for better throughput
            channel.basic_consume(
                queue="ml.classify",
                on_message_callback=_on_classify_message,
                auto_ack=False,
            )
            logger.info("Started consuming from ml.classify queue")
            channel.start_consuming()
        except Exception as e:
            logger.error(f"RabbitMQ consumer error: {e}. Reconnecting in 5s...")
            import time

            time.sleep(5)


def _on_classify_message(channel, method, properties, body):
    """Process a classification job from RabbitMQ."""
    job_id = None
    try:
        message = json.loads(body)
        job_id = message.get("job_id")
        if not job_id:
            logger = logging.getLogger("ml.classify.consumer")
            logger.error("Received message without job_id")
            channel.basic_ack(delivery_tag=method.delivery_tag)
            return

        logger = logging.getLogger("ml.classify.consumer")
        logger.info(f"Processing classification job {job_id}")

        # Run the same classification logic as the sync endpoint
        physio_data = message["physiological_data"]

        pd = type("PhysioData", (), physio_data)  # Simple object from dict
        features = [
            pd.heart_rate,
            (
                pd.heart_rate_variability
                if pd.heart_rate_variability is not None
                else 50.0
            ),
            pd.spo2 if pd.spo2 is not None else 98.0,
            pd.temperature if pd.temperature is not None else 37.0,
            (
                pd.blood_pressure_systolic
                if pd.blood_pressure_systolic is not None
                else 120.0
            ),
            (
                pd.blood_pressure_diastolic
                if pd.blood_pressure_diastolic is not None
                else 80.0
            ),
            pd.sleep_hours if pd.sleep_hours is not None else 7.0,
        ]

        features_array = np.array(features).reshape(1, -1)
        features_scaled = scaler.transform(features_array)

        probabilities = model.predict(features_scaled, verbose=0)[0]
        predicted_class = int(np.argmax(probabilities))
        confidence = float(probabilities[predicted_class])

        class_info = TRAINING_CLASSES[predicted_class]

        result = {
            "job_id": job_id,
            "status": "completed",
            "result": {
                "predicted_class": class_info["name"],
                "predicted_class_ru": class_info["name_ru"],
                "confidence": round(confidence, 4),
                "probabilities": {
                    TRAINING_CLASSES[i]["name"]: round(float(p), 4)
                    for i, p in enumerate(probabilities)
                },
                "description": class_info["description"],
                "hr_range": class_info["hr_range"],
                "recommendations": class_info["recommendations"],
            },
            "completed_at": __import__("datetime").datetime.utcnow().isoformat() + "Z",
        }

        # Store result in Redis with TTL 1 hour
        redis_client.setex(f"ml:result:{job_id}", 3600, json.dumps(result))
        logger.info(f"Job {job_id} completed and stored in Redis")

        channel.basic_ack(delivery_tag=method.delivery_tag)

    except Exception as e:
        logger = logging.getLogger("ml.classify.consumer")
        logger.error(f"Error processing job {job_id}: {e}")
        # Reject and requeue
        channel.basic_nack(delivery_tag=method.delivery_tag, requeue=True)


def _do_classify(physiological_data, user_profile=None):
    """Core classification logic, shared between sync and async endpoints."""
    if model is None or scaler is None:
        raise RuntimeError("Models not loaded")

    # Prepare features (7 dimensions)
    features = [
        physiological_data["heart_rate"],
        physiological_data.get("heart_rate_variability") or 50.0,
        physiological_data.get("spo2") or 98.0,
        physiological_data.get("temperature") or 37.0,
        physiological_data.get("blood_pressure_systolic") or 120.0,
        physiological_data.get("blood_pressure_diastolic") or 80.0,
        physiological_data.get("sleep_hours") or 7.0,
    ]

    # Scale features
    features_array = np.array(features).reshape(1, -1)
    features_scaled = scaler.transform(features_array)

    # Predict
    probabilities = model.predict(features_scaled, verbose=0)[0]
    predicted_class = int(np.argmax(probabilities))
    confidence = float(probabilities[predicted_class])

    # Get class info
    class_info = TRAINING_CLASSES[predicted_class]

    # Generate personalized notes
    personalized_notes = None
    if user_profile:
        up = user_profile
        notes = []

        if up.get("fitness_level") == "beginner":
            notes.append("Рекомендуется снизить интенсивность на 10-15%")

        if up.get("age", 0) > 50:
            notes.append("Учитывайте возраст при планировании восстановления")

        if up.get("health_conditions"):
            notes.append(
                f"Проконсультируйтесь с врачом при: {', '.join(up['health_conditions'])}"
            )

        if up.get("goals"):
            goals_lower = [g.lower() for g in up["goals"]]
            if "похудение" in goals_lower and predicted_class == 0:
                notes.append("Для похудения добавьте кардио в зоне E1-E2")

        personalized_notes = " | ".join(notes) if notes else None

    return {
        "predicted_class": class_info["name"],
        "predicted_class_ru": class_info["name_ru"],
        "confidence": round(confidence, 4),
        "probabilities": {
            TRAINING_CLASSES[i]["name"]: round(float(p), 4)
            for i, p in enumerate(probabilities)
        },
        "description": class_info["description"],
        "hr_range": class_info["hr_range"],
        "recommendations": class_info["recommendations"],
        "personalized_notes": personalized_notes,
    }


@app.on_event("startup")
async def startup_event():
    """Load models on startup and initialize async processing if enabled."""
    load_models()
    init_async()


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "model_loaded": model is not None,
        "scaler_loaded": scaler is not None,
        "async_enabled": ml_async_enabled,
    }


@app.get("/classes")
async def get_training_classes():
    """Get available training classes"""
    return TRAINING_CLASSES


@app.post("/classify", response_model=ClassificationResponse)
async def classify_training(request: ClassificationRequest):
    """
    Classify training type based on physiological parameters.
    Synchronous endpoint (always available for backward compatibility).
    """
    if model is None or scaler is None:
        raise HTTPException(status_code=503, detail="Models not loaded")

    try:
        physio_dict = request.physiological_data.dict()
        user_profile_dict = (
            request.user_profile.dict() if request.user_profile else None
        )

        result = _do_classify(physio_dict, user_profile_dict)

        classification_confidence.labels(
            model_version=getattr(model, "name", "unknown"),
            class_name=result["predicted_class"],
        ).set(result["confidence"])

        return ClassificationResponse(**result)

    except RuntimeError as e:
        raise HTTPException(status_code=503, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


# === MLflow metrics export endpoint ===
@app.get("/metrics")
async def get_metrics():
    """Prometheus-style metrics endpoint for MLflow/Prometheus scraping"""
    from prometheus_client import CONTENT_TYPE_LATEST, generate_latest

    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)


# === Model info endpoint ===
@app.get("/model-info")
async def model_info():
    """Get current model metadata"""
    if model is None:
        return {"error": "Model not loaded"}

    return {
        "model_name": getattr(model, "name", "unknown"),
        "input_shape": model.input_shape,
        "output_shape": model.output_shape,
        "total_params": model.count_params(),
        "training_classes": TRAINING_CLASSES,
        "loaded_at": datetime.now().isoformat(),
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8001)
