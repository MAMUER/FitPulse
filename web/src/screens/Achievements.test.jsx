import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Achievements from './Achievements';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'achievements',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    achievements: [],
    achievementsBackend: [],
  },
  t: (ru, _en) => ru,
  go: vi.fn(),
  loadAchievements: vi.fn(() => Promise.resolve()),
};

function renderAchievements() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Achievements />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Achievements', () => {
  it('renders achievements screen', () => {
    renderAchievements();
    expect(screen.getByText('Достижения')).toBeInTheDocument();
  });

  it('shows loading state initially', () => {
    renderAchievements();
    expect(screen.getByText('Загрузка...')).toBeInTheDocument();
  });

  it('navigates back home', () => {
    renderAchievements();
    screen.getByText('Назад').click();
    expect(defaultContext.go).toHaveBeenCalledWith('home');
  });
});
