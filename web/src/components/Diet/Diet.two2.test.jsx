import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockGetProfile = vi.fn();

vi.mock('../../utils/api', () => ({
  getProfile: (...args) => mockGetProfile(...args),
}));

import Diet from './Diet';

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
              }),
            100
          )
        )
    );
    render(<Diet />);
    expect(screen.getByText('Загрузка...')).toBeDefined();
  });

  it('renders something else', async () => {
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
    expect(screen.getByText('Загрузка...')).toBeDefined();
  });
});
