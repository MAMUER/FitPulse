import { describe, expect, it, vi } from 'vitest';

vi.mock('../../utils/api', () => ({
  getProfile: () => Promise.resolve({}),
}));

import Diet from './Diet';

describe('Diet', () => {
  it('imports', () => {
    expect(Diet).toBeDefined();
  });
});
