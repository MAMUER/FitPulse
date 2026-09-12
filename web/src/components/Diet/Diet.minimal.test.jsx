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
    mockGetProfile.mockReset();
  });

  it('test 1', async () => {
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

    // Wait for loading to finish
    await screen.findByText(/План питания на сегодня/i, {}, { timeout: 5000 });
  });
});
