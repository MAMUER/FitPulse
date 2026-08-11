import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import Dashboard from './Dashboard';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const renderDashboard = () => {
  useAuth.mockReturnValue({
    token: 'test-token',
  });

  return render(
    <AuthProvider>
      <Dashboard />
    </AuthProvider>
  );
};

describe('Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading state initially', () => {
    vi.spyOn(api, 'getBiometricRecords').mockImplementation(
      () => new Promise(() => {})
    );
    renderDashboard();

    expect(screen.getByText('Загрузка рекомендаций...')).toBeInTheDocument();
  });

  it('displays biometric metrics', async () => {
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 75 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 98 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 7.5 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 120 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 80 }],
    });
    vi.spyOn(api, 'classifyState').mockResolvedValueOnce({
      predicted_class_ru: 'Норма',
      description: 'Все показатели в норме',
    });
    vi.spyOn(api, 'getTrainingPlans').mockResolvedValueOnce({ plans: [] });
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Норма')).toBeInTheDocument();
    });
  });

  it('displays AI recommendation', async () => {
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 75 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 98 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 7.5 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 120 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 80 }],
    });
    vi.spyOn(api, 'classifyState').mockResolvedValueOnce({
      predicted_class_ru: 'Отличная форма',
      description: 'Продолжайте в том же духе',
    });
    vi.spyOn(api, 'getTrainingPlans').mockResolvedValueOnce({ plans: [] });
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Отличная форма')).toBeInTheDocument();
    });

    expect(screen.getByText('Продолжайте в том же духе')).toBeInTheDocument();
  });

  it('displays rest workout when no plans', async () => {
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 75 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 98 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 7.5 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 120 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 80 }],
    });
    vi.spyOn(api, 'classifyState').mockResolvedValueOnce({
      predicted_class_ru: 'Норма',
      description: 'Все показатели в норме',
    });
    vi.spyOn(api, 'getTrainingPlans').mockResolvedValueOnce({ plans: [] });
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Норма')).toBeInTheDocument();
    });

    expect(screen.getByText(/Сегодня нет тренировки/)).toBeInTheDocument();
  });

  it('handles load error gracefully', async () => {
    vi.spyOn(api, 'getBiometricRecords').mockRejectedValueOnce(
      new Error('load failed')
    );
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText('--').length).toBeGreaterThanOrEqual(1);
    });
  });

  it('displays blood pressure when available', async () => {
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 75 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 98 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 7.5 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 120 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 80 }],
    });
    vi.spyOn(api, 'classifyState').mockResolvedValueOnce({
      predicted_class_ru: 'Норма',
      description: 'Все показатели в норме',
    });
    vi.spyOn(api, 'getTrainingPlans').mockResolvedValueOnce({ plans: [] });
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Норма')).toBeInTheDocument();
    });

    expect(screen.getByText('120/80')).toBeInTheDocument();
  });

  it('displays training plan when available', async () => {
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 75 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 98 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 7.5 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 120 }],
    });
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [{ value: 80 }],
    });
    vi.spyOn(api, 'classifyState').mockResolvedValueOnce({
      predicted_class_ru: 'Норма',
      description: 'Все показатели в норме',
    });
    vi.spyOn(api, 'getTrainingPlans').mockResolvedValueOnce({
      plans: [{ plan_id: '1' }],
    });
    vi.spyOn(api, 'getPlan').mockResolvedValueOnce({
      plan: {
        plan_data: {
          weeks: [
            {
              days: [
                {
                  day_of_week: new Date().getDay(),
                  training_type: 'cardio',
                  exercises: [
                    {
                      exercise_name: 'running',
                      sets: 3,
                      reps: 10,
                    },
                  ],
                  duration: 30,
                },
              ],
            },
          ],
        },
      },
    });
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Норма')).toBeInTheDocument();
    });

    expect(screen.getByText(/Кардио/)).toBeInTheDocument();
  });
});
