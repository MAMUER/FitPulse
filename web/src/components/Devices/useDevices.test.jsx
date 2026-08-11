import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useDevices } from './useDevices';
import * as api from '../../utils/api';
import { useAuth } from '../../contexts/AuthContext';

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}));

const mockUseAuth = useAuth;

describe('useDevices', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.OpenWearablesWidget = undefined;
    document.body.innerHTML = '';
  });

  it('logs error when getProviders fails', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    mockUseAuth.mockReturnValue({ token: 'test-token' });
    vi.spyOn(api, 'getProviders').mockRejectedValueOnce(
      new Error('providers failed')
    );

    const { result } = renderHook(() => useDevices());

    await act(async () => {
      window.dispatchEvent(
        new MessageEvent('message', {
          data: { type: 'OPEN_WEARABLES_CONNECTED' },
          origin: 'https://openwearables.com',
        })
      );
    });

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(consoleErrorSpy).toHaveBeenCalledWith(
      'Failed to load providers:',
      expect.any(Error)
    );
    consoleErrorSpy.mockRestore();
  });

  it('calls widget init when already loaded', async () => {
    mockUseAuth.mockReturnValue({ token: 'test-token' });
    window.OpenWearablesWidget = {
      init: vi.fn((opts) => {
        opts.onSuccess?.();
      }),
    };

    const { result } = renderHook(() => useDevices());

    await act(async () => {
      result.current.handleConnect();
    });

    expect(window.OpenWearablesWidget.init).toHaveBeenCalledTimes(1);
    const initCall = window.OpenWearablesWidget.init.mock.calls[0][0];
    expect(initCall.appId).toBeDefined();
    expect(initCall.userId).toBe('anonymous');
  });

  it('sets error state when widget reports error', async () => {
    mockUseAuth.mockReturnValue({ token: 'test-token' });
    window.OpenWearablesWidget = {
      init: vi.fn((opts) => {
        opts.onError?.(new Error('widget failed'));
      }),
    };

    const { result } = renderHook(() => useDevices());

    await act(async () => {
      result.current.handleConnect();
    });

    expect(result.current.status).toBe('error');
    expect(result.current.error).toBe('widget failed');
  });

  it('sets idle state when widget closes', async () => {
    mockUseAuth.mockReturnValue({ token: 'test-token' });
    window.OpenWearablesWidget = {
      init: vi.fn((opts) => {
        opts.onClose?.();
      }),
    };

    const { result } = renderHook(() => useDevices());

    await act(async () => {
      result.current.handleConnect();
    });

    expect(result.current.status).toBe('idle');
  });

  it('returns anonymous for invalid token', async () => {
    mockUseAuth.mockReturnValue({ token: 'invalid-token' });
    window.OpenWearablesWidget = {
      init: vi.fn(),
    };

    const { result } = renderHook(() => useDevices());

    await act(async () => {
      result.current.handleConnect();
    });

    const initCall = window.OpenWearablesWidget.init.mock.calls[0][0];
    expect(initCall.userId).toBe('anonymous');
  });

  it('shows error when widget does not load after retry', async () => {
    mockUseAuth.mockReturnValue({ token: 'test-token' });
    window.OpenWearablesWidget = undefined;

    vi.useFakeTimers();
    const { result } = renderHook(() => useDevices());

    await act(async () => {
      result.current.handleConnect();
    });

    await act(async () => {
      vi.advanceTimersByTime(1000);
    });

    expect(result.current.status).toBe('error');
    expect(result.current.error).toBe(
      'Виджет не загрузился. Попробуйте позже.'
    );
    vi.useRealTimers();
  });
});
