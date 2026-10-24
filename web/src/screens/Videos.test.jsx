import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Videos from './Videos';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

describe('Videos', () => {
  it('renders empty state', () => {
    const loadVideos = vi.fn();
    useApp.mockReturnValue({
      state: { videos: [], trainingVideos: [] },
      t: (ru, _en) => ru,
      go: vi.fn(),
      loadVideos,
    });

    render(<Videos />);
    expect(loadVideos).toHaveBeenCalled();
    expect(screen.getByText('Нет видео')).toBeInTheDocument();
  });

  it('renders videos from trainingVideos fallback', () => {
    const loadVideos = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: {
        videos: [],
        trainingVideos: [{ id: 1, title: 'Test', minutes: 10, kcal: 100 }],
      },
      t: (ru, _en) => ru,
      go,
      loadVideos,
    });

    render(<Videos />);
    expect(screen.getByText('Test')).toBeInTheDocument();
    expect(screen.getByText(/10/)).toBeInTheDocument();
  });

  it('uses state.videos when available', () => {
    const loadVideos = vi.fn();
    const go = vi.fn();
    useApp.mockReturnValue({
      state: {
        videos: [{ id: 2, title: 'V2', duration: 15, calories: 200 }],
        trainingVideos: [],
      },
      t: (_ru, en) => en,
      go,
      loadVideos,
    });

    render(<Videos />);
    expect(screen.getByText('V2')).toBeInTheDocument();
  });
});
