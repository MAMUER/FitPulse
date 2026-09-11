import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useDevices } from './useDevices';

const mockUseAuth = vi.fn();
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => mockUseAuth(),
}));

const mockGetProviders = vi.fn();
const mockDisconnectIntegration = vi.fn();

vi.mock('../../utils/api', () => ({
  getProviders: (...args) => mockGetProviders(...args),
  disconnectIntegration: (...args) => mockDisconnectIntegration(...args),
}));

describe('useDevices', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuth.mockReturnValue({ token: 'test-token' });
  });

  it('initializes with idle status', () => {
    const { result } = renderHook(() => useDevices());
    expect(result.current.status).toBe('idle');
  });

  it('loads providers', async () => {
    mockGetProviders.mockResolvedValue({ providers: [] });
    renderHook(() => useDevices());
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });
    expect(mockGetProviders).toHaveBeenCalled();
  });
});
