import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Achievements from './Achievements';

vi.mock('../../utils/api', () => ({
  getAchievements: vi.fn(),
  getProgress: vi.fn(),
}));

import { getAchievements, getProgress } from '../../utils/api';

describe('Achievements', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading and then achievements', async () => {
    getAchievements.mockResolvedValue({ achievements: [] });
    getProgress.mockResolvedValue({ progress_data: [] });
    render(<Achievements refreshKey={0} />);
    expect(screen.getByText(/Достижения/i)).toBeDefined();
    await waitFor(() => expect(getAchievements).toHaveBeenCalled());
  });

  it('renders achievements list when data is loaded', async () => {
    getAchievements.mockResolvedValue({
      achievements: [
        {
          achievement_id: 'first_workout',
          title: 'Первая тренировка',
          description: 'Завершите первую тренировку',
          earned_date: '2024-01-01',
          icon_url: '',
        },
      ],
    });
    getProgress.mockResolvedValue({ progress_data: [] });
    render(<Achievements refreshKey={0} />);
    await waitFor(() =>
      expect(screen.getByText('Первая тренировка')).toBeDefined()
    );
  });

  it('renders empty state when no achievements', async () => {
    getAchievements.mockResolvedValue({ achievements: [] });
    getProgress.mockResolvedValue({ progress_data: [] });
    render(<Achievements refreshKey={0} />);
    await waitFor(() =>
      expect(screen.getByText('Нет достижений')).toBeDefined()
    );
  });

  it('renders competitions', async () => {
    getAchievements.mockResolvedValue({ achievements: [] });
    getProgress.mockResolvedValue({ progress_data: [] });
    render(<Achievements refreshKey={0} />);
    expect(screen.getByText('Персональный рекорд')).toBeDefined();
    expect(screen.getByText('Серия тренировок')).toBeDefined();
  });

  it('renders progress chart canvas', async () => {
    getAchievements.mockResolvedValue({ achievements: [] });
    getProgress.mockResolvedValue({ progress_data: [] });
    render(<Achievements refreshKey={0} />);
    expect(screen.getByLabelText(/График динамики пульса/i)).toBeDefined();
  });
});
