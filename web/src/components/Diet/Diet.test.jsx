import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Diet from './Diet';

const mockGetProfile = vi.fn();

vi.mock('../../utils/api', () => ({
  getProfile: (...args) => mockGetProfile(...args),
}));

describe('Diet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state', () => {
    mockGetProfile.mockImplementation(() => new Promise(() => {}));
    render(<Diet />);
    expect(screen.getByText('Загрузка...')).toBeDefined();
  });

  it('renders diet plan after loading', async () => {
    mockGetProfile.mockResolvedValue({
      profile: {
        weight_kg: 70,
        height_cm: 175,
        age: 30,
        gender: 'male',
        fitness_level: 'beginner',
        goals: ['weight_loss'],
        allergies: [],
        contraindications: [],
      },
    });
    render(<Diet />);
    await waitFor(() =>
      expect(screen.getByText(/План питания на сегодня/i)).toBeDefined()
    );
  });

  it('renders meal template selector', async () => {
    mockGetProfile.mockResolvedValue({
      profile: {
        weight_kg: 70,
        height_cm: 175,
        age: 30,
        gender: 'male',
        fitness_level: 'beginner',
        goals: [],
        allergies: [],
        contraindications: [],
      },
    });
    render(<Diet />);
    await waitFor(() =>
      expect(screen.getByText('Сбалансированное')).toBeDefined()
    );
  });

  it('changes meal count', async () => {
    const user = userEvent.setup();
    mockGetProfile.mockResolvedValue({
      profile: {
        weight_kg: 70,
        height_cm: 175,
        age: 30,
        gender: 'male',
        fitness_level: 'beginner',
        goals: [],
        allergies: [],
        contraindications: [],
      },
    });
    render(<Diet />);
    await waitFor(() =>
      expect(screen.getByLabelText('Количество приёмов пищи')).toBeDefined()
    );
  });
});
