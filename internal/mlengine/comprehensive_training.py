"""
Comprehensive Training Plan Generator

This module provides advanced training plan generation based on user biometrics,
health profile, and real-time state classification.
"""

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
    RECOVERY = "recovery"
    WEIGHT_GAIN = "weight_gain"
    ENDURANCE_ATHLETE = "endurance_athlete"


class TrainingLocation(str, Enum):
    HOME = "home"
    GYM = "gym"
    POOL = "pool"
    OUTDOOR = "outdoor"


class DietType(str, Enum):
    BALANCED = "balanced"
    HIGH_PROTEIN = "high_protein"
    WEIGHT_LOSS = "weight_loss"
    LOW_CARB = "low_carb"
    VEGETARIAN = "vegetarian"


class UserState(str, Enum):
    RECOVERY = "recovery"
    ENDURANCE_E1E2 = "endurance_e1e2"
    THRESHOLD_E3 = "threshold_e3"
    STRENGTH_HIIT = "strength_hiit"
    OVERTRAINING = "overtraining"
    ILLNESS = "illness"


class TimeOfDay(str, Enum):
    MORNING = "morning"  # 06:00 - 12:00
    AFTERNOON = "afternoon"  # 12:00 - 18:00
    EVENING = "evening"  # 18:00 - 22:00


# ==========================================
# User Health Profile
# ==========================================


class UserHealthProfile(BaseModel):
    """Extended user health profile"""

    # Basic data
    age: int = Field(..., ge=10, le=100, description="Age")
    weight: float = Field(..., gt=20, le=300, description="Weight (kg)")
    height: int = Field(..., gt=100, le=250, description="Height (cm)")

    # Biometrics from devices
    heart_rate: Optional[float] = Field(None, description="Heart rate (BPM)")
    ecg_data: Optional[List[float]] = Field(None, description="ECG data")
    blood_pressure_systolic: Optional[float] = Field(
        None, description="Systolic blood pressure"
    )
    blood_pressure_diastolic: Optional[float] = Field(
        None, description="Diastolic blood pressure"
    )
    spo2: Optional[float] = Field(None, description="SpO2 (%)")
    temperature: Optional[float] = Field(None, description="Temperature (°C)")
    sleep_hours: Optional[float] = Field(None, description="Sleep hours")
    hrv: Optional[float] = Field(None, description="HRV (ms)")

    # Health conditions
    diseases: Optional[str] = Field(None, description="Diseases (text field)")
    contraindications: Optional[List[str]] = Field(
        None, description="Contraindications"
    )
    injuries: Optional[List[str]] = Field(None, description="Injuries")

    # Goals and preferences
    training_goal: TrainingGoal = Field(
        TrainingGoal.GENERAL_FITNESS, description="Training goal"
    )
    training_location: TrainingLocation = Field(
        TrainingLocation.GYM, description="Training location"
    )
    available_days: List[int] = Field(
        [1, 3, 5], description="Available days of week (0=Mon, 6=Sun)"
    )
    available_time: TimeOfDay = Field(
        TimeOfDay.EVENING, description="Available time of day"
    )

    # Connected devices
    connected_devices: List[str] = Field([], description="Connected devices")

    def has_device_capability(self, capability: str) -> bool:
        """Check if devices have a certain capability"""
        device_capabilities = {
            "apple_watch": ["heart_rate", "ecg", "spo2", "sleep", "hrv"],
            "samsung_galaxy_watch": [
                "heart_rate",
                "ecg",
                "spo2",
                "temperature",
                "sleep",
                "hrv",
            ],
            "huawei_watch_d2": [
                "heart_rate",
                "ecg",
                "blood_pressure",
                "spo2",
                "temperature",
                "sleep",
                "hrv",
            ],
            "amazfit_trex3": ["heart_rate", "spo2", "sleep", "hrv"],
        }

        for device in self.connected_devices:
            if device in device_capabilities:
                if capability in device_capabilities[device]:
                    return False

        return False

    def calculate_bmi(self) -> float:
        """Calculate BMI"""
        height_m = self.height / 100.0
        return self.weight / (height_m**2)

    def calculate_max_heart_rate(self) -> float:
        """Calculate max heart rate using formula: 220 - age"""
        return 220 - self.age

    def has_health_risk(self) -> bool:
        """Check if user has health risk factors"""
        if self.age > 60:
            return True
        if self.calculate_bmi() > 30 or self.calculate_bmi() < 18.5:
            return True
        if self.diseases:
            return True
        return False


# ==========================================
# Training Plan Models
# ==========================================


class Exercise(BaseModel):
    """Exercise in a workout"""

    name: str
    name_ru: str
    duration_minutes: int
    intensity: float  # 0.0 - 1.0
    sets: Optional[int] = None
    reps: Optional[int] = None
    rest_seconds: int = 60
    description_ru: Optional[str] = None
    video_url: Optional[str] = None


class DailyPlan(BaseModel):
    """Plan for one day"""

    date: str
    day_of_week: int  # 0=Mon, 6=Sun
    time_of_day: TimeOfDay
    training_type: str
    training_type_ru: str
    exercises: List[Exercise]
    total_duration_minutes: int
    intensity_level: float  # 0.0 - 1.0
    notes_ru: str = ""
    is_rest_day: bool = False


class WeeklyPlan(BaseModel):
    """Plan for one week"""

    week_number: int
    days: List[DailyPlan]
    total_training_days: int
    total_duration_minutes: int
    average_intensity: float


class TrainingPlan(BaseModel):
    """Complete training plan"""

    user_id: str
    generated_at: str
    plan_duration_weeks: int
    weeks: List[WeeklyPlan]
    diet: Optional["DietPlan"] = None
    recommendations: List[str] = []
    warnings: List[str] = []


# ==========================================
# Diet Models
# ==========================================


class MealItem(BaseModel):
    """Meal item"""

    name: str
    name_ru: str
    portion_grams: int
    calories: float
    protein_g: float
    carbs_g: float
    fat_g: float
    time: str  # "08:00", "13:00", etc.


class DailyDiet(BaseModel):
    """Diet for one day"""

    day_of_week: int
    meals: List[MealItem]
    total_calories: float
    total_protein_g: float
    total_carbs_g: float
    total_fat_g: float
    notes: str = ""
    notes_ru: str = ""


class DietPlan(BaseModel):
    """Diet plan"""

    diet_type: DietType
    diet_type_ru: str
    daily_calories_target: float
    macros_ratio: Dict[str, float]  # protein, carbs, fat
    days: List[DailyDiet]


# ==========================================
# Training State Classifier
# ==========================================


class TrainingStateClassifier:
    """
    Classifier of user state based on biometrics.
    Determines optimal training type in real-time.
    """

    @staticmethod
    def classify_state(
        profile: UserHealthProfile,
        current_biometrics: Dict[str, float],
    ) -> Dict[str, Any]:
        """Classify user state based on biometrics"""
        scores = {}

        # Heart rate analysis
        if "heart_rate" in current_biometrics:
            hr = current_biometrics["heart_rate"]
            max_hr = profile.calculate_max_heart_rate()
            hr_ratio = hr / max_hr

            if hr_ratio < 0.5:
                scores[UserState.RECOVERY] = 0.8
            elif hr_ratio < 0.7:
                scores[UserState.ENDURANCE_E1E2] = 0.7
            elif hr_ratio < 0.85:
                scores[UserState.THRESHOLD_E3] = 0.6
            else:
                scores[UserState.STRENGTH_HIIT] = 0.5

        # HRV analysis (higher HRV = better recovery)
        if "hrv" in current_biometrics:
            hrv = current_biometrics["hrv"]
            if hrv > 80:
                scores[UserState.RECOVERY] = scores.get(UserState.RECOVERY, 0) + 0.3
            elif hrv < 30:
                scores[UserState.OVERTRAINING] = 0.7

        # Sleep analysis
        if "sleep_hours" in current_biometrics:
            sleep = current_biometrics["sleep_hours"]
            if sleep < 5:
                scores[UserState.RECOVERY] = scores.get(UserState.RECOVERY, 0) + 0.4
            elif sleep > 8:
                scores[UserState.STRENGTH_HIIT] = (
                    scores.get(UserState.STRENGTH_HIIT, 0) + 0.2
                )

        # Temperature check
        if "temperature" in current_biometrics:
            temp = current_biometrics["temperature"]
            if temp > 37.5:
                scores[UserState.ILLNESS] = 0.9

        # SpO2 check
        if "spo2" in current_biometrics:
            spo2 = current_biometrics["spo2"]
            if spo2 < 92:
                scores[UserState.ILLNESS] = scores.get(UserState.ILLNESS, 0) + 0.5

        # Find best state
        if not scores:
            best_class = UserState.ENDURANCE_E1E2
            max_score = 0.5
        else:
            best_class = max(scores, key=scores.get)  # type: ignore
            max_score = scores[best_class]

        # Normalize confidence
        total_score = sum(scores.values())
        confidence = max_score / total_score if total_score > 0 else 0.5

        recommendations = TrainingStateClassifier._get_recommendations(
            best_class, profile
        )

        return {
            "state": best_class.value,
            "state_ru": TrainingStateClassifier._state_to_russian(best_class),
            "confidence": round(confidence, 2),
            "scores": {k.value: round(v, 2) for k, v in scores.items()},
            "recommendations": recommendations,
        }

    @staticmethod
    def _get_recommendations(
        state: UserState, profile: UserHealthProfile
    ) -> List[str]:
        """Return recommendations for state"""
        recs = {
            UserState.RECOVERY: [
                "Light activity: walking, yoga, stretching",
                "Focus on recovery and mobility",
                "Good sleep and nutrition are important today",
                "Avoid intense loads",
            ],
            UserState.ENDURANCE_E1E2: [
                "Great day for base endurance",
                "Running/cycling in aerobic zone (65-80% HRmax)",
                "Duration: 45-90 minutes",
                "Maintain conversational pace",
            ],
            UserState.THRESHOLD_E3: [
                "Can work at threshold",
                "Tempo intervals (80-90% HRmax)",
                "Duration: 30-60 minutes",
                "Monitor breathing technique",
            ],
            UserState.STRENGTH_HIIT: [
                "Ready for high intensity",
                "HIIT, strength, sprints",
                "Duration: 20-45 minutes",
                "Mandatory warm-up 10 minutes",
            ],
            UserState.OVERTRAINING: [
                "SIGNS OF OVERTRAINING",
                "Need rest 1-3 days",
                "Light activity only",
                "Check sleep and nutrition",
                "If symptoms persist - consult doctor",
            ],
            UserState.ILLNESS: [
                "SIGNS OF ILLNESS",
                "Stop training until recovery",
                "Rest and plenty of fluids",
                "See doctor if temperature > 38°C",
            ],
        }
        return recs.get(state, [])

    @staticmethod
    def _state_to_russian(state: UserState) -> str:
        mapping = {
            UserState.RECOVERY: "Recovery",
            UserState.ENDURANCE_E1E2: "Base endurance (E1-E2)",
            UserState.THRESHOLD_E3: "Threshold endurance (E3)",
            UserState.STRENGTH_HIIT: "Strength/HIIT",
            UserState.OVERTRAINING: "Overtraining",
            UserState.ILLNESS: "Illness",
        }
        return mapping.get(state, "Not determined")


# ==========================================
# Training Plan Generator
# ==========================================


class TrainingPlanGenerator:
    """
    Training plan generator.
    Creates training program based on:
    - User health profile
    - Current state classification
    - Training location
    - Available days
    """

    EXERCISES_BY_LOCATION = {
        TrainingLocation.HOME: {
            "warmup": [
                {"name": "jumping_jacks", "name_ru": "Jumping jacks", "duration": 5},
                {"name": "arm_circles", "name_ru": "Arm circles", "duration": 3},
                {"name": "high_knees", "name_ru": "High knees", "duration": 3},
            ],
            "main": [
                {
                    "name": "pushups",
                    "name_ru": "Push-ups",
                    "sets": 3,
                    "reps": 15,
                    "rest": 60,
                    "intensity": 0.6,
                },
                {
                    "name": "squats",
                    "name_ru": "Squats",
                    "sets": 4,
                    "reps": 20,
                    "rest": 60,
                    "intensity": 0.6,
                },
                {
                    "name": "plank",
                    "name_ru": "Plank",
                    "sets": 3,
                    "duration": 60,
                    "rest": 45,
                    "intensity": 0.5,
                },
                {
                    "name": "lunges",
                    "name_ru": "Lunges",
                    "sets": 3,
                    "reps": 12,
                    "rest": 60,
                    "intensity": 0.6,
                },
                {
                    "name": "burpees",
                    "name_ru": "Burpees",
                    "sets": 3,
                    "reps": 10,
                    "rest": 90,
                    "intensity": 0.8,
                },
                {
                    "name": "mountain_climbers",
                    "name_ru": "Mountain climber",
                    "sets": 3,
                    "duration": 45,
                    "rest": 30,
                    "intensity": 0.7,
                },
            ],
            "cooldown": [
                {"name": "stretching", "name_ru": "Stretching", "duration": 10},
                {
                    "name": "deep_breathing",
                    "name_ru": "Deep breathing",
                    "duration": 5,
                },
            ],
        },
        TrainingLocation.GYM: {
            "warmup": [
                {
                    "name": "treadmill_walk",
                    "name_ru": "Treadmill walk",
                    "duration": 10,
                },
                {
                    "name": "dynamic_stretch",
                    "name_ru": "Dynamic stretching",
                    "duration": 5,
                },
            ],
            "main": [
                {
                    "name": "bench_press",
                    "name_ru": "Bench press",
                    "sets": 4,
                    "reps": 10,
                    "rest": 90,
                    "intensity": 0.7,
                },
                {
                    "name": "deadlift",
                    "name_ru": "Deadlift",
                    "sets": 4,
                    "reps": 8,
                    "rest": 120,
                    "intensity": 0.8,
                },
                {
                    "name": "leg_press",
                    "name_ru": "Leg press",
                    "sets": 4,
                    "reps": 12,
                    "rest": 90,
                    "intensity": 0.7,
                },
                {
                    "name": "lat_pulldown",
                    "name_ru": "Lat pulldown",
                    "sets": 3,
                    "reps": 12,
                    "rest": 60,
                    "intensity": 0.6,
                },
                {
                    "name": "shoulder_press",
                    "name_ru": "Shoulder press",
                    "sets": 3,
                    "reps": 10,
                    "rest": 90,
                    "intensity": 0.7,
                },
                {
                    "name": "cable_rows",
                    "name_ru": "Cable rows",
                    "sets": 3,
                    "reps": 12,
                    "rest": 60,
                    "intensity": 0.6,
                },
            ],
            "cooldown": [
                {"name": "foam_rolling", "name_ru": "Foam rolling", "duration": 10},
                {
                    "name": "static_stretching",
                    "name_ru": "Static stretching",
                    "duration": 10,
                },
            ],
        },
        TrainingLocation.POOL: {
            "warmup": [
                {"name": "easy_swim", "name_ru": "Easy swimming", "duration": 10},
            ],
            "main": [
                {
                    "name": "freestyle",
                    "name_ru": "Freestyle",
                    "sets": 8,
                    "duration": 120,
                    "rest": 30,
                    "intensity": 0.7,
                },
                {
                    "name": "breaststroke",
                    "name_ru": "Breaststroke",
                    "duration": 20,
                    "intensity": 0.5,
                },
                {
                    "name": "backstroke",
                    "name_ru": "Backstroke",
                    "duration": 15,
                    "intensity": 0.6,
                },
                {
                    "name": "kickboard_drills",
                    "name_ru": "Kickboard drills",
                    "sets": 6,
                    "duration": 60,
                    "rest": 30,
                    "intensity": 0.6,
                },
            ],
            "cooldown": [
                {"name": "easy_swim", "name_ru": "Easy swimming", "duration": 5},
                {
                    "name": "pool_stretching",
                    "name_ru": "Pool stretching",
                    "duration": 5,
                },
            ],
        },
        TrainingLocation.OUTDOOR: {
            "warmup": [
                {"name": "brisk_walk", "name_ru": "Brisk walk", "duration": 5},
                {"name": "leg_swings", "name_ru": "Leg swings", "duration": 3},
            ],
            "main": [
                {"name": "running", "name_ru": "Running", "duration": 30, "intensity": 0.7},
                {
                    "name": "cycling",
                    "name_ru": "Cycling",
                    "duration": 45,
                    "intensity": 0.6,
                },
                {
                    "name": "hill_sprints",
                    "name_ru": "Hill sprints",
                    "sets": 8,
                    "duration": 30,
                    "rest": 90,
                    "intensity": 0.9,
                },
                {
                    "name": "bodyweight_circuit",
                    "name_ru": "Bodyweight circuit",
                    "sets": 4,
                    "duration": 180,
                    "rest": 60,
                    "intensity": 0.7,
                },
            ],
            "cooldown": [
                {"name": "walk_recovery", "name_ru": "Walking", "duration": 5},
                {"name": "stretching", "name_ru": "Stretching", "duration": 10},
            ],
        },
    }

    @staticmethod
    def generate_plan(
        user_id: str,
        profile: UserHealthProfile,
        state_classification: Dict[str, Any],
        duration_weeks: int = 4,
    ) -> TrainingPlan:
        """Generate complete training plan"""

        state = UserState(state_classification["state"])
        weeks = []

        for week_num in range(duration_weeks):
            week_plan = TrainingPlanGenerator._generate_week(
                week_num, profile, state, duration_weeks
            )
            weeks.append(week_plan)

            # Progression: increase intensity each week
            if state == UserState.ENDURANCE_E1E2:
                state = (
                    UserState.THRESHOLD_E3 if week_num > duration_weeks // 2 else state
                )
            elif state == UserState.RECOVERY:
                state = UserState.ENDURANCE_E1E2 if week_num > 1 else state

        warnings = []
        if profile.has_health_risk():
            warnings.append(
                "You have risk factors. Consult doctor before training."
            )

        recommendations = TrainingPlanGenerator._get_plan_recommendations(
            profile, state
        )

        return TrainingPlan(
            user_id=user_id,
            generated_at=datetime.utcnow().isoformat(),
            plan_duration_weeks=duration_weeks,
            weeks=weeks,
            recommendations=recommendations,
            warnings=warnings,
        )

    @staticmethod
    def _generate_week(
        week_num: int, profile: UserHealthProfile, state: UserState, total_weeks: int
    ) -> WeeklyPlan:
        """Generate plan for one week"""

        days = []
        total_duration = 0
        training_days = 0

        for day in range(7):
            if day in profile.available_days:
                intensity_multiplier = 1.0 + (week_num / total_weeks) * 0.2
                daily_plan = TrainingPlanGenerator._generate_training_day(
                    day, profile, state, intensity_multiplier
                )
                days.append(daily_plan)
                total_duration += daily_plan.total_duration_minutes
                training_days += 1
            else:
                days.append(TrainingPlanGenerator._generate_rest_day(day))

        return WeeklyPlan(
            week_number=week_num + 1,
            days=days,
            total_training_days=training_days,
            total_duration_minutes=total_duration,
            average_intensity=round(
                total_duration / (training_days * 60) if training_days > 0 else 0, 2
            ),
        )

    @staticmethod
    def _generate_training_day(
        day: int,
        profile: UserHealthProfile,
        state: UserState,
        intensity_multiplier: float,
    ) -> DailyPlan:
        """Generate training day"""

        location = profile.training_location
        exercises_db = TrainingPlanGenerator.EXERCISES_BY_LOCATION.get(location, {})

        time_of_day = profile.available_time
        training_type = state.value
        training_type_ru = TrainingStateClassifier._state_to_russian(state)

        exercises = []

        # Warmup
        warmups = exercises_db.get("warmup", [])
        for ex in warmups:
            exercises.append(
                Exercise(
                    name=ex["name"],
                    name_ru=ex["name_ru"],
                    duration_minutes=ex["duration"],
                    intensity=0.3,
                    rest_seconds=30,
                    description_ru=f"Warm-up: {ex['name_ru']}",
                )
            )

        # Main workout
        mains = exercises_db.get("main", [])
        intensity_base = 0.5
        if state == UserState.ENDURANCE_E1E2:
            intensity_base = 0.6
        elif state == UserState.THRESHOLD_E3:
            intensity_base = 0.75
        elif state == UserState.STRENGTH_HIIT:
            intensity_base = 0.85

        for ex in mains[:3]:  # 3 main exercises
            ex_intensity = min(
                1.0,
                float(ex.get("intensity", 0.6)) * intensity_multiplier * intensity_base,
            )

            exercises.append(
                Exercise(
                    name=ex["name"],
                    name_ru=ex["name_ru"],
                    duration_minutes=ex.get("duration", 0),
                    intensity=round(ex_intensity, 2),
                    sets=ex.get("sets"),
                    reps=ex.get("reps"),
                    rest_seconds=ex.get("rest", 60),
                    description_ru=f"Main exercise: {ex['name_ru']}",
                )
            )

        # Cooldown
        cooldowns = exercises_db.get("cooldown", [])
        for ex in cooldowns:
            exercises.append(
                Exercise(
                    name=ex["name"],
                    name_ru=ex["name_ru"],
                    duration_minutes=ex["duration"],
                    intensity=0.3,
                    rest_seconds=30,
                    description_ru=f"Cool-down: {ex['name_ru']}",
                )
            )

        total_duration = sum(ex.duration_minutes for ex in exercises)

        return DailyPlan(
            date=(datetime.utcnow() + timedelta(days=day)).strftime("%Y-%m-%d"),
            day_of_week=day,
            time_of_day=time_of_day,
            training_type=training_type,
            training_type_ru=training_type_ru,
            exercises=exercises,
            total_duration_minutes=total_duration,
            intensity_level=round(
                sum(ex.intensity for ex in exercises) / len(exercises)
                if exercises
                else 0,
                2,
            ),
            notes_ru="",
        )

    @staticmethod
    def _generate_rest_day(day: int) -> DailyPlan:
        """Generate rest day"""
        return DailyPlan(
            date=(datetime.utcnow() + timedelta(days=day)).strftime("%Y-%m-%d"),
            day_of_week=day,
            time_of_day=TimeOfDay.MORNING,
            training_type="rest",
            training_type_ru="Rest",
            exercises=[],
            total_duration_minutes=0,
            intensity_level=0.0,
            is_rest_day=True,
            notes_ru="Rest day. Light walk or stretching if desired.",
        )

    @staticmethod
    def _get_plan_recommendations(
        profile: UserHealthProfile, state: UserState
    ) -> List[str]:
        """Return recommendations for plan"""
        recs = [
            "Warm up at least 5-10 minutes before training",
            "Drink water during training",
            "Monitor exercise technique",
        ]

        if profile.age > 50:
            recs.append("After 50, increase warm-up and cool-down time")

        return recs


# ==========================================
# Diet Plan Generator
# ==========================================


class DietPlanGenerator:
    """Diet plan generator"""

    # Diet templates
    DIET_TEMPLATES = {
        DietType.BALANCED: {
            "name_ru": "Balanced nutrition",
            "macros": {"protein": 0.30, "carbs": 0.45, "fat": 0.25},
            "meals": [
                {
                    "time": "07:00",
                    "name": "breakfast",
                    "name_ru": "Breakfast",
                    "cal_pct": 0.25,
                },
                {
                    "time": "10:00",
                    "name": "snack1",
                    "name_ru": "Snack",
                    "cal_pct": 0.10,
                },
                {"time": "13:00", "name": "lunch", "name_ru": "Lunch", "cal_pct": 0.35},
                {
                    "time": "16:00",
                    "name": "snack2",
                    "name_ru": "Afternoon snack",
                    "cal_pct": 0.10,
                },
                {"time": "19:00", "name": "dinner", "name_ru": "Dinner", "cal_pct": 0.20},
            ],
        },
        DietType.HIGH_PROTEIN: {
            "name_ru": "High protein diet",
            "macros": {"protein": 0.40, "carbs": 0.35, "fat": 0.25},
            "meals": [
                {
                    "time": "07:00",
                    "name": "breakfast",
                    "name_ru": "Breakfast",
                    "cal_pct": 0.25,
                },
                {
                    "time": "10:00",
                    "name": "snack1",
                    "name_ru": "Snack",
                    "cal_pct": 0.15,
                },
                {"time": "13:00", "name": "lunch", "name_ru": "Lunch", "cal_pct": 0.30},
                {
                    "time": "16:00",
                    "name": "snack2",
                    "name_ru": "Afternoon snack",
                    "cal_pct": 0.15,
                },
                {"time": "19:00", "name": "dinner", "name_ru": "Dinner", "cal_pct": 0.15},
            ],
        },
        DietType.WEIGHT_LOSS: {
            "name_ru": "Weight loss diet",
            "macros": {"protein": 0.35, "carbs": 0.35, "fat": 0.30},
            "meals": [
                {
                    "time": "08:00",
                    "name": "breakfast",
                    "name_ru": "Breakfast",
                    "cal_pct": 0.25,
                },
                {
                    "time": "11:00",
                    "name": "snack1",
                    "name_ru": "Snack",
                    "cal_pct": 0.10,
                },
                {"time": "14:00", "name": "lunch", "name_ru": "Lunch", "cal_pct": 0.35},
                {"time": "18:00", "name": "dinner", "name_ru": "Dinner", "cal_pct": 0.30},
            ],
        },
    }

    # Meal examples
    MEAL_EXAMPLES = {
        "breakfast": [
            {
                "name": "oatmeal",
                "name_ru": "Oatmeal with fruits",
                "cal": 350,
                "protein": 12,
                "carbs": 60,
                "fat": 8,
            },
            {
                "name": "eggs",
                "name_ru": "Eggs with vegetables",
                "cal": 300,
                "protein": 18,
                "carbs": 10,
                "fat": 22,
            },
        ],
        "lunch": [
            {
                "name": "chicken_rice",
                "name_ru": "Chicken with rice",
                "cal": 550,
                "protein": 40,
                "carbs": 60,
                "fat": 15,
            },
            {
                "name": "fish_salad",
                "name_ru": "Fish with salad",
                "cal": 450,
                "protein": 35,
                "carbs": 20,
                "fat": 25,
            },
        ],
        "dinner": [
            {
                "name": "turkey_veg",
                "name_ru": "Turkey with vegetables",
                "cal": 400,
                "protein": 35,
                "carbs": 25,
                "fat": 18,
            },
            {
                "name": "cottage_cheese",
                "name_ru": "Cottage cheese with berries",
                "cal": 250,
                "protein": 25,
                "carbs": 20,
                "fat": 8,
            },
        ],
        "snack": [
            {
                "name": "nuts",
                "name_ru": "Nuts",
                "cal": 200,
                "protein": 6,
                "carbs": 8,
                "fat": 18,
            },
            {
                "name": "yogurt",
                "name_ru": "Greek yogurt",
                "cal": 150,
                "protein": 15,
                "carbs": 12,
                "fat": 5,
            },
        ],
    }

    @staticmethod
    def generate_diet(profile: UserHealthProfile) -> DietPlan:
        """Generate diet plan"""
        diet_type = DietType.BALANCED
        if profile.training_goal == TrainingGoal.WEIGHT_LOSS:
            diet_type = DietType.WEIGHT_LOSS
        elif profile.training_goal in [
            TrainingGoal.MUSCLE_GAIN,
            TrainingGoal.STRENGTH,
        ]:
            diet_type = DietType.HIGH_PROTEIN

        template = DietPlanGenerator.DIET_TEMPLATES[diet_type]
        daily_calories = DietPlanGenerator._calculate_daily_calories(profile)

        days = []
        for day in range(7):
            day_diet = DietPlanGenerator._generate_day_diet(
                day, template, daily_calories, profile
            )
            days.append(day_diet)

        return DietPlan(
            diet_type=diet_type,
            diet_type_ru=template["name_ru"],
            daily_calories_target=daily_calories,
            macros_ratio=template["macros"],
            days=days,
        )

    @staticmethod
    def _calculate_daily_calories(profile: UserHealthProfile) -> float:
        """Calculate daily calorie needs using Mifflin-St Jeor equation"""
        if profile.height > 0 and profile.weight > 0 and profile.age > 0:
            bmr = (
                10 * profile.weight + 6.25 * profile.height - 5 * profile.age + 5
            )
        else:
            bmr = 2000
        return max(1200, bmr)

    @staticmethod
    def _generate_day_diet(
        day: int, template: Dict, daily_calories: float, profile: UserHealthProfile
    ) -> DailyDiet:
        """Generate diet for one day"""

        meals = []
        total_cal = 0
        total_protein = 0
        total_carbs = 0
        total_fat = 0

        for meal_template in template["meals"]:
            meal_cal = daily_calories * meal_template["cal_pct"]
            meal_name = meal_template["name"]

            # Select meal from examples
            meal_examples = DietPlanGenerator.MEAL_EXAMPLES.get(
                meal_name, DietPlanGenerator.MEAL_EXAMPLES["snack"]
            )
            meal_data = meal_examples[day % len(meal_examples)]

            # Scale portion to match calorie target
            scale_factor = meal_cal / meal_data["cal"]
            portion_grams = int(200 * scale_factor)

            meal = MealItem(
                name=meal_data["name"],
                name_ru=meal_data["name_ru"],
                portion_grams=portion_grams,
                calories=round(meal_cal, 1),
                protein_g=round(meal_data["protein"] * scale_factor, 1),
                carbs_g=round(meal_data["carbs"] * scale_factor, 1),
                fat_g=round(meal_data["fat"] * scale_factor, 1),
                time=meal_template["time"],
            )
            meals.append(meal)

            total_cal += meal_cal
            total_protein += meal_data["protein"] * scale_factor
            total_carbs += meal_data["carbs"] * scale_factor
            total_fat += meal_data["fat"] * scale_factor

        return DailyDiet(
            day_of_week=day,
            meals=meals,
            total_calories=round(total_cal, 1),
            total_protein_g=round(total_protein, 1),
            total_carbs_g=round(total_carbs, 1),
            total_fat_g=round(total_fat, 1),
            notes="",
            notes_ru="",
        )


# ==========================================
# Adaptive Plan Modifier (Daily)
# ==========================================


class AdaptivePlanModifier:
    """
    Daily adapts training plan based on deviations in indicators.
    """

    @staticmethod
    def adapt_plan(
        original_plan: TrainingPlan,
        current_biometrics: Dict[str, float],
        profile: UserHealthProfile,
    ) -> Dict[str, Any]:
        """
        Adapt plan based on current indicators.
        Returns: adapted_plan, changes, warnings
        """
        changes = []
        warnings = []

        # Heart rate check
        if "heart_rate" in current_biometrics:
            hr = current_biometrics["heart_rate"]
            max_hr = profile.calculate_max_heart_rate()

            if hr > max_hr * 0.9:
                changes.append("Reduce intensity today - heart rate above normal")
                warnings.append(
                    "Heart rate too high. Rest or reduce load."
                )
            elif hr < 50:
                changes.append("Heart rate below normal - check sensor")

        # SpO2 check
        if "spo2" in current_biometrics:
            spo2 = current_biometrics["spo2"]
            if spo2 < 92:
                warnings.append("Low SpO2. Consult doctor.")
                changes.append("Skip today's training")

        return {
            "original_plan": original_plan.dict(),
            "changes": changes,
            "warnings": warnings,
            "adapted": len(changes) > 0,
        }