import { describe, expect, it, vi } from 'vitest';
import * as api from './ml';

describe('ml api', () => {
  const mockResponse = (data) => ({
    ok: true,
    status: 200,
    headers: new Headers({ 'content-type': 'application/json' }),
    json: () => Promise.resolve(data),
  });

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('classifies state', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ state: 'normal' })
    );

    const biometrics = { heart_rate: 75, spo2: 98 };
    const result = await api.classifyState(biometrics);
    expect(result).toEqual({ state: 'normal' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/ml/classify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(biometrics),
    });
  });

  it('generates ml plan', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ plan: {} }));

    const result = await api.generateMLPlan(
      'strength',
      { age: 25 },
      'muscle_gain',
      {
        time: '30min',
      }
    );
    expect(result).toEqual({ plan: {} });
    expect(fetch).toHaveBeenCalledWith('/api/v1/ml/generate-plan', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        training_class: 'strength',
        user_profile: { age: 25 },
        goal: 'muscle_gain',
        constraints: { time: '30min' },
      }),
    });
  });

  it('handles api error', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: false,
      status: 500,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: () => Promise.resolve({ message: 'Server error' }),
    });

    await expect(api.classifyState({})).rejects.toThrow('Server error');
  });
});
