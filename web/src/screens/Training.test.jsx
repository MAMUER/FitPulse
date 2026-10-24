import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Training from './Training';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const makePlan = (overrides = {}) => ({
  plan_id: 'plan-1',
  plan_data: { name: 'Базовый план', training_goal: 'Выносливость' },
  training_goal: 'Выносливость',
  ...overrides,
});

const baseContext = (overrides = {}) => ({
  state: {
    screen: 'training',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    trainingPlans: [],
    ...overrides.state,
  },
  t: (ru, _en) => ru,
  startWorkout: vi.fn(),
  go: vi.fn(),
  loadTrainingPlans: vi.fn(),
  generatePlan: vi.fn().mockResolvedValue({}),
  getPlanDetails: vi.fn(),
  ...overrides,
});

function renderTraining(context) {
  useApp.mockReturnValue(context);
  return render(
    <AppProvider>
      <Training />
    </AppProvider>
  );
}

describe('Training', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders training screen', () => {
    renderTraining(baseContext());
    expect(screen.getByText('Тренировки')).toBeInTheDocument();
  });

  it('loads training plans on mount', () => {
    const loadTrainingPlans = vi.fn();
    renderTraining(baseContext({ loadTrainingPlans }));
    expect(loadTrainingPlans).toHaveBeenCalledTimes(1);
  });

  it('renders plans tab with generate button and empty state', () => {
    renderTraining(baseContext());
    expect(screen.getByText('Планы')).toBeInTheDocument();
    expect(screen.getByText('Создать план')).toBeInTheDocument();
    expect(screen.getByText('Нет планов')).toBeInTheDocument();
  });

  it('renders timer tab label', () => {
    renderTraining(baseContext());
    expect(screen.getByText('Таймер')).toBeInTheDocument();
  });

  it('switches to start tab and shows workout controls', async () => {
    const user = userEvent.setup();
    renderTraining(baseContext());
    await user.click(screen.getByText('Таймер'));
    expect(screen.getByText('Начать тренировку')).toBeInTheDocument();
    expect(screen.getByText('На главную')).toBeInTheDocument();
    expect(screen.queryByText('Создать план')).not.toBeInTheDocument();
  });

  it('starts workout from start tab', async () => {
    const user = userEvent.setup();
    const startWorkout = vi.fn();
    renderTraining(baseContext({ startWorkout }));
    await user.click(screen.getByText('Таймер'));
    await user.click(screen.getByText('Начать тренировку'));
    expect(startWorkout).toHaveBeenCalledWith('Тренировка');
  });

  it('navigates home from start tab', async () => {
    const user = userEvent.setup();
    const go = vi.fn();
    renderTraining(baseContext({ go }));
    await user.click(screen.getByText('Таймер'));
    await user.click(screen.getByText('На главную'));
    expect(go).toHaveBeenCalledWith('home');
  });

  it('renders training plans list with details buttons', () => {
    const plans = [
      makePlan({ plan_id: 'p1' }),
      makePlan({ plan_id: 'p2', plan_data: { name: 'Интенсив', training_goal: 'Сила' } }),
    ];
    renderTraining(baseContext({ state: { trainingPlans: plans } }));
    expect(screen.getByText('Базовый план')).toBeInTheDocument();
    expect(screen.getByText('Интенсив')).toBeInTheDocument();
    const detailsButtons = screen.getAllByText('Подробнее');
    expect(detailsButtons).toHaveLength(2);
  });

  it('uses plan_id as title when plan_data name is missing', () => {
    const plans = [makePlan({ plan_id: 'abc-123', plan_data: null })];
    renderTraining(baseContext({ state: { trainingPlans: plans } }));
    expect(screen.getByText('abc-123')).toBeInTheDocument();
  });

  it('uses training_goal as subtitle', () => {
    const plans = [makePlan({ plan_id: 'p1', training_goal: 'Кардио' })];
    renderTraining(baseContext({ state: { trainingPlans: plans } }));
    expect(screen.getByText('Кардио')).toBeInTheDocument();
  });

  it('falls back to plan_data.training_goal for subtitle', () => {
    const plans = [
      makePlan({ plan_id: 'p1', training_goal: null, plan_data: { training_goal: 'Гибкость' } }),
    ];
    renderTraining(baseContext({ state: { trainingPlans: plans } }));
    expect(screen.getByText('Гибкость')).toBeInTheDocument();
  });

  it('opens plan details when details button is clicked', async () => {
    const user = userEvent.setup();
    const getPlanDetails = vi.fn();
    const plans = [makePlan({ plan_id: 'p1' })];
    renderTraining(
      baseContext({ state: { trainingPlans: plans }, getPlanDetails })
    );
    const detailsButtons = screen.getAllByText('Подробнее');
    await user.click(detailsButtons[0]);
    expect(getPlanDetails).toHaveBeenCalledWith('p1');
  });

  it('generates plan and returns to plans tab', async () => {
    const user = userEvent.setup();
    let resolveGenerate;
    const generatePlan = vi.fn(
      () =>
        new Promise((resolve) => {
          resolveGenerate = resolve;
        })
    );
    renderTraining(baseContext({ generatePlan }));
    await user.click(screen.getByText('Создать план'));
    expect(screen.getByText('Генерация...')).toBeInTheDocument();
    expect(generatePlan).toHaveBeenCalledWith({
      durationWeeks: 4,
      availableDays: [1, 3, 5],
      class: 'endurance_basic',
      confidence: 0.8,
    });
    resolveGenerate({});
    await waitFor(() => {
      expect(screen.queryByText('Генерация...')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Создать план')).toBeInTheDocument();
  });

  it('uses English translations when language is en', () => {
    const enContext = baseContext({
      state: { language: 'en' },
      t: (_ru, en) => en,
    });
    renderTraining(enContext);
    expect(screen.getByText('Workouts')).toBeInTheDocument();
    expect(screen.getByText('Plans')).toBeInTheDocument();
    expect(screen.getByText('Timer')).toBeInTheDocument();
    expect(screen.getByText('Generate plan')).toBeInTheDocument();
    expect(screen.getByText('No plans')).toBeInTheDocument();
  });

  it('renders empty state in English', () => {
    const enContext = baseContext({
      state: { language: 'en' },
      t: (_ru, en) => en,
    });
    renderTraining(enContext);
    expect(screen.getByText('No plans')).toBeInTheDocument();
  });

  it('renders details button in English', () => {
    const plans = [makePlan({ plan_id: 'p1' })];
    const enContext = baseContext({
      state: { language: 'en', trainingPlans: plans },
      t: (_ru, en) => en,
    });
    renderTraining(enContext);
    expect(screen.getByText('Details')).toBeInTheDocument();
  });

  it('renders start tab controls in English', async () => {
    const user = userEvent.setup();
    const enContext = baseContext({
      state: { language: 'en' },
      t: (_ru, en) => en,
    });
    renderTraining(enContext);
    await user.click(screen.getByText('Timer'));
    expect(screen.getByText('Start workout')).toBeInTheDocument();
    expect(screen.getByText('Go home')).toBeInTheDocument();
  });
});