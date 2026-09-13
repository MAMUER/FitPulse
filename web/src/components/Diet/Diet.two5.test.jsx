import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../../utils/api', () => ({
  getProfile: () => Promise.resolve({
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
}));

import Diet from './Diet';

describe('Diet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state', async () => {
    render(<Diet />);
    expect(screen.getByText('Загрузка...')).toBeDefined();
  });

  it('renders diet plan after loading', async () => {
    render(<Diet />);
    await waitFor(() =>
      expect(screen.queryByText('Загрузка...')).not.toBeInTheDocument()
    );
    expect(screen.getByText(/План питания на сегодня/i)).toBeDefined();
  });
});
