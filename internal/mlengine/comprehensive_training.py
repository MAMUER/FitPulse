"""
ML Comprehensive Training Engine
Комплексная логика генерации тренировок, диеты и адаптивных программ
"""

import random
from datetime import datetime, timedelta
from enum import Enum
from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


# ==========================================
# Enums
# ==========================================

class TrainingGoal(str, Enum):
    WEIGHT_LOSS = "weight_loss"
    MUSCLE_GAIN = "muscle_gain"
    ENDURANCE = "endurance"
    STRENGTH = "strength"
    FLEXIBILITY = "flexibility"
    GENERAL_FITNESS = "general_fitness"
    REHABILITATION = "rehabilitation"


class TrainingLocation(str, Enum):
    HOME = "home"
    GYM = "gym"
    POOL = "pool"
    OUTDOOR = "outdoor"
    MIXED = "mixed"


class UserState(str, Enum):
    RECOVERY = "recovery"
    ENDURANCE_E1E2 = "endurance_e1e2"
    THRESHOLD_E3 = "threshold_e3"
    STRENGTH_HIIT = "strength_hiit"
    OVERTRAINING = "overtraining"
    ILLNESS = "illness"


class DietType(str, Enum):
    BALANCED = "balanced"
    HIGH_PROTEIN = "high_protein"
    LOW_CARB = "low_carb"
    MEDITERRANEAN = "mediterranean"
    VEGETARIAN = "vegetarian"
    VEGAN = "vegan"
    KETO = "keto"
    WEIGHT_LOSS = "weight_loss"
    WEIGHT_GAIN = "weight_gain"
    ENDURANCE_ATHLETE = "endurance_athlete"


class TimeOfDay(str, Enum):
    MORNING = "morning"       # 06:00 - 12:00
    AFTERNOON = "afternoon"   # 12:00 - 18:00
    EVENING = "evening"       # 18:00 - 22:00


# ==========================================
# User Health Profile
# ==========================================

class UserHealthProfile(BaseModel):
    """Расширенный профиль здоровья пользователя"""
    # Базовые данные
    age: int = Field(..., ge=10, le=100, description="Возраст")
    weight: float = Field(..., gt=20, le=300, description="Вес (кг)")
    height: int = Field(..., gt=100, le=250, description="Рост (см)")

    # Биометрия с устройств
    heart_rate: Optional[float] = Field(None, description="Пульс (BPM)")
    ecg_data: Optional[List[float]] = Field(None, description="ЭКГ данные")
    blood_pressure_systolic: Optional[float] = Field(None, description="Систолическое давление")
    blood_pressure_diastolic: Optional[float] = Field(None, description="Диастолическое давление")
    spo2: Optional[float] = Field(None, description="SpO2 (%)")
    temperature: Optional[float] = Field(None, description="Температура (°C)")
    sleep_hours: Optional[float] = Field(None, description="Часы сна")
    hrv: Optional[float] = Field(None, description="HRV (мс)")

    # Здоровье
    diseases: Optional[str] = Field(None, description="Заболевания (текстовое поле)")
    contraindications: Optional[List[str]] = Field(None, description="Противопоказания")
    injuries: Optional[List[str]] = Field(None, description="Травмы")

    # Цели и предпочтения
    training_goal: TrainingGoal = Field(TrainingGoal.GENERAL_FITNESS, description="Цель тренировок")
    training_location: TrainingLocation = Field(TrainingLocation.GYM, description="Место тренировок")
    available_days: List[int] = Field([1, 3, 5], description="Доступные дни недели (0=Пн, 6=Вс)")
    available_time: TimeOfDay = Field(TimeOfDay.EVENING, description="Доступное время суток")

    # Подключенные устройства
    connected_devices: List[str] = Field([], description="Подключенные устройства")

    def has_device_capability(self, capability: str) -> bool:
        """Проверяет, есть ли у устройств определённая возможность"""
        device_capabilities = {
            'apple_watch': ['heart_rate', 'ecg', 'spo2', 'sleep', 'hrv'],
            'samsung_galaxy_watch': ['heart_rate', 'ecg', 'spo2', 'temperature', 'sleep', 'hrv'],
            'huawei_watch_d2': ['heart_rate', 'ecg', 'blood_pressure', 'spo2', 'temperature', 'sleep', 'hrv'],
            'amazfit_trex3': ['heart_rate', 'spo2', 'sleep', 'hrv'],
        }

        for device in self.connected_devices:
            if device in device_capabilities:
                if capability in device_capabilities[device]:
                    return True
        return False

    def calculate_bmi(self) -> float:
        """Рассчитывает BMI"""
        height_m = self.height / 100.0
        return self.weight / (height_m ** 2)

    def calculate_max_heart_rate(self) -> float:
        """Рассчитывает максимальный пульс по формуле: 220 - возраст"""
        return 220 - self.age

    def has_health_risk(self) -> bool:
        """Проверяет наличие рисков для здоровья"""
        if self.diseases and len(self.diseases.strip()) > 0:
            return True
        if self.blood_pressure_systolic and self.blood_pressure_systolic > 140:
            return True
        if self.blood_pressure_systolic and self.blood_pressure_systolic < 90:
            return True
        if self.heart_rate and (self.heart_rate > 100 or self.heart_rate < 50):
            return True
        return False


# ==========================================
# Training Plan Models
# ==========================================

class Exercise(BaseModel):
    """Упражнение в тренировке"""
    name: str
    name_ru: str
    duration_minutes: int
    intensity: float  # 0.0 - 1.0
    sets: Optional[int] = None
    reps: Optional[int] = None
    rest_seconds: int = 60
    description_ru: str = ""
    video_url: Optional[str] = None


class DailyPlan(BaseModel):
    """План на один день"""
    date: str
    day_of_week: int  # 0=Пн, 6=Вс
    time_of_day: TimeOfDay
    training_type: str
    training_type_ru: str
    exercises: List[Exercise]
    total_duration_minutes: int
    intensity_level: float  # 0.0 - 1.0
    notes_ru: str = ""
    is_rest_day: bool = False


class WeeklyPlan(BaseModel):
    """План на неделю"""
    week_number: int
    days: List[DailyPlan]
    total_training_days: int
    total_duration_minutes: int
    average_intensity: float


class TrainingPlan(BaseModel):
    """Полный тренировочный план"""
    user_id: str
    generated_at: str
    plan_duration_weeks: int
    weeks: List[WeeklyPlan]
    diet: Optional['DietPlan'] = None
    recommendations: List[str] = []
    warnings: List[str] = []


# ==========================================
# Diet Models
# ==========================================

class MealItem(BaseModel):
    """Приём пищи"""
    name: str
    name_ru: str
    portion_grams: int
    calories: float
    protein_g: float
    carbs_g: float
    fat_g: float
    time: str  # "08:00", "13:00", etc.


class DailyDiet(BaseModel):
    """Диета на день"""
    day_of_week: int
    meals: List[MealItem]
    total_calories: float
    total_protein_g: float
    total_carbs_g: float
    total_fat_g: float
    water_liters: float
    notes_ru: str = ""


class DietPlan(BaseModel):
    """План питания"""
    diet_type: DietType
    diet_type_ru: str
    daily_calories_target: float
    macros_ratio: Dict[str, float]  # protein, carbs, fat
    days: List[DailyDiet]
    recommendations: List[str] = []
    contraindications: List[str] = []


# ==========================================
# Training State Classifier
# ==========================================

class TrainingStateClassifier:
    """
    Классификатор состояния пользователя на основе биометрии.
    Определяет оптимальный тип тренировки в реальном времени.
    """

    @staticmethod
    def classify_state(profile: UserHealthProfile) -> Dict[str, Any]:
        """
        Классифицирует состояние пользователя.
        Возвращает: class, confidence, recommendations
        """
        scores = {
            UserState.RECOVERY: 0.0,
            UserState.ENDURANCE_E1E2: 0.0,
            UserState.THRESHOLD_E3: 0.0,
            UserState.STRENGTH_HIIT: 0.0,
            UserState.OVERTRAINING: 0.0,
            UserState.ILLNESS: 0.0,
        }

        # Проверка на болезнь (температура)
        if profile.temperature:
            if profile.temperature > 37.5:
                scores[UserState.ILLNESS] += 0.8
            elif profile.temperature > 37.0:
                scores[UserState.ILLNESS] += 0.4

        # Проверка на перетренированность (низкий HRV + высокий пульс покоя)
        if profile.hrv and profile.heart_rate:
            if profile.hrv < 20 and profile.heart_rate > 80:
                scores[UserState.OVERTRAINING] += 0.7
            elif profile.hrv < 30:
                scores[UserState.RECOVERY] += 0.4

        # Оценка готовности к нагрузке
        if profile.sleep_hours:
            if profile.sleep_hours < 6:
                scores[UserState.RECOVERY] += 0.5
            elif profile.sleep_hours >= 7:
                scores[UserState.ENDURANCE_E1E2] += 0.3

        # Пульс и готовность к интенсивности
        if profile.heart_rate:
            max_hr = profile.calculate_max_heart_rate()
            hr_percentage = profile.heart_rate / max_hr

            if hr_percentage < 0.6:
                scores[UserState.RECOVERY] += 0.4
            elif hr_percentage < 0.75:
                scores[UserState.ENDURANCE_E1E2] += 0.5
            elif hr_percentage < 0.85:
                scores[UserState.THRESHOLD_E3] += 0.5
            else:
                scores[UserState.STRENGTH_HIIT] += 0.5

        # SpO2
        if profile.spo2:
            if profile.spo2 < 94:
                scores[UserState.RECOVERY] += 0.6
            elif profile.spo2 < 96:
                scores[UserState.ENDURANCE_E1E2] += 0.3

        # Если есть заболевания — снижаем интенсивность
        if profile.has_health_risk():
            scores[UserState.RECOVERY] += 0.4
            scores[UserState.ENDURANCE_E1E2] += 0.2

        # Находим класс с максимальным score
        best_class = max(scores, key=lambda k: scores[k])
        max_score = scores[best_class]

        # Нормализуем confidence
        total_score = sum(scores.values())
        confidence = max_score / total_score if total_score > 0 else 0.5

        recommendations = TrainingStateClassifier._get_recommendations(best_class, profile)

        return {
            "state": best_class.value,
            "state_ru": TrainingStateClassifier._state_to_russian(best_class),
            "confidence": round(confidence, 2),
            "scores": {k.value: round(v, 2) for k, v in scores.items()},
            "recommendations": recommendations
        }

    @staticmethod
    def _get_recommendations(state: UserState, profile: UserHealthProfile) -> List[str]:
        """Возвращает рекомендации для состояния"""
        recs = {
            UserState.RECOVERY: [
                "Лёгкая активность: ходьба, йога, растяжка",
                "Фокус на восстановление и мобильность",
                "Хороший сон и питание важны сегодня",
                "Избегайте интенсивных нагрузок"
            ],
            UserState.ENDURANCE_E1E2: [
                "Отличный день для базовой выносливости",
                "Бег/велосипед в аэробной зоне (65-80% HRmax)",
                "Длительность: 45-90 минут",
                "Поддерживайте разговорный темп"
            ],
            UserState.THRESHOLD_E3: [
                "Можно работать на пороге",
                "Темповые интервалы (80-90% HRmax)",
                "Длительность: 30-60 минут",
                "Следите за техникой дыхания"
            ],
            UserState.STRENGTH_HIIT: [
                "Готовность к высокой интенсивности",
                "HIIT, силовые, спринты",
                "Длительность: 20-45 минут",
                "Обязательная разминка 10 минут"
            ],
            UserState.OVERTRAINING: [
                "ПРИЗНАКИ ПЕРЕТРЕНИРОВАННОСТИ",
                "Необходим отдых 1-3 дня",
                "Лёгкая активность только",
                "Проверьте сон и питание",
                "При сохранении симптомов — консультация врача"
            ],
            UserState.ILLNESS: [
                "ПРИЗНАКИ ЗАБОЛЕВАНИЯ",
                "Прекратите тренировки до выздоровления",
                "Отдых и обильное питьё",
                "Обратитесь к врачу при температуре > 38°C"
            ]
        }
        return recs.get(state, [])

    @staticmethod
    def _state_to_russian(state: UserState) -> str:
        mapping = {
            UserState.RECOVERY: "Восстановление",
            UserState.ENDURANCE_E1E2: "Базовая выносливость (E1-E2)",
            UserState.THRESHOLD_E3: "Пороговая выносливость (E3)",
            UserState.STRENGTH_HIIT: "Силовая/HIIT",
            UserState.OVERTRAINING: "Перетренированность",
            UserState.ILLNESS: "Заболевание"
        }
        return mapping.get(state, "Не определено")


# ==========================================
# Training Plan Generator
# ==========================================

class TrainingPlanGenerator:
    """
    Генератор тренировочных планов.
    Создаёт программу тренировок на основе:
    - Состояния пользователя (из ML классификатора)
    - Целей тренировок
    - Места тренировок
    - Доступных дней
    - Устройств (возможностей биометрии)
    """

    # Библиотека упражнений по месту тренировок
    EXERCISES_BY_LOCATION = {
        TrainingLocation.HOME: {
            "warmup": [
                {"name": "jumping_jacks", "name_ru": "Прыжки на месте", "duration": 5},
                {"name": "arm_circles", "name_ru": "Вращение руками", "duration": 3},
                {"name": "high_knees", "name_ru": "Подъем коленей", "duration": 3},
            ],
            "main": [
                {"name": "pushups", "name_ru": "Отжимания", "sets": 3, "reps": 15, "rest": 60, "intensity": 0.6},
                {"name": "squats", "name_ru": "Приседания", "sets": 4, "reps": 20, "rest": 60, "intensity": 0.6},
                {"name": "plank", "name_ru": "Планка", "sets": 3, "duration": 60, "rest": 45, "intensity": 0.5},
                {"name": "lunges", "name_ru": "Выпады", "sets": 3, "reps": 12, "rest": 60, "intensity": 0.6},
                {"name": "burpees", "name_ru": "Бёрпи", "sets": 3, "reps": 10, "rest": 90, "intensity": 0.8},
                {
                    "name": "mountain_climbers",
                    "name_ru": "Альпинист",
                    "sets": 3,
                    "duration": 45,
                    "rest": 60,
                    "intensity": 0.7,
                },
            ],
            "cooldown": [
                {"name": "stretching", "name_ru": "Растяжка", "duration": 10},
                {"name": "deep_breathing", "name_ru": "Глубокое дыхание", "duration": 5},
            ]
        },
        TrainingLocation.GYM: {
            "warmup": [
                {"name": "treadmill_walk", "name_ru": "Ходьба на беговой дорожке", "duration": 10},
                {"name": "dynamic_stretch", "name_ru": "Динамическая растяжка", "duration": 5},
            ],
            "main": [
                {"name": "bench_press", "name_ru": "Жим лёжа", "sets": 4, "reps": 10, "rest": 90, "intensity": 0.7},
                {"name": "deadlift", "name_ru": "Становая тяга", "sets": 4, "reps": 8, "rest": 120, "intensity": 0.8},
                {"name": "leg_press", "name_ru": "Жим ногами", "sets": 4, "reps": 12, "rest": 90, "intensity": 0.7},
                {"name": "lat_pulldown", "name_ru": "Тяга верхнего блока", "sets": 3, "reps": 12, "rest": 60, "intensity": 0.6},
                {"name": "shoulder_press", "name_ru": "Жим плечами", "sets": 3, "reps": 10, "rest": 90, "intensity": 0.7},
                {"name": "cable_rows", "name_ru": "Тяга блока", "sets": 3, "reps": 12, "rest": 60, "intensity": 0.6},
            ],
            "cooldown": [
                {"name": "foam_rolling", "name_ru": "Фоам-роллинг", "duration": 10},
                {"name": "static_stretching", "name_ru": "Статическая растяжка", "duration": 10},
            ]
        },
        TrainingLocation.POOL: {
            "warmup": [
                {"name": "easy_swim", "name_ru": "Лёгкое плавание", "duration": 10},
            ],
            "main": [
                {
                    "name": "freestyle_intervals",
                    "name_ru": "Интервалы вольным стилем",
                    "sets": 8,
                    "duration": 120,
                    "rest": 30,
                    "intensity": 0.7,
                },
                {"name": "breaststroke", "name_ru": "Брасс", "duration": 20, "intensity": 0.5},
                {"name": "backstroke", "name_ru": "На спине", "duration": 15, "intensity": 0.6},
                {"name": "kickboard_drills", "name_ru": "Работа с доской", "sets": 6, "duration": 60, "rest": 30, "intensity": 0.6},
            ],
            "cooldown": [
                {"name": "easy_swim", "name_ru": "Лёгкое плавание", "duration": 5},
                {"name": "pool_stretching", "name_ru": "Растяжка в бассейне", "duration": 5},
            ]
        },
        TrainingLocation.OUTDOOR: {
            "warmup": [
                {"name": "brisk_walk", "name_ru": "Быстрая ходьба", "duration": 5},
                {"name": "leg_swings", "name_ru": "Махи ногами", "duration": 3},
            ],
            "main": [
                {"name": "running", "name_ru": "Бег", "duration": 30, "intensity": 0.7},
                {"name": "cycling", "name_ru": "Велосипед", "duration": 45, "intensity": 0.6},
                {"name": "hill_sprints", "name_ru": "Спринты в гору", "sets": 8, "duration": 30, "rest": 90, "intensity": 0.9},
                {"name": "bodyweight_circuit", "name_ru": "Круговая тренировка", "sets": 4, "duration": 180, "rest": 60, "intensity": 0.7},
            ],
            "cooldown": [
                {"name": "walk_recovery", "name_ru": "Ходьба", "duration": 5},
                {"name": "stretching", "name_ru": "Растяжка", "duration": 10},
            ]
        }
    }

    @staticmethod
    def generate_plan(
        user_id: str,
        profile: UserHealthProfile,
        state_classification: Dict[str, Any],
        duration_weeks: int = 4
    ) -> TrainingPlan:
        """Генерирует полный тренировочный план"""

        state = UserState(state_classification["state"])
        weeks = []

        for week_num in range(duration_weeks):
            week_plan = TrainingPlanGenerator._generate_week(
                week_num, profile, state, duration_weeks
            )
            weeks.append(week_plan)

            # Прогрессия: увеличиваем интенсивность каждую неделю
            if state == UserState.ENDURANCE_E1E2:
                state = UserState.THRESHOLD_E3 if week_num > duration_weeks // 2 else state
            elif state == UserState.RECOVERY:
                state = UserState.ENDURANCE_E1E2 if week_num > 1 else state

        warnings = []
        if profile.has_health_risk():
            warnings.append("У вас есть факторы риска. Проконсультируйтесь с врачом перед тренировками.")

        recommendations = TrainingPlanGenerator._get_plan_recommendations(profile, state)

        return TrainingPlan(
            user_id=user_id,
            generated_at=datetime.utcnow().isoformat(),
            plan_duration_weeks=duration_weeks,
            weeks=weeks,
            recommendations=recommendations,
            warnings=warnings
        )

    @staticmethod
    def _generate_week(
        week_num: int,
        profile: UserHealthProfile,
        state: UserState,
        total_weeks: int
    ) -> WeeklyPlan:
        """Генерирует план на одну неделю"""

        days = []
        total_duration = 0
        training_days = 0

        intensity_multiplier = 1.0 + (week_num * 0.05)  # +5% каждую неделю

        for day in range(7):
            is_training_day = day in profile.available_days

            if is_training_day:
                daily = TrainingPlanGenerator._generate_training_day(
                    day, profile, state, intensity_multiplier
                )
                training_days += 1
            else:
                daily = TrainingPlanGenerator._generate_rest_day(day)

            total_duration += daily.total_duration_minutes
            days.append(daily)

        return WeeklyPlan(
            week_number=week_num + 1,
            days=days,
            total_training_days=training_days,
            total_duration_minutes=total_duration,
            average_intensity=round(total_duration / (training_days * 60) if training_days > 0 else 0, 2)
        )

    @staticmethod
    def _generate_training_day(
        day: int,
        profile: UserHealthProfile,
        state: UserState,
        intensity_multiplier: float
    ) -> DailyPlan:
        """Генерирует тренировочный день"""

        location = profile.training_location
        exercises_db = TrainingPlanGenerator.EXERCISES_BY_LOCATION.get(location, {})

        # Определяем тип тренировки по состоянию
        training_type_map = {
            UserState.RECOVERY: ("recovery", "Восстановление"),
            UserState.ENDURANCE_E1E2: ("endurance", "Выносливость"),
            UserState.THRESHOLD_E3: ("threshold", "Пороговая"),
            UserState.STRENGTH_HIIT: ("hiit", "HIIT/Силовая"),
        }

        train_type, train_type_ru = training_type_map.get(state, ("general", "Общая"))

        exercises = []

        # Warmup
        warmups = exercises_db.get("warmup", [])
        for ex in warmups:
            exercises.append(Exercise(
                name=ex["name"],
                name_ru=ex["name_ru"],
                duration_minutes=ex["duration"],
                intensity=0.3,
                rest_seconds=30,
                description_ru=f"Разминка: {ex['name_ru']}"
            ))

        # Main workout
        mains = exercises_db.get("main", [])
        intensity_base = 0.5
        if state == UserState.ENDURANCE_E1E2:
            intensity_base = 0.6
        elif state == UserState.THRESHOLD_E3:
            intensity_base = 0.75
        elif state == UserState.STRENGTH_HIIT:
            intensity_base = 0.85

        for ex in mains[:3]:  # 3 основных упражнения
            ex_intensity = min(1.0, float(ex.get("intensity", 0.6)) * intensity_multiplier * intensity_base)

            exercises.append(Exercise(
                name=ex["name"],
                name_ru=ex["name_ru"],
                duration_minutes=ex.get("duration", 0),
                intensity=round(ex_intensity, 2),
                sets=ex.get("sets"),
                reps=ex.get("reps"),
                rest_seconds=ex.get("rest", 60),
                description_ru=f"Основное упражнение: {ex['name_ru']}"
            ))

        # Cooldown
        cooldowns = exercises_db.get("cooldown", [])
        for ex in cooldowns:
            exercises.append(Exercise(
                name=ex["name"],
                name_ru=ex["name_ru"],
                duration_minutes=ex["duration"],
                intensity=0.3,
                rest_seconds=30,
                description_ru=f"Заминка: {ex['name_ru']}"
            ))

        total_duration = sum(ex.duration_minutes for ex in exercises)

        return DailyPlan(
            date=(datetime.utcnow() + timedelta(days=day)).strftime("%Y-%m-%d"),
            day_of_week=day,
            time_of_day=profile.available_time,
            training_type=train_type,
            training_type_ru=train_type_ru,
            exercises=exercises,
            total_duration_minutes=total_duration,
            intensity_level=round(intensity_base * intensity_multiplier, 2),
            notes_ru=f"Тренировка: {train_type_ru}. Следите за пульсом.",
        )

    @staticmethod
    def _generate_rest_day(day: int) -> DailyPlan:
        """Генерирует день отдыха"""

        return DailyPlan(
            date=(datetime.utcnow() + timedelta(days=day)).strftime("%Y-%m-%d"),
            day_of_week=day,
            time_of_day=TimeOfDay.MORNING,
            training_type="rest",
            training_type_ru="Отдых",
            exercises=[],
            total_duration_minutes=0,
            intensity_level=0.0,
            is_rest_day=True,
            notes_ru="День отдыха. Лёгкая прогулка или растяжка по желанию."
        )

    @staticmethod
    def _get_plan_recommendations(profile: UserHealthProfile, state: UserState) -> List[str]:
        """Возвращает рекомендации для плана"""
        recs = [
            "Разминайтесь минимум 5-10 минут перед тренировкой",
            "Пейте воду во время тренировки",
            "Следите за техникой выполнения упражнений"
        ]

        if profile.age > 50:
            recs.append("После 50 лет увеличьте время разминки и заминки")

        if profile.has_health_risk():
            recs.append("Контролируйте пульс во время тренировки")
            recs.append("При головокрусти или боли остановитесь")

        return recs


# ==========================================
# Diet Plan Generator
# ==========================================

class DietPlanGenerator:
    """Генератор планов питания"""

    # Шаблоны диет
    DIET_TEMPLATES = {
        DietType.BALANCED: {
            "name_ru": "Сбалансированное питание",
            "macros": {"protein": 0.30, "carbs": 0.45, "fat": 0.25},
            "meals": [
                {"time": "07:00", "name": "breakfast", "name_ru": "Завтрак", "cal_pct": 0.25},
                {"time": "10:00", "name": "snack1", "name_ru": "Перекус", "cal_pct": 0.10},
                {"time": "13:00", "name": "lunch", "name_ru": "Обед", "cal_pct": 0.35},
                {"time": "16:00", "name": "snack2", "name_ru": "Полдник", "cal_pct": 0.10},
                {"time": "19:00", "name": "dinner", "name_ru": "Ужин", "cal_pct": 0.20},
            ]
        },
        DietType.HIGH_PROTEIN: {
            "name_ru": "Высокобелковая диета",
            "macros": {"protein": 0.40, "carbs": 0.35, "fat": 0.25},
            "meals": [
                {"time": "07:00", "name": "breakfast", "name_ru": "Завтрак", "cal_pct": 0.25},
                {"time": "10:00", "name": "snack1", "name_ru": "Перекус", "cal_pct": 0.15},
                {"time": "13:00", "name": "lunch", "name_ru": "Обед", "cal_pct": 0.30},
                {"time": "16:00", "name": "snack2", "name_ru": "Полдник", "cal_pct": 0.15},
                {"time": "19:00", "name": "dinner", "name_ru": "Ужин", "cal_pct": 0.15},
            ]
        },
        DietType.WEIGHT_LOSS: {
            "name_ru": "Диета для похудения",
            "macros": {"protein": 0.35, "carbs": 0.35, "fat": 0.30},
            "meals": [
                {"time": "08:00", "name": "breakfast", "name_ru": "Завтрак", "cal_pct": 0.25},
                {"time": "11:00", "name": "snack1", "name_ru": "Перекус", "cal_pct": 0.10},
                {"time": "14:00", "name": "lunch", "name_ru": "Обед", "cal_pct": 0.35},
                {"time": "18:00", "name": "dinner", "name_ru": "Ужин", "cal_pct": 0.30},
            ]
        },
    }

    # Примеры блюд
    MEAL_EXAMPLES = {
        "breakfast": [
            {"name": "oatmeal", "name_ru": "Овсянка с фруктами", "cal": 350, "protein": 12, "carbs": 60, "fat": 8},
            {"name": "eggs", "name_ru": "Яичница с овощами", "cal": 300, "protein": 18, "carbs": 10, "fat": 22},
        ],
        "lunch": [
            {"name": "chicken_rice", "name_ru": "Курица с рисом", "cal": 550, "protein": 40, "carbs": 60, "fat": 15},
            {"name": "fish_salad", "name_ru": "Рыба с салатом", "cal": 450, "protein": 35, "carbs": 20, "fat": 25},
        ],
        "dinner": [
            {"name": "turkey_veg", "name_ru": "Индейка с овощами", "cal": 400, "protein": 35, "carbs": 25, "fat": 18},
            {"name": "cottage_cheese", "name_ru": "Творог с ягодами", "cal": 250, "protein": 25, "carbs": 20, "fat": 8},
        ],
        "snack": [
            {"name": "nuts", "name_ru": "Орехи", "cal": 200, "protein": 6, "carbs": 8, "fat": 18},
            {"name": "yogurt", "name_ru": "Греческий йогурт", "cal": 150, "protein": 15, "carbs": 12, "fat": 5},
        ]
    }

    @staticmethod
    def generate_diet(profile: UserHealthProfile) -> DietPlan:
        """Генерирует план питания"""

        # Определяем тип диеты
        diet_type = DietPlanGenerator._select_diet_type(profile)
        template = DietPlanGenerator.DIET_TEMPLATES.get(
            diet_type, DietPlanGenerator.DIET_TEMPLATES[DietType.BALANCED]
        )

        # Рассчитываем калории
        bmr = DietPlanGenerator._calculate_bmr(profile)
        activity_multiplier = (
            1.3 + (profile.training_goal == TrainingGoal.ENDURANCE) * 0.3
        )
        daily_calories = bmr * activity_multiplier

        # Генерируем дни
        days = []
        for day in range(7):
            daily_diet = DietPlanGenerator._generate_day_diet(
                day, template, daily_calories, profile
            )
            days.append(daily_diet)

        contraindications = []
        if profile.diseases:
            contraindications.append("Учтите ваши заболевания при выборе продуктов")

        return DietPlan(
            diet_type=diet_type,
            diet_type_ru=template["name_ru"],
            daily_calories_target=round(daily_calories),
            macros_ratio=template["macros"],
            days=days,
            recommendations=[
                "Пейте минимум 2 литра воды в день",
                "Ешьте каждые 3-4 часа",
                "Избегайте обработанных продуктов",
            ],
            contraindications=contraindications,
        )

    @staticmethod
    def _select_diet_type(profile: UserHealthProfile) -> DietType:
        """Выбирает тип диеты на основе профиля"""
        if profile.training_goal == TrainingGoal.WEIGHT_LOSS:
            return DietType.WEIGHT_LOSS
        elif profile.training_goal == TrainingGoal.MUSCLE_GAIN:
            return DietType.HIGH_PROTEIN
        elif profile.training_goal == TrainingGoal.ENDURANCE:
            return DietType.ENDURANCE_ATHLETE
        elif profile.calculate_bmi() > 25:
            return DietType.WEIGHT_LOSS
        else:
            return DietType.BALANCED

    @staticmethod
    def _calculate_bmr(profile: UserHealthProfile) -> float:
        """Рассчитывает базовый метаболизм (формула Миффлина-Сан Жеора)"""
        if profile.gender if hasattr(profile, "gender") else True:  # male
            bmr = 10 * profile.weight + 6.25 * profile.height - 5 * profile.age + 5
        else:
            bmr = 10 * profile.weight + 6.25 * profile.height - 5 * profile.age - 161
        return max(1200, bmr)

    @staticmethod
    def _generate_day_diet(
        day: int,
        template: Dict,
        daily_calories: float,
        profile: UserHealthProfile
    ) -> DailyDiet:
        """Генерирует диету на один день"""

        meals = []
        total_cal = 0
        total_protein = 0
        total_carbs = 0
        total_fat = 0

        for meal_template in template["meals"]:
            meal_cal = daily_calories * meal_template["cal_pct"]

            # Выбираем случайное блюдо из категории
            meal_category = meal_template["name"]
            if "snack" in meal_category:
                options = DietPlanGenerator.MEAL_EXAMPLES["snack"]
            else:
                options = DietPlanGenerator.MEAL_EXAMPLES.get(meal_category, [])

            if options:
                dish = random.choice(options)
                # Масштабируем под нужные калории
                scale = meal_cal / dish["cal"]

                meals.append(
                    MealItem(
                        name=dish["name"],
                        name_ru=dish["name_ru"],
                        portion_grams=int(200 * scale),
                        calories=round(meal_cal),
                        protein_g=round(dish["protein"] * scale, 1),
                        carbs_g=round(dish["carbs"] * scale, 1),
                        fat_g=round(dish["fat"] * scale, 1),
                        time=meal_template["time"],
                    )
                )

                total_cal += meal_cal
                total_protein += dish["protein"] * scale
                total_carbs += dish["carbs"] * scale
                total_fat += dish["fat"] * scale

        return DailyDiet(
            day_of_week=day,
            meals=meals,
            total_calories=round(total_cal),
            total_protein_g=round(total_protein, 1),
            total_carbs_g=round(total_carbs, 1),
            total_fat_g=round(total_fat, 1),
            water_liters=2.0 + profile.weight * 0.01,
            notes_ru=f"День {day + 1}. Следите за балансом макронутриентов.",
        )


# ==========================================
# Adaptive Plan Modifier (Daily)
# ==========================================

class AdaptivePlanModifier:
    """
    Ежедневно адаптирует план тренировок на основе отклонений показателей.
    """

    @staticmethod
    def adapt_plan(
        original_plan: TrainingPlan,
        current_biometrics: Dict[str, float],
        profile: UserHealthProfile
    ) -> Dict[str, Any]:
        """
        Адаптирует план на основе текущих показателей.
        Возвращает: adapted_plan, changes, warnings
        """
        changes = []
        warnings = []

        # Проверка пульса
        if "heart_rate" in current_biometrics:
            hr = current_biometrics["heart_rate"]
            max_hr = profile.calculate_max_heart_rate()

            if hr > max_hr * 0.9:
                changes.append("Снизить интенсивность today — пульс выше нормы")
                warnings.append("Пульс слишком высокий. Отдохните или снизьте нагрузку.")
            elif hr < 50:
                changes.append("Пульс ниже нормы — проверьте датчик")

        # Проверка SpO2
        if "spo2" in current_biometrics:
            if current_biometrics["spo2"] < 92:
                changes.append("SpO2 низкий — прекратите тренировку")
                warnings.append("Низкая сатурация! Остановитесь и обратитесь к врачу.")

        # Проверка температуры
        if "temperature" in current_biometrics:
            if current_biometrics["temperature"] > 37.5:
                changes.append("Температура повышена — пропустите тренировку")
                warnings.append("При температуре тренироваться нельзя!")

        # Проверка сна
        if profile.sleep_hours and profile.sleep_hours < 5:
            changes.append("Недосып — замените интенсивную тренировку на лёгкую")

        return {
            "original_plan": original_plan.dict(),
            "changes": changes,
            "warnings": warnings,
            "adapted": len(changes) > 0
        }