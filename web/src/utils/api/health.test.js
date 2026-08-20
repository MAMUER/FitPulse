import { describe, expect, it, vi } from 'vitest';
import * as api from './health';

describe('health api', () => {
  const mockResponse = (data) => ({
    ok: true,
    status: 200,
    headers: new Headers({ 'content-type': 'application/json' }),
    json: () => Promise.resolve(data),
  });

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('adds biometric record', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ id: '1' }));

    const result = await api.addBiometricRecord(
      'heart_rate',
      75,
      '2024-01-01',
      'manual'
    );
    expect(result).toEqual({ id: '1' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/biometrics', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        metric_type: 'heart_rate',
        value: 75,
        timestamp: '2024-01-01',
        device_type: 'manual',
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('gets biometric records with filters', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ records: [] })
    );

    const result = await api.getBiometricRecords(
      'heart_rate',
      '2024-01-01',
      '2024-01-02',
      10
    );
    expect(result).toEqual({ records: [] });
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/biometrics?metric_type=heart_rate&limit=10&from=2024-01-01&to=2024-01-02',
      {
        headers: { 'Content-Type': 'application/json' },
        signal: expect.any(AbortSignal),
      }
    );
  });

  it('gets biometric records with only from', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ records: [] })
    );

    const result = await api.getBiometricRecords('heart_rate', '2024-01-01');
    expect(result).toEqual({ records: [] });
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/biometrics?metric_type=heart_rate&limit=100&from=2024-01-01',
      {
        headers: { 'Content-Type': 'application/json' },
        signal: expect.any(AbortSignal),
      }
    );
  });

  it('gets biometric records with only to', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ records: [] })
    );

    const result = await api.getBiometricRecords(
      'heart_rate',
      null,
      '2024-01-02'
    );
    expect(result).toEqual({ records: [] });
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/biometrics?metric_type=heart_rate&limit=100&to=2024-01-02',
      {
        headers: { 'Content-Type': 'application/json' },
        signal: expect.any(AbortSignal),
      }
    );
  });

  it('lists health conditions', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ conditions: [] })
    );

    const result = await api.listHealthConditions('allergy');
    expect(result).toEqual({ conditions: [] });
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/health/conditions?condition_type=allergy',
      {
        headers: { 'Content-Type': 'application/json' },
        signal: expect.any(AbortSignal),
      }
    );
  });

  it('upserts health condition', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ id: '1' }));

    const result = await api.upsertHealthCondition({
      condition_type: 'allergy',
      condition_name: 'Peanuts',
    });
    expect(result).toEqual({ id: '1' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/health/conditions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        condition_type: 'allergy',
        condition_name: 'Peanuts',
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('deletes health condition', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.deleteHealthCondition('1');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/health/conditions/1', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      signal: expect.any(AbortSignal),
    });
  });

  it('lists body composition', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ records: [] })
    );

    const result = await api.listBodyComposition('2024-01-01', '2024-01-02');
    expect(result).toEqual({ records: [] });
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/health/body-composition?limit=100&from=2024-01-01&to=2024-01-02',
      {
        headers: { 'Content-Type': 'application/json' },
        signal: expect.any(AbortSignal),
      }
    );
  });

  it('creates body composition', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ id: '1' }));

    const result = await api.createBodyComposition({
      weight_kg: 70,
      height_cm: 175,
    });
    expect(result).toEqual({ id: '1' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/health/body-composition', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        weight_kg: 70,
        height_cm: 175,
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('lists menstrual cycles', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ cycles: [] })
    );

    const result = await api.listMenstrualCycles();
    expect(result).toEqual({ cycles: [] });
    expect(fetch).toHaveBeenCalledWith('/api/v1/health/menstrual-cycles', {
      headers: { 'Content-Type': 'application/json' },
      signal: expect.any(AbortSignal),
    });
  });

  it('creates menstrual cycle', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ id: '1' }));

    const result = await api.createMenstrualCycle({
      cycle_start_date: '2024-01-01',
    });
    expect(result).toEqual({ id: '1' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/health/menstrual-cycles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        cycle_start_date: '2024-01-01',
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('updates menstrual cycle', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse({ id: '1' }));

    const result = await api.updateMenstrualCycle('1', {
      cycle_start_date: '2024-01-01',
    });
    expect(result).toEqual({ id: '1' });
    expect(fetch).toHaveBeenCalledWith('/api/v1/health/menstrual-cycles/1', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        cycle_start_date: '2024-01-01',
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('deletes menstrual cycle', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({ success: true })
    );

    const result = await api.deleteMenstrualCycle('1');
    expect(result).toEqual({ success: true });
    expect(fetch).toHaveBeenCalledWith('/api/v1/health/menstrual-cycles/1', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      signal: expect.any(AbortSignal),
    });
  });
});
