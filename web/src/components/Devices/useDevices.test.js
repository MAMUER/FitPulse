import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

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

import { useDevices } from './useDevices';

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
      window.dispatchEvent(
        new MessageEvent('message', {
          data: { type: 'OPEN_WEARABLES_CONNECTED' },
          origin: 'https://openwearables.com',
        })
      );
    });
    expect(mockGetProviders).toHaveBeenCalled();
  });
});
