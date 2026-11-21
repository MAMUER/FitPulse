import { fireEvent, render, screen } from '@testing-library/react';
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
  {
    id: 1,
    title: ['Силовая база', 'Strength base'],
    level: ['Средний уровень', 'Medium level'],
    duration: '42 мин',
    calories: 420,
    time: '18:30',
    offset: 0,
    exercises: [
      ['Приседания', '3×12'],
      ['Жим лёжа', '3×10'],
      ['Тяга', '3×12'],
      ['Планка', '3×45 с'],
      ['Гиперэкстензия', '3×15'],
    ],
  },
  {
    id: 2,
    title: ['Мобильность и восстановление', 'Mobility and recovery'],
    level: ['Лёгкий уровень', 'Light level'],
    duration: '20 мин',
    calories: 160,
    time: '21:30',
    offset: 1,
    exercises: [
      ['Растяжка', '5 мин'],
      ['Мяч', '5 мин'],
      ['Дыхание', '5 мин'],
      ['Поза', '5 мин'],
    ],
  },
  {
    id: 3,
    title: ['Кардио в бассейне', 'Pool cardio'],
    level: ['Средний уровень', 'Medium level'],
    duration: '35 мин',
    calories: 510,
    time: '10:00',
    offset: 3,
    exercises: [
      ['Плавание', '10 мин'],
      ['Аквааэробика', '15 мин'],
      ['Отдых', '10 мин'],
    ],
  },
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
      ...stateOverrides,
    },
    // useTraining exposes customExercises at the top level of the context
    // value, not inside state — Training.jsx reads it from useApp().
    customExercises: [{ name: 'Становая тяга', sets: '3 × 5' }],
    t: (ru, _en) => ru,
    canEdit: true,
    formatRelativeDate: vi.fn((offset) => `date-${offset}`),
    startWorkout: vi.fn(),
    toggleWorkout: vi.fn(),
    setTrainingPeriod: vi.fn(),
    loadTrainingPlans: vi.fn(),
    generatePlan: vi.fn(),
    addCustomExercise: vi.fn(),
    removeCustomExercise: vi.fn(),
    quickAI: vi.fn(),
    go: vi.fn(),
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
    // "Силовая база" is rendered both in the plan summary and in the workout card.
    expect(screen.getAllByText('Силовая база').length).toBeGreaterThan(0);
    expect(
      screen.getAllByText('Мобильность и восстановление').length
    ).toBeGreaterThan(0);
    expect(screen.getAllByText('Кардио в бассейне').length).toBeGreaterThan(0);
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

  it('starts a workout session and finishes it', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);
    fireEvent.click(screen.getAllByText('Начать')[0]);

    // Открывается сессия с таймером и кнопкой завершения.
    const finish = screen.getByText('Завершить тренировку');
    expect(finish).toBeInTheDocument();
    fireEvent.click(finish);

    expect(context.startWorkout).toHaveBeenCalledWith('Силовая база');
  });

  it('shows completed workouts as disabled', () => {
    const context = makeContext({
      state: { completedWorkouts: ['Силовая база'] },
    });
    useApp.mockReturnValue(context);

    render(<Training />);

    // "Выполнено" appears both as a status label and as the disabled button.
    const completed = screen.getByRole('button', { name: 'Выполнено' });
    expect(completed).toBeDisabled();
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

    expect(screen.getByText('Становая тяга')).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('Удалить упражнение'));

    expect(context.removeCustomExercise).toHaveBeenCalledWith(0);
  });

  it('renders FitPulse plans and generates a new one', () => {
    const context = makeContext();
    useApp.mockReturnValue(context);

    render(<Training />);

    expect(screen.getByText('Планы FitPulse')).toBeInTheDocument();
    // План виден и в карточке «Мой план», и в списке «Планы FitPulse».
    expect(screen.getAllByText('Силовая база').length).toBeGreaterThan(1);
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

    // Без видео раздел «Видео-уроки» не показываем.
    expect(screen.queryByText('Видео-уроки')).not.toBeInTheDocument();
    expect(screen.queryByText('Планы FitPulse')).not.toBeInTheDocument();
  });
});
