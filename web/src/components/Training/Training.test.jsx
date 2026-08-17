import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import Training from './Training';

vi.mock('../../utils/api');
vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const renderTraining = () => {
  useAuth.mockReturnValue({
    token: 'test-token',
  });

  return render(
    <AuthProvider>
      <Training />
    </AuthProvider>
  );
};

describe('Training', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading state initially', () => {
    api.getTrainingPlans.mockImplementation(() => new Promise(() => {}));
    renderTraining();
    expect(screen.getByText('Загрузка программ...')).toBeInTheDocument();
  });

  it('displays training_goal and duration_weeks on plan card', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({
      plans: [
        {
          plan_id: 1,
          plan_data: { name: 'Strength Plan' },
          training_goal: 'Сила',
          duration_weeks: 8,
        },
      ],
    });
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Strength Plan')).toBeInTheDocument();
    });

    expect(screen.getByText('Цель: Сила')).toBeInTheDocument();
    expect(screen.getByText('8 недель')).toBeInTheDocument();
  });

  it('falls back to default goal and duration when fields are missing', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({
      plans: [
        {
          plan_id: 1,
          plan_data: { name: 'Default Plan' },
        },
      ],
    });
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Default Plan')).toBeInTheDocument();
    });

    expect(screen.getByText('Цель: Общая тренировка')).toBeInTheDocument();
    expect(screen.getByText('4 недель')).toBeInTheDocument();
  });

  it('shows empty state when no plans', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({ plans: [] });
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Нет активных программ')).toBeInTheDocument();
    });
  });

  it('displays training plans', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({
      plans: [
        {
          plan_id: 1,
          plan_data: { name: 'Test Plan' },
          training_goal: 'Strength',
          duration_weeks: 4,
        },
      ],
    });
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Test Plan')).toBeInTheDocument();
    });
  });

  it('generates a new plan', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({ plans: [] });
    api.classifyState.mockResolvedValueOnce({
      predicted_class: 'strength',
      confidence: 0.9,
    });
    api.generateTrainingPlan.mockResolvedValueOnce({});
    api.getTrainingPlans.mockResolvedValueOnce({
      plans: [
        {
          plan_id: 2,
          plan_data: { name: 'Generated Plan' },
          training_goal: 'Strength',
          duration_weeks: 4,
        },
      ],
    });
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Нет активных программ')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Generated Plan')).toBeInTheDocument();
    });
  });

  it('handles load error gracefully', async () => {
    api.getTrainingPlans.mockRejectedValueOnce(new Error('load failed'));
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Нет активных программ')).toBeInTheDocument();
    });
  });

  it('handles generation error gracefully', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({ plans: [] });
    api.classifyState.mockResolvedValueOnce({
      predicted_class: 'strength',
      confidence: 0.9,
    });
    api.generateTrainingPlan.mockRejectedValueOnce(
      new Error('generation failed')
    );
    vi.spyOn(window, 'alert').mockImplementation(() => {});
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Нет активных программ')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText('Сгенерировать план'));

    await waitFor(() => {
      expect(window.alert).toHaveBeenCalledWith(
        'Ошибка генерации: generation failed'
      );
    });
  });

  it('uses default classification when classify fails', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({ plans: [] });
    api.classifyState.mockRejectedValueOnce(new Error('classify failed'));
    api.generateTrainingPlan.mockResolvedValueOnce({});
    api.getTrainingPlans.mockResolvedValueOnce({
      plans: [
        {
          plan_id: 2,
          plan_data: { name: 'Default Plan' },
          training_goal: 'recovery',
          duration_weeks: 4,
        },
      ],
    });
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Нет активных программ')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Default Plan')).toBeInTheDocument();
    });
  });

  it('falls back to empty plans array when API returns undefined', async () => {
    api.getTrainingPlans.mockResolvedValueOnce(undefined);
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Нет активных программ')).toBeInTheDocument();
    });
  });

  it('renders default plan name when plan_data name is missing', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({
      plans: [
        {
          plan_id: 1,
          plan_data: {},
          training_goal: 'Сила',
          duration_weeks: 4,
        },
      ],
    });
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Персонализированная программа')).toBeInTheDocument();
    });
  });

  it('uses predicted_class and confidence from classifyState', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({ plans: [] });
    api.classifyState.mockResolvedValueOnce({
      predicted_class: 'endurance',
      confidence: 0.85,
    });
    api.generateTrainingPlan.mockResolvedValueOnce({});
    api.getTrainingPlans.mockResolvedValueOnce({
      plans: [
        {
          plan_id: 2,
          plan_data: { name: 'Endurance Plan' },
          training_goal: 'Endurance',
          duration_weeks: 6,
        },
      ],
    });
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Нет активных программ')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Endurance Plan')).toBeInTheDocument();
    });
  });

  it('falls back to default training class when classifyState returns no predicted_class', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({ plans: [] });
    api.classifyState.mockResolvedValueOnce({
      confidence: 0.7,
    });
    api.generateTrainingPlan.mockResolvedValueOnce({});
    api.getTrainingPlans.mockResolvedValueOnce({
      plans: [
        {
          plan_id: 2,
          plan_data: { name: 'Default Class Plan' },
          training_goal: 'Recovery',
          duration_weeks: 4,
        },
      ],
    });
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Нет активных программ')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Default Class Plan')).toBeInTheDocument();
    });
  });

  it('falls back to default confidence when classifyState returns falsy confidence', async () => {
    api.getTrainingPlans.mockResolvedValueOnce({ plans: [] });
    api.classifyState.mockResolvedValueOnce({
      predicted_class: 'strength',
      confidence: 0,
    });
    api.generateTrainingPlan.mockResolvedValueOnce({});
    api.getTrainingPlans.mockResolvedValueOnce({
      plans: [
        {
          plan_id: 2,
          plan_data: { name: 'Default Confidence Plan' },
          training_goal: 'Strength',
          duration_weeks: 4,
        },
      ],
    });
    renderTraining();

    await waitFor(() => {
      expect(screen.getByText('Нет активных программ')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText('Сгенерировать план'));

    await waitFor(() => {
      expect(screen.getByText('Default Confidence Plan')).toBeInTheDocument();
    });
  });
});
