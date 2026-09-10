import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ML from './ML';

const mockClassifyState = vi.fn();
const mockGenerateMLPlan = vi.fn();
const mockGetPlan = vi.fn();

vi.mock('../../utils/api', () => ({
  classifyState: (...args) => mockClassifyState(...args),
  generateMLPlan: (...args) => mockGenerateMLPlan(...args),
  getPlan: (...args) => mockGetPlan(...args),
}));

describe('ML', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders ML page', () => {
    render(<ML />);
    expect(screen.getByText('Классификация состояния')).toBeDefined();
    expect(screen.getByText('Генерация плана')).toBeDefined();
  });

  it('classifies state when button clicked', async () => {
    const user = userEvent.setup();
    mockClassifyState.mockResolvedValue({
      predicted_class_ru: 'Восстановление',
      confidence: 0.9,
    });
    render(<ML />);
    await user.click(screen.getByText('Анализировать'));
    await waitFor(() =>
      expect(screen.getByText('Восстановление')).toBeDefined()
    );
  });

  it('generates plan when form submitted', async () => {
    const user = userEvent.setup();
    mockClassifyState.mockResolvedValue({});
    mockGenerateMLPlan.mockResolvedValue({ plan_id: '1' });
    render(<ML />);
    await user.click(screen.getByText('Сгенерировать план'));
    expect(mockGenerateMLPlan).toHaveBeenCalled();
  });
});
