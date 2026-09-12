import { describe, expect, it, vi } from 'vitest';

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
