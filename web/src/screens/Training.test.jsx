import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Training from './Training';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

const videos = [
  {
    id: 'v1',
    title: 'Утренняя зарядка',
    description: 'Лёгкий комплекс для старта дня.',
    image: '/video1.jpg',
    duration: '12 мин',
    level: 'Лёгкий',
    equipment: ['Коврик'],
  },
  {
    id: 'v2',
    title: 'Силовая тренировка',
    description: 'Базовые упражнения с гантелями.',
    image: '/video2.jpg',
    duration: '25 мин',
    level: 'Средний',
    equipment: ['Гантели', 'Коврик'],
  },
];

const trainingPlans = [
  { id: 1, title: 'План на месяц', duration_weeks: 4 },
  { id: 2, name: 'План на неделю', durationWeeks: 1 },
];

function makeContext(overrides = {}) {
  const { state: stateOverrides, ...rest } = overrides;
  return {
    state: {
      language: 'ru',
      trainingPeriod: 'day',
      trainingData: {
        day: {
          completed: 1,
          goal: 3,
          calories: 620,
          minutes: 55,
          workouts: ['Силовая база', 'Мобильность'],
        },
        week: {
          completed: 2,
          goal: 5,
          calories: 1200,
          minutes: 120,
          workouts: ['Кардио', 'Силовая'],
        },
        month: {
          completed: 8,
          goal: 16,
          calories: 5200,
          minutes: 520,
          workouts: ['Кардио', 'Силовая', 'Плавание'],
        },
      },
      completedWorkouts: [],
      selectedWorkout: null,
      trainingVideos: videos,
      trainingPlans,
      instructions: {
        gym: 'Протирай тренажёры после использования.',
        pool: 'Принимай душ до и после бассейна.',
        hygiene: 'Используй личное полотенце.',
      },
      customExercises: [{ name: 'Отжимания', sets: '3 × 15' }],
      ...stateOverrides,
    },
    t: (ru, _en) => ru,
    formatRelativeDate: vi.fn((offset) => `date-${offset}`),
    startWorkout: vi.fn(),
    toggleWorkout: vi.fn(),
    setTrainingPeriod: vi.fn(),
    loadTrainingPlans: vi.fn(),
    generatePlan: vi.fn(),
    addCustomExercise: vi.fn(),
    removeCustomExercise: vi.fn(),
    quickAI: vi.fn(),
    notify: vi.fn(),
    ...rest,
  };
}

describe('Training', () => {
  it('renders the training screen and loads plans', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);

    expect(context.loadTrainingPlans).toHaveBeenCalled();
    expect(screen.getByText('План тренировок')).toBeInTheDocument();
    expect(screen.getByText('1/3')).toBeInTheDocument();
    expect(screen.getByText('Силовая база')).toBeInTheDocument();
    expect(screen.getByText('Мобильность и восстановление')).toBeInTheDocument();
    expect(screen.getByText('Кардио в бассейне')).toBeInTheDocument();
    expect(screen.getByText('Правила в зале')).toBeInTheDocument();
  });

  it('switches the training period', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);
    fireEvent.click(screen.getByRole('button', { name: 'Неделя' }));

    expect(context.setTrainingPeriod).toHaveBeenCalledWith('week');
  });

  it('toggles workout details', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);
    fireEvent.click(screen.getAllByText('Упражнения')[0]);

    expect(context.toggleWorkout).toHaveBeenCalledWith(1);
  });

  it('starts a workout', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);
    fireEvent.click(screen.getAllByText('Начать')[0]);

    expect(context.startWorkout).toHaveBeenCalledWith('Силовая база');
  });

  it('shows completed workouts as disabled', () => {
    const context = makeContext({
      state: { completedWorkouts: ['Силовая база'] },
    });
    useApp.mockReturnValue(context);

    render(<Training />);

    const completed = screen.getByText('Выполнено');
    expect(completed.closest('button')).toBeDisabled();
  });

  it('marks a workout as completed for the selected period', () => {
    const context = makeContext({ state: { trainingPeriod: 'week' } });
    useApp.mockReturnValue(context);

    render(<Training />);

    expect(screen.getByText('2/5')).toBeInTheDocument();
    expect(screen.getByText('Кардио')).toBeInTheDocument();
  });

  it('adds a custom exercise', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);
    fireEvent.change(screen.getByLabelText('Название упражнения'), {
      target: { value: 'Приседания' },
    });
    fireEvent.change(screen.getByLabelText('Подходы × Повторения'), {
      target: { value: '4 × 12' },
    });
    fireEvent.click(screen.getByText('Добавить'));

    expect(context.addCustomExercise).toHaveBeenCalledWith({
      name: 'Приседания',
      sets: '4 × 12',
    });
  });

  it('renders and removes custom exercises', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);

    expect(screen.getByText('Отжимания')).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('Удалить упражнение'));

    expect(context.removeCustomExercise).toHaveBeenCalledWith(0);
  });

  it('opens a video and switches to the next one', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);
    fireEvent.click(screen.getByText('Утренняя зарядка'));

    let dialog = screen.getByRole('dialog');
    expect(within(dialog).getByRole('heading', { name: 'Утренняя зарядка' })).toBeInTheDocument();
    fireEvent.click(within(dialog).getByLabelText('Следующее видео'));

    dialog = screen.getByRole('dialog');
    expect(
      within(dialog).getByRole('heading', { name: 'Силовая тренировка' })
    ).toBeInTheDocument();
  });

  it('closes the video modal', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);
    fireEvent.click(screen.getByText('Утренняя зарядка'));
    fireEvent.click(screen.getByLabelText('Закрыть'));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('renders FitPulse plans and generates a new one', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);

    expect(screen.getByText('Планы FitPulse')).toBeInTheDocument();
    expect(screen.getByText('План на месяц')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Сгенерировать'));

    expect(context.generatePlan).toHaveBeenCalledWith({ durationWeeks: 4 });
  });

  it('asks AI for a home exercise', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);
    fireEvent.click(screen.getByText('Спросить AI'));

    expect(context.quickAI).toHaveBeenCalledWith(
      'Предложи упражнение для дома'
    );
  });

  it('renders training instructions', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);

    expect(screen.getByText('Правила в бассейне')).toBeInTheDocument();
    expect(screen.getByText('Гигиена')).toBeInTheDocument();
    expect(
      screen.getByText('Протирай тренажёры после использования.')
    ).toBeInTheDocument();
  });

  it('renders without videos and plans', () => {
    const context = makeContext({
      state: { trainingVideos: null, trainingPlans: null },
    });
    useApp.mockReturnValue(context);

    render(<Training />);

    expect(screen.queryByText('Видео-уроки')).toBeInTheDocument();
    expect(screen.queryByText('Планы FitPulse')).not.toBeInTheDocument();
  });
});
