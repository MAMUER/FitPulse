#!/usr/bin/env python3
"""Преобразование реального набора тренировок в формат обучения условной ГАН.

Источник: Gym Members Exercise Dataset (V. Khorasani, 2024, Apache 2.0),
973 записи с ЧСС, длительностью, калориями, типом тренировки и профилем.
Файл: datasets/raw/gym_members_exercise_tracking.csv

На выходе — datasets/processed/training_plans.csv в формате
p0..p18,class,age,gender,weight,height,goal, где 19-мерный вектор плана
получен детерминированным отображением реальных параметров тренировки.
"""

import argparse
import csv
from pathlib import Path

PLAN_DIM = 19

# Тип тренировки -> индекс упражнения (p6..p12).
WORKOUT_TYPE_INDEX = {
    "Strength": 0,
    "HIIT": 1,
    "Cardio": 2,
    "Yoga": 3,
}
OTHER_TYPE_INDEX = 4

# Базовые параметры по типу: (подходы, повторы, отдых в секундах).
TYPE_PARAMS = {
    "Strength": (4, 10, 75.0),
    "HIIT": (4, 18, 35.0),
    "Cardio": (2, 1, 0.0),
    "Yoga": (1, 1, 0.0),
}


def clamp01(value: float) -> float:
    return max(0.0, min(1.0, value))


def goal_from_experience(experience: float) -> float:
    if experience >= 3:
        return 0.0
    if experience == 2:
        return 1.0
    return 2.0


def classify(
    intensity: float,
    calories: float,
    duration: float,
    frequency: float,
    workout_type: str,
) -> int:
    if calories < 600.0 and duration < 1.0:
        return 5
    if frequency >= 5.0:
        return 4
    if workout_type == "HIIT" or intensity >= 0.85:
        return 3
    if intensity >= 0.72:
        return 2
    if intensity >= 0.65:
        return 1
    return 0


def build_vector(row: dict) -> tuple[list[float], list[float]]:
    age = float(row["Age"])
    gender = 1.0 if row["Gender"].strip().lower() == "male" else 0.0
    weight = float(row["Weight (kg)"])
    height = float(row["Height (m)"]) * 100.0
    max_bpm = float(row["Max_BPM"])
    avg_bpm = float(row["Avg_BPM"])
    resting = float(row["Resting_BPM"])
    duration = float(row["Session_Duration (hours)"])
    calories = float(row["Calories_Burned"])
    workout_type = row["Workout_Type"].strip()
    frequency = float(row["Workout_Frequency (days/week)"])
    experience = float(row["Experience_Level"])
    fat = float(row["Fat_Percentage"])
    water = float(row["Water_Intake (liters)"])
    goal = goal_from_experience(experience)

    intensity = clamp01((avg_bpm - resting) / max(max_bpm - resting, 1.0))

    sets, reps, rest = TYPE_PARAMS.get(workout_type, (2, 8, 60.0))

    plan = [0.0] * PLAN_DIM
    plan[0] = round(intensity, 4)
    plan[1] = round(clamp01(duration / 3.0), 4)
    plan[2] = round(clamp01(frequency / 7.0), 4)
    plan[3] = round(clamp01(sets / 6.0), 4)
    plan[4] = round(clamp01(reps / 25.0), 4)
    plan[5] = round(clamp01(rest / 120.0), 4)

    type_index = WORKOUT_TYPE_INDEX.get(workout_type, OTHER_TYPE_INDEX)
    for i in range(7):
        plan[6 + i] = 1.0 if i == type_index else 0.05

    plan[13] = round(clamp01(calories / 1500.0), 4)
    plan[14] = round(clamp01(water / 5.0), 4)
    plan[15] = round(clamp01(fat / 50.0), 4)
    plan[16] = round(clamp01(experience / 3.0), 4)
    plan[17] = round(clamp01((max_bpm - 150.0) / 50.0), 4)
    plan[18] = round(clamp01(intensity * 0.5 + experience / 6.0), 4)

    class_index = classify(
        intensity,
        calories,
        duration,
        frequency,
        workout_type,
    )

    profile = [class_index, age, gender, weight, height, goal]
    return plan, profile


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--input",
        type=Path,
        default=Path("datasets/raw/gym_members_exercise_tracking.csv"),
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("datasets/processed/training_plans.csv"),
    )
    args = parser.parse_args()

    rows: list[list[float]] = []
    with args.input.open(newline="", encoding="utf-8") as fh:
        for row in csv.DictReader(fh):
            plan, profile = build_vector(row)
            rows.append(plan + [float(v) for v in profile])

    header = [f"p{i}" for i in range(PLAN_DIM)] + [
        "class",
        "age",
        "gender",
        "weight",
        "height",
        "goal",
    ]
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w", newline="", encoding="utf-8") as fh:
        writer = csv.writer(fh)
        writer.writerow(header)
        writer.writerows(rows)

    counts: dict[int, int] = {}
    for record in rows:
        class_index = int(record[PLAN_DIM])
        counts[class_index] = counts.get(class_index, 0) + 1
    print(f"Записано {len(rows)} строк в {args.output}")
    print("Распределение классов:", dict(sorted(counts.items())))


if __name__ == "__main__":
    main()
