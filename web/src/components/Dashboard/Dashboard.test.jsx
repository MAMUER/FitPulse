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

vi.mock('chart.js/auto', () => {
  const MockChart = vi.fn(function Chart() {
    this.destroy = vi.fn();
  });
  return { Chart: MockChart };
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
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({}));
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

  it('displays AI analysis with predicted_class fallback', async () => {
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
      predicted_class: 'recovery',
    });
    vi.spyOn(api, 'getTrainingPlans').mockResolvedValueOnce({ plans: [] });
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('recovery')).toBeInTheDocument();
    });

    expect(screen.getByText('AI анализ требует больше данных')).toBeInTheDocument();
  });

  it('handles AI analysis error', async () => {
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
    vi.spyOn(api, 'classifyState').mockRejectedValueOnce(
      new Error('analysis failed')
    );
    vi.spyOn(api, 'getTrainingPlans').mockResolvedValueOnce({ plans: [] });
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Ошибка анализа')).toBeInTheDocument();
    });

    expect(screen.getByText('Сервис AI временно недоступен')).toBeInTheDocument();
  });

  it('handles getBiometricRecords load error', async () => {
    vi.spyOn(api, 'getBiometricRecords').mockRejectedValueOnce(
      new Error('biometric failed')
    );
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText('--').length).toBeGreaterThanOrEqual(1);
    });
  });

  it('displays rest workout when today workout has no matching day', async () => {
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
                  day_of_week: 0,
                  training_type: 'cardio',
                  exercises: [],
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

    expect(screen.getByText(/Сегодня нет тренировки/)).toBeInTheDocument();
  });

  it('handles getPlan error gracefully', async () => {
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
    vi.spyOn(api, 'getPlan').mockRejectedValueOnce(new Error('plan failed'));
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Норма')).toBeInTheDocument();
    });

    expect(screen.getByText(/Сегодня нет тренировки/)).toBeInTheDocument();
  });

  it('handles getTrainingPlans error gracefully', async () => {
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
    vi.spyOn(api, 'getTrainingPlans').mockRejectedValueOnce(
      new Error('plans failed')
    );
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText('--').length).toBeGreaterThanOrEqual(1);
    });
  });

  it('displays sleep value correctly', async () => {
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

    expect(screen.getByText('7.5')).toBeInTheDocument();
  });

  it('renders heart rate chart when multiple records available', async () => {
    vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
      records: [
        { value: 70, timestamp: '2024-01-01T00:00:00Z' },
        { value: 75, timestamp: '2024-01-01T00:05:00Z' },
        { value: 80, timestamp: '2024-01-01T00:10:00Z' },
      ],
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

    expect(screen.getByText('Динамика пульса')).toBeInTheDocument();
  });

  it('handles dashboard load error from ai recommendation', async () => {
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
    vi.spyOn(api, 'classifyState').mockRejectedValueOnce(
      new Error('analysis failed')
    );
    vi.spyOn(api, 'getTrainingPlans').mockResolvedValueOnce({ plans: [] });
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText('Ошибка анализа')).toBeInTheDocument();
    });

    expect(screen.getByText('Сервис AI временно недоступен')).toBeInTheDocument();
  });
});
