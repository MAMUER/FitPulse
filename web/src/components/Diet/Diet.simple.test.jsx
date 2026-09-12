import { describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';

const mockGetProfile = vi.fn();

vi.mock('../../utils/api', () => ({
  getProfile: (...args) => mockGetProfile(...args),
}));

import Diet from './Diet';

describe('Diet', () => {
  it('renders loading state', () => {
    mockGetProfile.mockImplementation(() => new Promise(() => {}));
    render(<Diet />);
    expect(screen.getByText('Загрузка...')).toBeDefined();
  });
});
