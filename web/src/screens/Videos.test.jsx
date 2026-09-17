import { render, screen } from '@testing-library/react';
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
    trainingVideos: [
      { title: 'Кардио', minutes: 20, kcal: 180 },
      { title: 'Силовая', minutes: 30, kcal: 240 },
    ],
  },
  t: (ru, _en) => ru,
  go: vi.fn(),
};

function renderVideos() {
  useApp.mockReturnValue(defaultContext);
  return render(
    <AppProvider>
      <Videos />
    </AppProvider>
  );
}

describe('Videos', () => {
  it('renders video list', () => {
    renderVideos();
    expect(screen.getByText('Видео-тренировки')).toBeInTheDocument();
    expect(screen.getByText('Кардио')).toBeInTheDocument();
    expect(screen.getByText('Силовая')).toBeInTheDocument();
  });

  it('navigates to training on start', () => {
    renderVideos();
    screen.getAllByText('Старт')[0].click();
    expect(defaultContext.go).toHaveBeenCalledWith('training');
  });
});
