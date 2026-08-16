import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import ML from './ML';

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

const renderML = (authOverrides = {}) => {
  useAuth.mockReturnValue({
    token: 'test-token',
    ...authOverrides,
  });

  return render(
    <AuthProvider>
      <ML /> {/* NOSONAR S6770 */}
    </AuthProvider>
  );
};

describe('ML', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders classification section', () => {
    renderML();

    expect(screen.getByText('Классификация состояния')).toBeInTheDocument();
    expect(screen.getByText('Анализировать')).toBeInTheDocument();
  });

  it('renders plan generation form', () => {
    renderML();

    expect(screen.getByText('Генерация плана')).toBeInTheDocument();
    expect(screen.getByLabelText('Тип тренировки')).toBeInTheDocument();
    expect(screen.getByLabelText('Длительность (недель)')).toBeInTheDocument();
  });

  it('classifies state successfully', async () => {
    vi.spyOn(api, 'classifyState').mockResolvedValueOnce({
      predicted_class_ru: 'Норма',
      confidence: 0.95,
      description: 'Все показатели в норме',
    });
    renderML();

    await userEvent.click(screen.getByText('Анализировать'));

    await waitFor(() => {
      expect(screen.getByText('Норма')).toBeInTheDocument();
    });

    expect(
      screen.getByText((content) => content.includes('95%'))
    ).toBeInTheDocument();
    expect(screen.getByText('Все показатели в норме')).toBeInTheDocument();
  });

  it('classifies state with predicted_class fallback', async () => {
    vi.spyOn(api, 'classifyState').mockResolvedValueOnce({
      predicted_class: 'recovery',
      confidence: 0.8,
    });
    renderML();

    await userEvent.click(screen.getByText('Анализировать'));

    await waitFor(() => {
      expect(screen.getByText('recovery')).toBeInTheDocument();
    });
  });

  it('handles classification error', async () => {
    vi.spyOn(api, 'classifyState').mockRejectedValueOnce(
      new Error('analysis failed')
    );
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderML();

    await userEvent.click(screen.getByText('Анализировать'));

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith(
        'Ошибка анализа: analysis failed'
      );
    });
  });

  it('generates plan successfully', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
      plan_data: {
        weeks: [
          {
            week_number: 1,
            days: [
              {
                day_of_week: 1,
                training_type: 'cardio',
                exercises: [
                  {
                    sort_order: 0,
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
    });
    vi.spyOn(api, 'getPlan').mockResolvedValueOnce({});
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Сгенерированный план')).toBeInTheDocument();
    });

    expect(screen.getByText('Неделя 1')).toBeInTheDocument();
    expect(screen.getByText('cardio')).toBeInTheDocument();
  });

  it('handles plan generation error', async () => {
    vi.spyOn(api, 'generateMLPlan').mockRejectedValueOnce(
      new Error('generation failed')
    );
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith(
        'Ошибка генерации: generation failed'
      );
    });
  });

  it('renders empty state when plan has no weeks', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
      plan_data: { weeks: [] },
    });
    vi.spyOn(api, 'getPlan').mockResolvedValueOnce({});
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('План пуст')).toBeInTheDocument();
    });
  });

  it('renders empty state when plan data has no weeks key', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
      plan_data: {},
    });
    vi.spyOn(api, 'getPlan').mockResolvedValueOnce({});
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('План пуст')).toBeInTheDocument();
    });
  });

  it('updates training class form field', async () => {
    renderML();

    const select = screen.getByLabelText('Тип тренировки');
    await userEvent.selectOptions(select, 'power_hiit');

    expect(select).toHaveValue('power_hiit');
  });

  it('updates duration weeks form field', async () => {
    renderML();

    const input = screen.getByLabelText('Длительность (недель)');
    await userEvent.clear(input);
    await userEvent.type(input, '8');

    expect(input).toHaveValue(8);
  });

  it('displays confidence percentage when available', async () => {
    vi.spyOn(api, 'classifyState').mockResolvedValueOnce({
      predicted_class_ru: 'Норма',
      confidence: 0.87,
    });
    renderML();

    await userEvent.click(screen.getByText('Анализировать'));

    await waitFor(() => {
      expect(
        screen.getByText((content) => content.includes('87%'))
      ).toBeInTheDocument();
    });
  });

  it('displays dash when no prediction or confidence is available', async () => {
    vi.spyOn(api, 'classifyState').mockResolvedValueOnce({
      confidence: 0,
    });
    renderML();

    await userEvent.click(screen.getByText('Анализировать'));

    await waitFor(() => {
      expect(screen.getByText('—')).toBeInTheDocument();
    });
  });

  it('generates plan with plan_id and fetches full plan', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
    });
    vi.spyOn(api, 'getPlan').mockResolvedValueOnce({
      plan: {
        plan_data: {
          weeks: [
            {
              week_number: 1,
              days: [
                {
                  day_of_week: 1,
                  training_type: 'cardio',
                  exercises: [
                    {
                      sort_order: 0,
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
    });
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Сгенерированный план')).toBeInTheDocument();
    });

    expect(api.getPlan).toHaveBeenCalledWith('plan-1');
  });

  it('renders empty plan state', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
      plan_data: { weeks: [] },
    });
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('План пуст')).toBeInTheDocument();
    });
  });

  it('renders fallback exercise name when missing', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
      plan_data: {
        weeks: [
          {
            week_number: 1,
            days: [
              {
                day_of_week: 1,
                training_type: 'cardio',
                exercises: [
                  {
                    sort_order: 0,
                    sets: 3,
                    reps: 10,
                  },
                ],
              },
            ],
          },
        ],
      },
    });
    vi.spyOn(api, 'getPlan').mockResolvedValueOnce({});
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Упражнение')).toBeInTheDocument();
    });
  });

  it('falls back to plan_data when plan wrapper is missing', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
      plan_data: {
        weeks: [
          {
            week_number: 1,
            days: [
              {
                day_of_week: 1,
                training_type: 'cardio',
                exercises: [
                  {
                    sort_order: 0,
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
    });
    vi.spyOn(api, 'getPlan').mockResolvedValueOnce({});
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Сгенерированный план')).toBeInTheDocument();
    });
  });

  it('renders plan detail chart when weeks are available', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
      plan_data: {
        weeks: [
          {
            week_number: 1,
            days: [
              {
                day_of_week: 1,
                duration: 30,
              },
            ],
          },
        ],
      },
    });
    vi.spyOn(api, 'getPlan').mockResolvedValueOnce({});
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Сгенерированный план')).toBeInTheDocument();
    });

    const canvas = document.getElementById('mlProgressChart');
    expect(canvas).toBeTruthy();
  });

  it('handles getPlan error in plan detail useEffect', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
      plan_data: {
        weeks: [
          {
            week_number: 1,
            days: [
              {
                day_of_week: 1,
                duration: 30,
              },
            ],
          },
        ],
      },
    });
    vi.spyOn(api, 'getPlan').mockRejectedValueOnce(
      new Error('plan detail failed')
    );
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Сгенерированный план')).toBeInTheDocument();
    });
  });

  it('toggles training day selection', async () => {
    renderML();

    const daysGrid = document.getElementById('training-days');
    const wednesdayLabel = daysGrid.querySelectorAll('label')[2];
    const wednesdayCheckbox = wednesdayLabel.querySelector(
      'input[type="checkbox"]'
    );
    await userEvent.click(wednesdayCheckbox);

    expect(wednesdayLabel.classList.contains('selected')).toBe(true);
  });

  it('deselects training day when already selected', async () => {
    renderML();

    const daysGrid = document.getElementById('training-days');
    const mondayLabel = daysGrid.querySelectorAll('label')[1];
    const mondayCheckbox = mondayLabel.querySelector('input[type="checkbox"]');
    await userEvent.click(mondayCheckbox);

    expect(mondayLabel.classList.contains('selected')).toBe(false);
  });

  it('renders day labels for all days of week', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
      plan_data: {
        weeks: [
          {
            week_number: 1,
            days: [
              { day_of_week: 0, training_type: 'rest', exercises: [] },
              { day_of_week: 2, training_type: 'cardio', exercises: [] },
              { day_of_week: 4, training_type: 'strength', exercises: [] },
              { day_of_week: 6, training_type: 'yoga', exercises: [] },
            ],
          },
        ],
      },
    });
    vi.spyOn(api, 'getPlan').mockResolvedValueOnce({});
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Сгенерированный план')).toBeInTheDocument();
    });

    const planSection = document.getElementById('generatedPlanDetail');
    expect(planSection.querySelectorAll('.plan-day-name')).toHaveLength(4);
    expect(planSection.textContent).toContain('Вс');
    expect(planSection.textContent).toContain('Вт');
    expect(planSection.textContent).toContain('Чт');
    expect(planSection.textContent).toContain('Сб');
  });

  it('renders plan content from original plan when getPlan returns different data', async () => {
    vi.spyOn(api, 'generateMLPlan').mockResolvedValueOnce({
      plan_id: 'plan-1',
      plan_data: {
        weeks: [
          {
            week_number: 1,
            days: [
              {
                day_of_week: 1,
                training_type: 'original_type',
                exercises: [
                  {
                    sort_order: 0,
                    exercise_name: 'original_exercise',
                    sets: 3,
                    reps: 10,
                  },
                ],
              },
            ],
          },
        ],
      },
    });
    vi.spyOn(api, 'getPlan').mockResolvedValueOnce({
      plan: {
        plan_data: {
          weeks: [
            {
              week_number: 1,
              days: [
                {
                  day_of_week: 1,
                  training_type: 'fetched_type',
                  exercises: [
                    {
                      sort_order: 0,
                      exercise_name: 'fetched_exercise',
                      sets: 5,
                      reps: 15,
                    },
                  ],
                },
              ],
            },
          ],
        },
      },
    });
    renderML();

    await userEvent.click(screen.getByText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Сгенерированный план')).toBeInTheDocument();
    });

    expect(screen.getByText('original_type')).toBeInTheDocument();
    expect(screen.getByText('original_exercise')).toBeInTheDocument();
    expect(screen.queryByText('fetched_type')).not.toBeInTheDocument();
  });
});
