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

  it.each([
    {
      name: 'handles load error gracefully',
      getBiometricRecords: [{ reject: new Error('load failed') }],
      getTrainingPlans: undefined,
      classifyState: undefined,
    },
    {
      name: 'handles getBiometricRecords load error',
      getBiometricRecords: [{ reject: new Error('biometric failed') }],
      getTrainingPlans: undefined,
      classifyState: undefined,
    },
    {
      name: 'handles getTrainingPlans error gracefully',
      getBiometricRecords: [
        { value: 75 },
        { value: 98 },
        { value: 7.5 },
        { value: 120 },
        { value: 80 },
      ],
      getTrainingPlans: { reject: new Error('plans failed') },
      classifyState: {
        predicted_class_ru: 'Норма',
        description: 'Все показатели в норме',
      },
    },
  ])(
    '$name',
    async ({ getBiometricRecords, getTrainingPlans, classifyState }) => {
      (getBiometricRecords || []).forEach((mock) => {
        if (mock.reject) {
          vi.spyOn(api, 'getBiometricRecords').mockRejectedValueOnce(
            mock.reject
          );
        } else {
          vi.spyOn(api, 'getBiometricRecords').mockResolvedValueOnce({
            records: [mock],
          });
        }
      });
      if (classifyState) {
        vi.spyOn(api, 'classifyState').mockResolvedValueOnce(classifyState);
      }
      if (getTrainingPlans) {
        vi.spyOn(api, 'getTrainingPlans').mockRejectedValueOnce(
          getTrainingPlans.reject
        );
      }
      renderDashboard();

      await waitFor(() => {
        expect(screen.getAllByText('--').length).toBeGreaterThanOrEqual(1);
      });
    }
  );

  it.each([
    {
      name: 'displays blood pressure when available',
      classifyState: {
        predicted_class_ru: 'Норма',
        description: 'Все показатели в норме',
      },
      getTrainingPlans: { plans: [] },
      getPlan: undefined,
      expectedText: '120/80',
    },
    {
      name: 'displays training plan when available',
      classifyState: {
        predicted_class_ru: 'Норма',
        description: 'Все показатели в норме',
      },
      getTrainingPlans: { plans: [{ plan_id: '1' }] },
      getPlan: {
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
                        duration: 30,
                      },
                    ],
                  },
                ],
              },
            ],
          },
        },
      },
      expectedText: /Кардио/,
    },
    {
      name: 'displays AI analysis with predicted_class fallback',
      classifyState: { predicted_class: 'recovery' },
      getTrainingPlans: { plans: [] },
      getPlan: undefined,
      expectedText: 'AI анализ требует больше данных',
      classificationText: 'recovery',
    },
  ])(
    '$name',
    async ({
      classifyState,
      getTrainingPlans,
      getPlan,
      expectedText,
      classificationText,
    }) => {
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
      vi.spyOn(api, 'classifyState').mockResolvedValueOnce(classifyState);
      vi.spyOn(api, 'getTrainingPlans').mockResolvedValueOnce(getTrainingPlans);
      if (getPlan) {
        vi.spyOn(api, 'getPlan').mockResolvedValueOnce(getPlan);
      }
      renderDashboard();

      await waitFor(() => {
        expect(
          screen.getByText(classificationText || 'Норма')
        ).toBeInTheDocument();
      });

      expect(screen.getByText(expectedText)).toBeInTheDocument();
    }
  );

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

    expect(
      screen.getByText('Сервис AI временно недоступен')
    ).toBeInTheDocument();
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

    expect(
      screen.getByText('Сервис AI временно недоступен')
    ).toBeInTheDocument();
  });

  it('handles dashboard load error gracefully', async () => {
    vi.spyOn(api, 'getBiometricRecords').mockRejectedValueOnce(
      new Error('load failed')
    );
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText('--').length).toBeGreaterThanOrEqual(1);
    });
  });

  it('logs error when loadDashboard throws', async () => {
    const consoleErrorSpy = vi
      .spyOn(console, 'error')
      .mockImplementation(() => {});
    const originalAllSettled = Promise.allSettled;
    Promise.allSettled = vi.fn(() => {
      throw new Error('settled failed');
    });

    renderDashboard();

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Dashboard load failed:',
        expect.any(Error)
      );
    });

    Promise.allSettled = originalAllSettled;
    consoleErrorSpy.mockRestore();
  });
});
