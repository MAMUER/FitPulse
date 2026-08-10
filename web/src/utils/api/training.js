import { apiRequest } from './client';

export async function generateTrainingPlan(
  durationWeeks = 4,
  availableDays = [1, 3, 5],
  classificationClass = '',
  confidence = 0
) {
  return apiRequest('/training/generate', {
    method: 'POST',
    body: JSON.stringify({
      duration_weeks: durationWeeks,
      available_days: availableDays,
      class: classificationClass,
      confidence: confidence,
    }),
  });
}

export async function getTrainingPlans(page = 1, pageSize = 10) {
  return apiRequest(`/training/plans?page=${page}&page_size=${pageSize}`);
}

export async function getPlan(planId) {
  return apiRequest(`/training/plans/${planId}`);
}

export async function completeWorkout(planId, workoutId, rating, feedback) {
  return apiRequest('/training/complete', {
    method: 'POST',
    body: JSON.stringify({
      plan_id: planId,
      workout_id: workoutId,
      rating,
      feedback,
    }),
  });
}

export async function getProgress() {
  return apiRequest('/training/progress');
}

export async function getAchievements() {
  return apiRequest('/achievements');
}
