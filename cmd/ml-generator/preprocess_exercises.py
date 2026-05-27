#!/usr/bin/env python3
import os
import json
import numpy as np
import pandas as pd
from pathlib import Path
from collections import defaultdict

SCRIPT_DIR = Path(__file__).parent.parent.parent
RAW_DIR = SCRIPT_DIR / "datasets" / "raw"
OUTPUT_DIR = SCRIPT_DIR / "datasets" / "processed"
OUTPUT_CSV = OUTPUT_DIR / "training_plans_exercises.csv"

MUSCLE_GROUPS = {
    "abs": 0, "adams apple": 0, "biceps": 1, "triceps": 1, "delts": 2,
    "pectorals": 3, "chest": 3, "upper chest": 3, "lower chest": 3,
    "back": 4, "lats": 4, "upper back": 4, "glutes": 5, "quadriceps": 6,
    "hamstrings": 6, "calves": 7, "lower legs": 7, "forearms": 8,
    "shoulders": 9, " traps": 9, "spine": 10
}

EQUIPMENT_ENCODING = {
    "body weight": 0, "dumbbell": 1, "barbell": 2, "kettlebell": 3,
    "cable": 4, "machine": 5, "sled machine": 5, "smith machine": 5,
    "leverage machine": 5, "weighted": 6, "assisted": 7
}

def load_exercises(exercise_dir):
    exercises = []
    exercises_file = exercise_dir / "exercises.json"
    if not exercises_file.exists():
        return exercises
    
    with open(exercises_file, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    for ex in data:
        muscle_ids = [MUSCLE_GROUPS.get(m.lower().strip(), -1) for m in ex.get("targetMuscles", [])]
        muscle_ids = [m for m in muscle_ids if m >= 0]
        primary_muscle = muscle_ids[0] if muscle_ids else np.random.randint(0, 11)
        
        equip = ex.get("equipments", [""])[0].lower()
        equipment_id = EQUIPMENT_ENCODING.get(equip, np.random.randint(0, 8))
        
        exercises.append({
            "id": ex.get("exerciseId", ""),
            "name": ex.get("name", ""),
            "primary_muscle": primary_muscle,
            "equipment": equipment_id,
            "instructions": ex.get("instructions", [])
        })
    
    return exercises

def generate_training_plans(exercises, n_plans=10000):
    np.random.seed(42)
    plans = []
    
    muscle_to_exercises = defaultdict(list)
    for i, ex in enumerate(exercises):
        muscle_to_exercises[ex["primary_muscle"]].append(i)
    
    for _ in range(n_plans):
        n_exercises = np.random.randint(3, 8)
        selected_indices = np.random.choice(len(exercises), size=n_exercises, replace=False)
        selected_exercises = [exercises[i] for i in selected_indices]
        
        duration = np.random.uniform(20, 90) / 100
        intensity = np.random.uniform(0.3, 1.0)
        rest_ratio = np.random.uniform(0.3, 0.7)
        weekly_freq = np.random.uniform(2, 6) / 7
        
        equipment_dist = np.zeros(8)
        for ex in selected_exercises:
            equipment_dist[ex["equipment"]] += 1
        equipment_dist = equipment_dist / len(selected_exercises)
        
        warmup = np.random.uniform(5, 15) / 100
        cooldown = np.random.uniform(5, 15) / 100
        progression = np.random.uniform(0.1, 0.3)
        age_factor = np.random.uniform(0.5, 1.0)
        fitness_factor = np.random.uniform(0.5, 1.0)
        health_factor = np.random.uniform(0.5, 1.0)
        goal_factor = np.random.uniform(0.5, 1.0)
        
        plan = np.concatenate([
            [duration, intensity, rest_ratio, weekly_freq],
            equipment_dist,
            [warmup, cooldown, progression, age_factor, fitness_factor, health_factor, goal_factor]
        ])
        
        plans.append({
            "plan_vector": plan.tolist(),
            "exercise_count": n_exercises,
            "exercise_names": ";".join([e["name"] for e in selected_exercises])
        })
    
    return plans

def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    
    exercise_dir = RAW_DIR / "exercisedb"
    if not exercise_dir.exists():
        print(f"Exercise data not found at {exercise_dir}")
        return
    
    print("Loading exercises...")
    exercises = load_exercises(exercise_dir)
    print(f"Loaded {len(exercises)} exercises")
    
    print("Generating training plans...")
    plans = generate_training_plans(exercises, n_plans=10000)
    
    df = pd.DataFrame(plans)
    df.to_csv(OUTPUT_CSV, index=False)
    print(f"Saved {len(plans)} training plans to {OUTPUT_CSV}")

if __name__ == "__main__":
    main()