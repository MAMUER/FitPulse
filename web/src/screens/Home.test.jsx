import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Home from './Home';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const story = {
  id: 1,
  title: 'Восстановление',
  description: 'План на сегодня',
  image: '/story.jpg',
};

const defaultContext = {
  state: {
    screen: 'home',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    profile: { name: 'Михаил' },
    profileImage: null,
    waterIntake: 2,
    waterGoal: 8,
    surveyCompleted: true,
    stories: [story],
    storySeen: [],
    selectedStory: null,
    calendarMonthIndex: new Date().getMonth(),
    calendarYear: new Date().getFullYear(),
    calendarEvents: {},
  },
  t: (ru) => ru,
  getMonthName: () => 'Ноябрь',
  go: vi.fn(),
  toggleTheme: vi.fn(),
  drinkWater: vi.fn(),
  openMetric: vi.fn(),
  changeCalendar: vi.fn(),
  openCalendarPage: vi.fn(),
  openStory: vi.fn(),
  showAllStories: vi.fn(),
  closeStory: vi.fn(),
  nextStory: vi.fn(),
  prevStory: vi.fn(),
  loadBiometrics: vi.fn(),
  loadTrainingPlans: vi.fn(),
  loadAchievements: vi.fn(),
  loadProgress: vi.fn(),
};

function renderHome(overrides = {}) {
  useApp.mockReturnValue({
    ...defaultContext,
    ...overrides,
    state: { ...defaultContext.state, ...overrides.state },
  });
  return render(
    <BrowserRouter>
      <AppProvider>
        <Home />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Home', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the reference home sections and profile entry', () => {
    renderHome();

    expect(screen.getByText('Привет, Михаил')).toBeInTheDocument();
    expect(screen.getByText('Готов к движению')).toBeInTheDocument();
    expect(screen.getByText('Истории')).toBeInTheDocument();
    expect(screen.getByText('Биоритм сегодня')).toBeInTheDocument();
    expect(screen.getByText('Активность сегодня')).toBeInTheDocument();
    expect(screen.getByText('2 / 8 стаканов')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Профиль' })).toBeInTheDocument();
  });

  it('routes profile and theme actions from the home header', async () => {
    const user = userEvent.setup();
    renderHome();

    await user.click(screen.getByRole('button', { name: 'Профиль' }));
    await user.click(screen.getByRole('button', { name: 'Сменить тему' }));

    expect(defaultContext.go).toHaveBeenCalledWith('profile');
    expect(defaultContext.toggleTheme).toHaveBeenCalledTimes(1);
  });

  it('routes the workout action to the training screen', () => {
    renderHome();

    fireEvent.click(screen.getByRole('button', { name: /Начать тренировку/ }));

    expect(defaultContext.go).toHaveBeenCalledWith('training');
  });

  it('opens metric details from a biorhythm card', () => {
    renderHome();

    fireEvent.click(screen.getByRole('button', { name: /Пульс/ }));

    expect(defaultContext.openMetric).toHaveBeenCalledWith('pulse');
  });

  it('adds water using the reference glass counter', () => {
    renderHome();

    fireEvent.click(
      screen.getByRole('button', { name: 'Добавить стакан воды' })
    );

    expect(defaultContext.drinkWater).toHaveBeenCalledTimes(1);
  });

  it('changes the mini calendar month and opens the full calendar', () => {
    renderHome();

    fireEvent.click(screen.getByRole('button', { name: 'Следующий месяц' }));
    fireEvent.click(screen.getByRole('button', { name: 'Все дни' }));

    expect(defaultContext.changeCalendar).toHaveBeenCalledWith(1);
    expect(defaultContext.openCalendarPage).toHaveBeenCalledWith(true);
    expect(defaultContext.go).toHaveBeenCalledWith('calendar');
  });

  it('opens story viewer controls for a selected story', async () => {
    const user = userEvent.setup();
    renderHome();

    await user.click(screen.getByRole('button', { name: /Восстановление/ }));

    expect(defaultContext.openStory).toHaveBeenCalledWith(0);
  });

  it('renders the selected story viewer', () => {
    renderHome({ state: { selectedStory: story } });

    expect(
      screen.getByRole('dialog', { name: 'Восстановление' })
    ).toBeVisible();
    expect(
      screen.getByRole('button', { name: 'Закрыть историю' })
    ).toBeInTheDocument();
  });

  it('offers the survey when personalization is incomplete', () => {
    renderHome({ state: { surveyCompleted: false } });

    fireEvent.click(screen.getByRole('button', { name: 'Продолжить опрос' }));

    expect(defaultContext.go).toHaveBeenCalledWith('survey');
  });

  it('scrolls the story strip and opens the body screen', () => {
    // jsdom does not implement scrollBy on elements.
    const scrollBy = vi.fn();
    const original = Element.prototype.scrollBy;
    Element.prototype.scrollBy = scrollBy;

    try {
      renderHome();

      fireEvent.click(screen.getByLabelText('Предыдущие истории'));
      expect(scrollBy).toHaveBeenLastCalledWith({
        left: -180,
        behavior: 'smooth',
      });

      fireEvent.click(screen.getByLabelText('Следующие истории'));
      expect(scrollBy).toHaveBeenLastCalledWith({
        left: 180,
        behavior: 'smooth',
      });

      fireEvent.click(screen.getByText('Открыть всё'));
      expect(defaultContext.go).toHaveBeenCalledWith('body');
    } finally {
      Element.prototype.scrollBy = original;
    }
  });

  it('navigates months and renders the life hack card', () => {
    renderHome({ state: { hackList: [['fa-dumbbell', 'Держи темп']] } });

    fireEvent.click(screen.getByLabelText('Предыдущий месяц'));
    expect(defaultContext.changeCalendar).toHaveBeenCalledWith(-1);

    fireEvent.click(screen.getByLabelText('Следующий месяц'));
    expect(defaultContext.changeCalendar).toHaveBeenLastCalledWith(1);

    expect(screen.getByText('Держи темп')).toBeInTheDocument();
  });
});
