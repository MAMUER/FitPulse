import { describe, expect, it, vi } from 'vitest';

const mockGetProfile = vi.fn();

vi.mock('../../utils/api', () => ({
  getProfile: (...args) => mockGetProfile(...args),
}));

import Diet from './Diet';

describe('Diet', () => {
  it('imports', () => {
    expect(Diet).toBeDefined();
  });
});
