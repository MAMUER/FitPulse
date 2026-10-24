import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Achievements from './Achievements';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const baseContext = (overrides = {}) => ({
  state: {
    screen: 'achievements',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    achievements: [],
    achievementsBackend: [],
    ...overrides.state,
  },
  t: (ru, _en) => ru,
  go: vi.fn(),
  loadAchievements: vi.fn(() => Promise.resolve()),
  ...overrides,
});

function renderAchievements(context) {
  useApp.mockReturnValue(context);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Achievements />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Achievements', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders achievements screen', () => {
    renderAchievements(baseContext());
    expect(screen.getByText('Достижения')).toBeInTheDocument();
  });

  it('shows loading state initially', () => {
    renderAchievements(baseContext());
    expect(screen.getByText('Загрузка...')).toBeInTheDocument();
  });

  it('shows no achievements when empty after loading completes', async () => {
    renderAchievements(baseContext());
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Нет достижений')).toBeInTheDocument();
  });

  it('loads achievements on mount', () => {
    const loadAchievements = vi.fn(() => Promise.resolve());
    renderAchievements(baseContext({ loadAchievements }));
    expect(loadAchievements).toHaveBeenCalledTimes(1);
  });

  it('renders achievements from achievementsBackend', async () => {
    const achievementsBackend = [
      {
        id: 1,
        name: 'Первый шаг',
        desc: 'Проведите первую тренировку',
        icon: 'fa-star',
        done: true,
      },
      {
        id: 2,
        name: 'Марфетка',
        description: 'Наберите 100 очков',
        achieved: false,
      },
    ];
    renderAchievements(baseContext({ state: { achievementsBackend } }));
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Первый шаг')).toBeInTheDocument();
    expect(screen.getByText('Марфетка')).toBeInTheDocument();
    expect(screen.getByText('Проведите первую тренировку')).toBeInTheDocument();
    expect(screen.getByText('Наберите 100 очков')).toBeInTheDocument();
  });

  it('falls back to state.achievements when backend is empty', async () => {
    const achievements = [{ name: 'Тест', desc: 'Описание' }];
    renderAchievements(baseContext({ state: { achievementsBackend: [], achievements } }));
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Тест')).toBeInTheDocument();
    expect(screen.getByText('Описание')).toBeInTheDocument();
  });

  it('renders default icon when no icon provided', async () => {
    const achievementsBackend = [{ id: 1, name: 'Без иконки' }];
    renderAchievements(baseContext({ state: { achievementsBackend } }));
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
    const nameElement = screen.getByText('Без иконки');
    const card = nameElement.closest('.achievement-card');
    expect(card.querySelector('.fa-trophy')).toBeInTheDocument();
  });

  it('renders custom icon when provided', async () => {
    const achievementsBackend = [{ id: 1, name: 'Со звездой', icon: 'fa-star' }];
    renderAchievements(baseContext({ state: { achievementsBackend } }));
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
    const nameElement = screen.getByText('Со звездой');
    const card = nameElement.closest('.achievement-card');
    expect(card.querySelector('.fa-star')).toBeInTheDocument();
  });

  it('applies done class when achievement is done', async () => {
    const achievementsBackend = [{ id: 1, name: 'Готово', done: true }];
    renderAchievements(baseContext({ state: { achievementsBackend } }));
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
    const cards = document.querySelectorAll('.achievement-card.done');
    expect(cards.length).toBe(1);
  });

  it('applies done class when achievement is achieved', async () => {
    const achievementsBackend = [{ id: 1, name: 'Достигнуто', achieved: true }];
    renderAchievements(baseContext({ state: { achievementsBackend } }));
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
    const cards = document.querySelectorAll('.achievement-card.done');
    expect(cards.length).toBe(1);
  });

  it('does not apply done class when not done', async () => {
    const achievementsBackend = [{ id: 1, name: 'Не готово' }];
    renderAchievements(baseContext({ state: { achievementsBackend } }));
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
    const cards = document.querySelectorAll('.achievement-card.done');
    expect(cards.length).toBe(0);
  });

  it('uses key fallback when id and name missing', async () => {
    const achievementsBackend = [{ desc: 'Описание только' }];
    renderAchievements(baseContext({ state: { achievementsBackend } }));
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Описание только')).toBeInTheDocument();
  });

  it('navigates back home', () => {
    const go = vi.fn();
    renderAchievements(baseContext({ go }));
    screen.getByText('Назад').click();
    expect(go).toHaveBeenCalledWith('home');
  });

  it('uses English translations when language is en', async () => {
    const enContext = baseContext({ state: { language: 'en' }, t: (_ru, en) => en });
    renderAchievements(enContext);
    await waitFor(() => {
      expect(screen.queryByText('Loading...')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Achievements')).toBeInTheDocument();
    expect(screen.getByText('Back')).toBeInTheDocument();
    expect(screen.getByText('No achievements')).toBeInTheDocument();
  });
});