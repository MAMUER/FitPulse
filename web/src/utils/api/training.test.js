import { describe, expect, it, vi } from 'vitest';
import * as api from './training';

describe('training api', () => {
  const mockResponse = (data) => ({
    ok: true,
    status: 200,
    headers: new Headers({ 'content-type': 'application/json' }),
    json: () => Promise.resolve(data),
  });

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('generates training plan', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ id: '1' }));

    const result = await api.generateTrainingPlan(
      4,
      [1, 3, 5],
      'strength',
      0.9
    );
    expect(result).toEqual({ id: '1' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/training/generate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        duration_weeks: 4,
        available_days: [1, 3, 5],
        class: 'strength',
        confidence: 0.9,
      }),
    });
  });

  it('gets training plans', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ plans: [] })
    );

    const result = await api.getTrainingPlans(1, 10);
    expect(result).toEqual({ plans: [] });
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/training/plans?page=1&page_size=10',
      {
        headers: { 'Content-Type': 'application/json' },
      }
    );
  });

  it('gets plan by id', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ plan: {} }));

    const result = await api.getPlan('1');
    expect(result).toEqual({ plan: {} });
    expect(fetch).toHaveBeenCalledWith('/api/v1/training/plans/1', {
      headers: { 'Content-Type': 'application/json' },
    });
  });

  it('completes workout', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.completeWorkout('1', '2', 5, 'Great workout');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/training/complete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        plan_id: '1',
        workout_id: '2',
        rating: 5,
        feedback: 'Great workout',
      }),
    });
  });

  it('gets progress', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ progress: [] })
    );

    const result = await api.getProgress();
    expect(result).toEqual({ progress: [] });
    expect(fetch).toHaveBeenCalledWith('/api/v1/training/progress', {
      headers: { 'Content-Type': 'application/json' },
    });
  });

  it('gets achievements', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ achievements: [] })
    );

    const result = await api.getAchievements();
    expect(result).toEqual({ achievements: [] });
    expect(fetch).toHaveBeenCalledWith('/api/v1/achievements', {
      headers: { 'Content-Type': 'application/json' },
    });
  });
});
