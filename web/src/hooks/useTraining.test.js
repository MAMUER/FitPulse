import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../services/api';
import { useTraining } from './useTraining';

vi.mock('../services/api', () => ({
  completeWorkout: vi.fn(),
  loadTrainingPlans: vi.fn(),
  generatePlan: vi.fn(),
  getPlanDetails: vi.fn(),
  loadProgress: vi.fn(),
}));

function makeState(overrides = {}) {
  return {
    completedWorkouts: [],
    dailyCompleted: 0,
    selectedWorkout: null,
    trainingPeriod: 'week',
    points: 100,
    xp: 200,
    trainingMinutes: 0,
    caloriesBurned: 0,
    trainingData: {
      day: { goal: 3, completed: 0, minutes: 0, calories: 0 },
    },
    ...overrides,
  };
}

let state;
let update;
let notify;
let result;

function setup(overrides = {}) {
  state = makeState(overrides);
  update = vi.fn();
  notify = vi.fn();
  result = renderHook(() => useTraining({ state, update, notify })).result;
}

beforeEach(() => {
  vi.resetAllMocks();
});

describe('useTraining', () => {
  it('returns all training callbacks', () => {
    setup();
    expect(Object.keys(result.current).sort()).toEqual([
      'completeWorkout',
      'generatePlan',
      'getPlanDetails',
      'loadProgress',
      'loadTrainingPlans',
      'setTrainingPeriod',
      'startWorkout',
      'toggleWorkout',
    ]);
  });

  it('setTrainingPeriod updates period', () => {
    setup();
    result.current.setTrainingPeriod('month');
    expect(update).toHaveBeenCalledWith({ trainingPeriod: 'month' });
  });

  it('toggleWorkout selects and deselects workout', () => {
    setup();
    result.current.toggleWorkout('w1');
    expect(update).toHaveBeenCalledWith({ selectedWorkout: 'w1' });
  });

  it('toggleWorkout clears selection for same id', () => {
    setup({ selectedWorkout: 'w1' });
    result.current.toggleWorkout('w1');
    expect(update).toHaveBeenCalledWith({ selectedWorkout: null });
  });

  it('startWorkout completes workout and updates stats', async () => {
    setup();
    api.completeWorkout.mockResolvedValue({});
    await act(async () => {
      await result.current.startWorkout('Жим лёжа');
    });
    expect(api.completeWorkout).toHaveBeenCalledWith(
      'default',
      'Жим лёжа',
      5,
      'Completed'
    );
    expect(update).toHaveBeenCalledWith({
      completedWorkouts: ['Жим лёжа'],
      dailyCompleted: 1,
      points: 140,
      xp: 240,
      trainingMinutes: 42,
      caloriesBurned: 420,
      trainingData: {
        day: { goal: 3, completed: 1, minutes: 42, calories: 420 },
      },
    });
    expect(notify).toHaveBeenCalledWith(
      'Жим лёжа\n✓ Тренировка завершена. +40 XP'
    );
  });

  it('startWorkout ignores already completed workout', async () => {
    setup({ completedWorkouts: ['Жим лёжа'] });
    await act(async () => {
      await result.current.startWorkout('Жим лёжа');
    });
    expect(api.completeWorkout).not.toHaveBeenCalled();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith(
      'Эта тренировка уже отмечена как выполненная'
    );
  });

  it('startWorkout keeps trainingData without day', async () => {
    setup({ trainingData: undefined });
    api.completeWorkout.mockResolvedValue({});
    await act(async () => {
      await result.current.startWorkout('Бег');
    });
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({ trainingData: undefined })
    );
  });

  it('startWorkout caps daily goal', async () => {
    setup({
      trainingData: {
        day: { goal: 2, completed: 1, minutes: 30, calories: 300 },
      },
    });
    api.completeWorkout.mockResolvedValue({});
    await act(async () => {
      await result.current.startWorkout('Бег');
    });
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({
        trainingData: {
          day: { goal: 2, completed: 2, minutes: 72, calories: 720 },
        },
      })
    );
  });

  it('startWorkout handles undefined completedWorkouts', async () => {
    setup({ completedWorkouts: undefined });
    api.completeWorkout.mockResolvedValue({});
    await act(async () => {
      await result.current.startWorkout('Бег');
    });
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({ completedWorkouts: ['Бег'] })
    );
  });

  it('startWorkout handles undefined trainingData goal', async () => {
    setup({ trainingData: { day: { completed: 0, minutes: 0, calories: 0 } } });
    api.completeWorkout.mockResolvedValue({});
    await act(async () => {
      await result.current.startWorkout('Бег');
    });
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({
        trainingData: {
          day: { completed: 1, minutes: 42, calories: 420 },
        },
      })
    );
  });

  it('startWorkout handles undefined points and xp', async () => {
    setup({ points: undefined, xp: undefined });
    api.completeWorkout.mockResolvedValue({});
    await act(async () => {
      await result.current.startWorkout('Бег');
    });
    expect(update).toHaveBeenCalledWith(
      expect.objectContaining({ points: 40, xp: 40 })
    );
  });

  it('startWorkout notifies on failure', async () => {
    setup();
    api.completeWorkout.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.startWorkout('Бег');
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Ошибка завершения тренировки');
  });

  it('loadTrainingPlans stores plans', async () => {
    setup();
    api.loadTrainingPlans.mockResolvedValue({ plans: [{ id: 1 }] });
    await act(async () => {
      await result.current.loadTrainingPlans();
    });
    expect(update).toHaveBeenCalledWith({ trainingPlans: [{ id: 1 }] });
  });

  it('loadTrainingPlans ignores invalid payload', async () => {
    setup();
    api.loadTrainingPlans.mockResolvedValue({});
    await act(async () => {
      await result.current.loadTrainingPlans();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('loadTrainingPlans swallows errors', async () => {
    setup();
    api.loadTrainingPlans.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.loadTrainingPlans();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('generatePlan reloads plans and notifies', async () => {
    setup();
    const plan = { id: 5, title: 'Силовой' };
    api.generatePlan.mockResolvedValue(plan);
    api.loadTrainingPlans.mockResolvedValue({ plans: [plan] });
    let returned;
    await act(async () => {
      returned = await result.current.generatePlan({ goal: 'mass' });
    });
    expect(api.generatePlan).toHaveBeenCalledWith({ goal: 'mass' });
    expect(update).toHaveBeenCalledWith({ trainingPlans: [plan] });
    expect(notify).toHaveBeenCalledWith('План создан');
    expect(returned).toBe(plan);
  });

  it('generatePlan returns null on empty response', async () => {
    setup();
    api.generatePlan.mockResolvedValue(null);
    let returned;
    await act(async () => {
      returned = await result.current.generatePlan();
    });
    expect(returned).toBeNull();
    expect(notify).not.toHaveBeenCalled();
  });

  it('generatePlan notifies on failure', async () => {
    setup();
    api.generatePlan.mockRejectedValue(new Error('boom'));
    let returned;
    await act(async () => {
      returned = await result.current.generatePlan({});
    });
    expect(returned).toBeNull();
    expect(notify).toHaveBeenCalledWith('Ошибка генерации плана');
  });

  it('getPlanDetails returns plan', async () => {
    setup();
    api.getPlanDetails.mockResolvedValue({ id: 9 });
    let returned;
    await act(async () => {
      returned = await result.current.getPlanDetails(9);
    });
    expect(api.getPlanDetails).toHaveBeenCalledWith(9);
    expect(returned).toEqual({ id: 9 });
  });

  it('getPlanDetails returns null on failure', async () => {
    setup();
    api.getPlanDetails.mockRejectedValue(new Error('boom'));
    let returned;
    await act(async () => {
      returned = await result.current.getPlanDetails(9);
    });
    expect(returned).toBeNull();
    expect(notify).toHaveBeenCalledWith('Ошибка загрузки плана');
  });

  it('loadProgress calls api', async () => {
    setup();
    api.loadProgress.mockResolvedValue({});
    await act(async () => {
      await result.current.loadProgress();
    });
    expect(api.loadProgress).toHaveBeenCalled();
  });

  it('loadProgress swallows errors', async () => {
    setup();
    api.loadProgress.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.loadProgress();
    });
    expect(notify).not.toHaveBeenCalled();
  });

  it('completeWorkout sends rating and feedback', async () => {
    setup();
    api.completeWorkout.mockResolvedValue({});
    await act(async () => {
      await result.current.completeWorkout(3, 'w2');
    });
    expect(api.completeWorkout).toHaveBeenCalledWith(3, 'w2', 5, '');
  });

  it('completeWorkout passes rating and feedback', async () => {
    setup();
    api.completeWorkout.mockResolvedValue({});
    await act(async () => {
      await result.current.completeWorkout(3, 'w2', 4, 'Отлично');
    });
    expect(api.completeWorkout).toHaveBeenCalledWith(3, 'w2', 4, 'Отлично');
  });

  it('completeWorkout notifies on failure', async () => {
    setup();
    api.completeWorkout.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.completeWorkout(3, 'w2');
    });
    expect(notify).toHaveBeenCalledWith('Ошибка завершения тренировки');
  });
});
