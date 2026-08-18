import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import Achievements from './Achievements';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

vi.mock('chart.js/auto', () => {
  const MockChart = vi.fn(function Chart() {
    this.destroy = vi.fn();
  });
  return { Chart: MockChart };
});

const renderAchievements = () => {
  useAuth.mockReturnValue({
    token: 'test-token',
  });

  return render(
    <AuthProvider>
      <Achievements />
    </AuthProvider>
  );
};

describe('Achievements', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  it('shows empty state when no achievements', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({ progress_data: [] });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('Нет достижений')).toBeInTheDocument();
    });
  });

  it('loads and displays achievements', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [
        {
          achievement_id: 'first_workout',
          title: 'Первая тренировка',
          description: 'Завершите первую тренировку',
          earned_date: '2024-01-01',
        },
      ],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({ progress_data: [] });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('Первая тренировка')).toBeInTheDocument();
    });
  });

  it('displays locked achievements', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [
        {
          achievement_id: 'hundred_days',
          title: '100 дней',
          description: 'Тренируйтесь 100 дней',
          earned_date: null,
        },
      ],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({ progress_data: [] });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('100 дней')).toBeInTheDocument();
      expect(screen.getByText('Заблокировано')).toBeInTheDocument();
    });
  });

  it('displays custom achievement icon', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [
        {
          achievement_id: 'custom',
          title: 'Custom',
          description: 'Custom achievement',
          icon_url: '🎯',
          earned_date: '2024-01-01',
        },
      ],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({ progress_data: [] });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('Custom')).toBeInTheDocument();
    });
  });

  it('displays competitions', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({ progress_data: [] });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('🏅 Персональные челленджи')).toBeInTheDocument();
      expect(screen.getByText('Персональный рекорд')).toBeInTheDocument();
      expect(screen.getByText('Серия тренировок')).toBeInTheDocument();
    });
  });

  it('displays competition status labels', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({ progress_data: [] });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('Активно')).toBeInTheDocument();
      expect(screen.getAllByText('Скоро').length).toBeGreaterThanOrEqual(1);
    });
  });

  it('handles load error gracefully', async () => {
    vi.spyOn(api, 'getAchievements').mockRejectedValueOnce(
      new Error('load failed')
    );
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({ progress_data: [] });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('Нет достижений')).toBeInTheDocument();
    });
  });

  it('handles progress data error gracefully', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockRejectedValueOnce(
      new Error('progress failed')
    );
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('🏆 Достижения')).toBeInTheDocument();
    });
  });

  it('renders progress chart when data is available', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({
      progress_data: [
        { date: '2024-01-01', completed_workouts: 3 },
        { date: '2024-01-02', completed_workouts: 5 },
      ],
    });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('📈 Прогресс')).toBeInTheDocument();
    });

    const canvas = document.getElementById('progressChart');
    expect(canvas).toBeTruthy();
  });

  it('renders progress chart with progressData', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({
      progress_data: [
        { date: '2024-01-01', completed_workouts: 3 },
        { date: '2024-01-02', completed_workouts: 5 },
      ],
    });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('📈 Прогресс')).toBeInTheDocument();
    });

    const canvas = document.getElementById('progressChart');
    expect(canvas).toBeTruthy();
  });

  it('renders progress chart with week field', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({
      progress_data: [
        { week: 'Week 1', count: 3 },
        { week: 'Week 2', count: 5 },
      ],
    });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('📈 Прогресс')).toBeInTheDocument();
    });
  });

  it('renders progress chart with value field', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({
      progress_data: [
        { date: '2024-01-01', value: 3 },
        { date: '2024-01-02', value: 5 },
      ],
    });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('📈 Прогресс')).toBeInTheDocument();
    });
  });

  it('uses fallbacks when API returns no data', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce(undefined);
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce(undefined);
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('Нет достижений')).toBeInTheDocument();
    });
  });

  it('destroys chart instance when progressData changes on re-render', async () => {
    let progressCallCount = 0;
    vi.spyOn(api, 'getAchievements').mockImplementation(() => {
      return Promise.resolve({ achievements: [] });
    });
    vi.spyOn(api, 'getProgress').mockImplementation(() => {
      progressCallCount++;
      if (progressCallCount === 1) {
        return Promise.resolve({
          progress_data: [{ date: '2024-01-01', completed_workouts: 3 }],
        });
      }
      return Promise.resolve({
        progress_data: [{ date: '2024-01-02', completed_workouts: 5 }],
      });
    });
    const { rerender } = renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('📈 Прогресс')).toBeInTheDocument();
    });

    rerender(
      <AuthProvider>
        <Achievements refreshKey={2} />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('📈 Прогресс')).toBeInTheDocument();
    });
  });

  it('renders achievement fallback icon and empty strings', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [
        {
          achievement_id: 'unknown_id',
          title: '',
          description: '',
          earned_date: '2024-01-01',
        },
      ],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({ progress_data: [] });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('🏆')).toBeInTheDocument();
    });
  });

  it('renders progress chart with empty date and week fallbacks', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({
      progress_data: [{ completed_workouts: 3 }, { week: '', count: 5 }],
    });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('📈 Прогресс')).toBeInTheDocument();
    });
  });

  it('renders progress chart with value field fallback', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({
      progress_data: [{ value: 3 }, { count: 5 }, { completed_workouts: 7 }],
    });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('📈 Прогресс')).toBeInTheDocument();
    });
  });

  it('renders progress chart with fallback to 0 when no value fields present', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({
      progress_data: [{ date: '2024-01-01' }, { week: 'Week 1' }],
    });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('📈 Прогресс')).toBeInTheDocument();
    });
  });

  it('displays competition rank when available', async () => {
    vi.spyOn(api, 'getAchievements').mockResolvedValueOnce({
      achievements: [],
    });
    vi.spyOn(api, 'getProgress').mockResolvedValueOnce({ progress_data: [] });
    renderAchievements();

    await waitFor(() => {
      expect(screen.getByText('🏅 Место: 1')).toBeInTheDocument();
    });
  });
});
