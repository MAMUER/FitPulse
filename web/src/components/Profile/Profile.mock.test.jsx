import { describe, expect, it, vi } from 'vitest';

const mockApi = vi.fn();

vi.mock('../../utils/api', () => ({
  getProfile: () => mockApi(),
}));

describe('Mock check', () => {
  it('awaits mock', async () => {
    mockApi.mockResolvedValue({ hello: 'world' });
    const result = await mockApi();
    expect(result).toEqual({ hello: 'world' });
  });
});
