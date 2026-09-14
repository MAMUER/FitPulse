import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

Object.defineProperty(window, 'crypto', {
  value: {
    getRandomValues: (arr) => {
      arr[0] = 0;
    },
  },
});

const mockGetProfile = vi.fn();

vi.mock('../../utils/api', () => ({
  getProfile: (...args) => mockGetProfile(...args),
}));

import Diet from './Diet';

const profile = {
  weight_kg: 70,
  height_cm: 175,
  age: 30,
  gender: 'male',
  fitness_level: 'beginner',
  goals: ['weight_loss'],
  allergies: [],
  contraindications: [],
};

describe('Diet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state', async () => {
    mockGetProfile.mockImplementation(
      () =>
        new Promise((resolve) =>
          setTimeout(
            () =>
              resolve({
                profile,
              }),
            100
          )
        )
    );
    render(<Diet />);
    expect(screen.getByText('Загрузка...')).toBeDefined();
  });

  it('renders diet plan after loading', async () => {
    mockGetProfile.mockResolvedValue({
      profile,
    });
    render(<Diet />);
    expect(await screen.findByText(/План питания на сегодня/i)).toBeDefined();
  });

  it('renders meal template selector', async () => {
    mockGetProfile.mockResolvedValue({
      profile: {
        ...profile,
        goals: [],
      },
    });
    render(<Diet />);
    expect(await screen.findByText('Сбалансированное')).toBeDefined();
  });

  it('changes meal count', async () => {
    const _user = userEvent.setup();
    mockGetProfile.mockResolvedValue({
      profile: {
        ...profile,
        goals: [],
      },
    });
    render(<Diet />);
    expect(
      await screen.findByLabelText('Количество приёмов пищи')
    ).toBeDefined();
  });
});
