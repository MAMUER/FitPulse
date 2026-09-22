import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AppProvider, useApp } from '../contexts/AppContext';
import Videos from './Videos';

vi.mock('../contexts/AppContext', async () => {
  const actual = await vi.importActual('../contexts/AppContext');
  return {
    ...actual,
    useApp: vi.fn(),
  };
});

const defaultContext = {
  state: {
    screen: 'videos',
    theme: 'dark',
    highContrast: false,
    language: 'ru',
    videos: [
      { id: 'v1', title: 'Кардио', minutes: 20, kcal: 180 },
      { id: 'v2', title: 'Силовая', minutes: 30, kcal: 240 },
    ],
    trainingVideos: [],
  },
  t: (ru, _en) => ru,
  go: vi.fn(),
  loadVideos: vi.fn(() => Promise.resolve()),
};

function renderVideos() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <BrowserRouter>
      <AppProvider>
        <Videos />
      </AppProvider>
    </BrowserRouter>
  );
}

describe('Videos', () => {
  it('renders video list', () => {
    renderVideos();
    expect(screen.getByText('Видео-тренировки')).toBeInTheDocument();
    expect(screen.getByText('Кардио')).toBeInTheDocument();
  });

  it('navigates to training on start', () => {
    renderVideos();
    const startButtons = screen.getAllByText('Старт');
    startButtons[0].click();
    expect(defaultContext.go).toHaveBeenCalledWith('training');
  });
});
