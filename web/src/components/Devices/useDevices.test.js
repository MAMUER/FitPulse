import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
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
    const { result } = renderHook(() => useDevices());
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });
    expect(mockGetProviders).toHaveBeenCalled();
  });
});
