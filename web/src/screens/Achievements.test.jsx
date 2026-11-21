import { act, render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Achievements from './Achievements';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

function makeContext(overrides = {}) {
  return {
    state: {
      achievements: [],
      achievementsBackend: [],
      streak: 0,
      maxStreak: 0,
      ...overrides.state,
    },
    t: (ru) => ru,
    go: vi.fn(),
    loadAchievements: vi.fn(() => Promise.resolve()),
    showAchievement: vi.fn(),
    ...overrides,
  };
}

function renderPage(context) {
  useApp.mockReturnValue(context);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Achievements />
      </AppProvider>
    </BrowserRouter>
  );
}

async function settle() {
  await act(async () => {
    await Promise.resolve();
  });
}

describe('Achievements', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads achievements on mount and renders the title', async () => {
    const loadAchievements = vi.fn(() => Promise.resolve());
    renderPage(makeContext({ loadAchievements }));

    expect(screen.getByText('Достижения')).toBeInTheDocument();
    expect(loadAchievements).toHaveBeenCalledTimes(1);
    await settle();
  });

  it('shows a loading marker until the request settles', async () => {
    renderPage(makeContext());
    expect(screen.getByText('Загрузка...')).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
  });

  it('shows an empty state when there are no achievements', async () => {
    renderPage(makeContext());
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Нет достижений')).toBeInTheDocument();
  });

  it('renders achievements from achievementsBackend', async () => {
    renderPage(
      makeContext({
        state: {
          achievementsBackend: [
            {
              id: 1,
              name: 'Первый шаг',
              desc: 'Проведите первую тренировку',
              done: true,
            },
            { id: 2, name: 'Марафон', description: 'Наберите 100 очков' },
          ],
        },
      })
    );
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });

    expect(screen.getByText('Первый шаг')).toBeInTheDocument();
    expect(screen.getByText('Проведите первую тренировку')).toBeInTheDocument();
    expect(screen.getByText('Марафон')).toBeInTheDocument();
    expect(screen.getByText('Наберите 100 очков')).toBeInTheDocument();
  });

  it('falls back to state.achievements when the backend list is empty', async () => {
    renderPage(
      makeContext({
        state: {
          achievementsBackend: [],
          achievements: [{ name: 'Тест', desc: 'Описание' }],
        },
      })
    );
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });

    expect(screen.getByText('Тест')).toBeInTheDocument();
    expect(screen.getByText('Описание')).toBeInTheDocument();
  });

  it('marks cards done when done=true or achieved=true', async () => {
    renderPage(
      makeContext({
        state: {
          achievementsBackend: [
            { id: 1, name: 'Готово', done: true },
            { id: 2, name: 'Достигнуто', achieved: true },
            { id: 3, name: 'Не готово' },
          ],
        },
      })
    );
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });

    expect(document.querySelectorAll('.achievement-card.done')).toHaveLength(2);
  });

  it('uses fa-trophy when no icon is provided and the given icon otherwise', async () => {
    renderPage(
      makeContext({
        state: {
          achievementsBackend: [
            { id: 1, name: 'Без иконки' },
            { id: 2, name: 'Со звездой', icon: 'fa-star' },
          ],
        },
      })
    );
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });

    const plain = screen.getByText('Без иконки').closest('.achievement-card');
    expect(plain?.querySelector('.fa-trophy')).toBeInTheDocument();

    const starred = screen.getByText('Со звездой').closest('.achievement-card');
    expect(starred?.querySelector('.fa-star')).toBeInTheDocument();
  });

  it('shows summary counters and progress', async () => {
    renderPage(
      makeContext({
        state: {
          streak: 5,
          maxStreak: 12,
          achievementsBackend: [
            { id: 1, name: 'A', done: true },
            { id: 2, name: 'B', done: true },
            { id: 3, name: 'C' },
            { id: 4, name: 'D' },
          ],
        },
      })
    );
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });

    expect(screen.getByText('2/4')).toBeInTheDocument();
    expect(screen.getByText('50%')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText('12')).toBeInTheDocument();
  });

  it('navigates home from the back button', async () => {
    const go = vi.fn();
    renderPage(makeContext({ go }));
    await settle();

    screen.getByText('Назад').click();
    expect(go).toHaveBeenCalledWith('home');
  });

  it('opens the achievement detail via showAchievement', async () => {
    const showAchievement = vi.fn();
    renderPage(
      makeContext({
        showAchievement,
        state: { achievementsBackend: [{ id: 1, name: 'Первый шаг' }] },
      })
    );
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });

    screen.getByText('Первый шаг').click();
    expect(showAchievement).toHaveBeenCalledWith('Первый шаг');
  });

  it('uses English labels when t() returns the English string', async () => {
    renderPage(
      makeContext({
        t: (_ru, en) => en,
      })
    );
    await waitFor(() => {
      expect(screen.queryByText('Loading...')).not.toBeInTheDocument();
    });

    expect(screen.getByText('Achievements')).toBeInTheDocument();
    expect(screen.getByText('No achievements')).toBeInTheDocument();
    expect(screen.getByText('Back')).toBeInTheDocument();
  });

  it('does not crash when an achievement has neither id nor name', async () => {
    renderPage(
      makeContext({
        state: { achievementsBackend: [{ desc: 'Только описание' }] },
      })
    );
    await waitFor(() => {
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument();
    });

    expect(screen.getByText('Только описание')).toBeInTheDocument();
  });
});
