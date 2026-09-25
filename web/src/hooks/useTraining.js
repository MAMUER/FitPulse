import { useCallback } from 'react';
import * as api from '../services/api';

export function useTraining({ state, update, notify }) {
  const setTrainingPeriod = useCallback(
    (period) => {
      update({ trainingPeriod: period });
    },
    [update]
  );

  const toggleWorkout = useCallback(
    (id) => {
      update({ selectedWorkout: state.selectedWorkout === id ? null : id });
    },
    [state.selectedWorkout, update]
  );

  const startWorkout = useCallback(
    async (title) => {
      const completed = state.completedWorkouts || [];
      if (completed.includes(title)) {
        notify('Эта тренировка уже отмечена как выполненная');
        return;
      }
      try {
        await api.completeWorkout('default', title, 5, 'Completed');
        const newCompleted = [...completed, title];
        const newDaily = (state.dailyCompleted || 0) + 1;
        const td = state.trainingData?.day;
        const newTrainingData = td
          ? {
              ...td,
              completed: Math.min(td.goal || 3, (td.completed || 0) + 1),
              minutes: (td.minutes || 0) + 42,
              calories: (td.calories || 0) + 420,
            }
          : td;
        update({
          completedWorkouts: newCompleted,
          dailyCompleted: newDaily,
          points: (state.points || 0) + 40,
          xp: (state.xp || 0) + 40,
          trainingMinutes: (state.trainingMinutes || 0) + 42,
          caloriesBurned: (state.caloriesBurned || 0) + 420,
          trainingData: newTrainingData
            ? { ...state.trainingData, day: newTrainingData }
            : state.trainingData,
        });
        notify(`${title}\n✓ Тренировка завершена. +40 XP`);
      } catch {
        notify('Ошибка завершения тренировки');
      }
    },
    [
      state.completedWorkouts,
      state.dailyCompleted,
      state.points,
      state.xp,
      state.trainingMinutes,
      state.caloriesBurned,
      state.trainingData,
      update,
      notify,
    ]
  );

  const loadTrainingPlans = useCallback(async () => {
    try {
      const data = await api.loadTrainingPlans();
      if (Array.isArray(data?.plans)) {
        update({ trainingPlans: data.plans });
      }
    } catch {
      // ignore
    }
  }, [update]);

  const generatePlan = useCallback(
    async (params = {}) => {
      try {
        const data = await api.generatePlan(params);
        if (data) {
          await loadTrainingPlans();
          notify('План создан');
          return data;
        }
      } catch {
        notify('Ошибка генерации плана');
      }
      return null;
    },
    [loadTrainingPlans, notify]
  );

  const getPlanDetails = useCallback(
    async (planId) => {
      try {
        const data = await api.getPlanDetails(planId);
        return data;
      } catch {
        notify('Ошибка загрузки плана');
        return null;
      }
    },
    [notify]
  );

  const loadProgress = useCallback(async () => {
    try {
      await api.loadProgress();
    } catch {
      // ignore
    }
  }, []);

  const completeWorkout = useCallback(
    async (planId, workoutId, rating = 5, feedback = '') => {
      try {
        await api.completeWorkout(planId, workoutId, rating, feedback);
      } catch {
        notify('Ошибка завершения тренировки');
      }
    },
    [notify]
  );

  return {
    setTrainingPeriod,
    toggleWorkout,
    startWorkout,
    loadTrainingPlans,
    generatePlan,
    getPlanDetails,
    loadProgress,
    completeWorkout,
  };
}
