import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Training from './Training';

const mockGetTrainingPlans = vi.fn();
const mockClassifyState = vi.fn();
const mockGenerateTrainingPlan = vi.fn();

vi.mock('../../utils/api', () => ({
  getTrainingPlans: (...args) => mockGetTrainingPlans(...args),
  classifyState: (...args) => mockClassifyState(...args),
  generateTrainingPlan: (...args) => mockGenerateTrainingPlan(...args),
}));

describe('Training', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state', () => {
    mockGetTrainingPlans.mockImplementation(() => new Promise(() => {}));
    render(<Training />);
    expect(screen.getByText('Загрузка программ...')).toBeDefined();
  });

  it('renders empty state when no plans', async () => {
    mockGetTrainingPlans.mockResolvedValue({ plans: [] });
    render(<Training />);
    await waitFor(() =>
      expect(screen.getByText('Нет активных программ')).toBeDefined()
    );
  });

  it('renders plans list', async () => {
    mockGetTrainingPlans.mockResolvedValue({
      plans: [
        {
          plan_id: '1',
          plan_data: { name: 'Test Plan' },
          training_goal: 'Strength',
          duration_weeks: 4,
        },
      ],
    });
    render(<Training />);
    await waitFor(() => expect(screen.getByText('Test Plan')).toBeDefined());
  });

  it('generates plan when button clicked', async () => {
    const user = userEvent.setup();
    mockGetTrainingPlans.mockResolvedValue({ plans: [] });
    mockClassifyState.mockResolvedValue({
      predicted_class: 'recovery',
      confidence: 0.5,
    });
    mockGenerateTrainingPlan.mockResolvedValue({});
    render(<Training />);
    await waitFor(() =>
      expect(screen.getByLabelText('Сгенерировать план')).toBeDefined()
    );
    await user.click(screen.getByLabelText('Сгенерировать план'));
    expect(mockGenerateTrainingPlan).toHaveBeenCalled();
  });
});
