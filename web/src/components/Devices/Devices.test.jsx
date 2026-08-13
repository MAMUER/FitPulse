import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
import * as api from '../../utils/api';
import Devices from './Devices';

vi.mock('../../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../../contexts/AuthContext');
  return {
    ...actual,
    useAuth: vi.fn(),
  };
});

const mockUseAuth = useAuth;

const renderDevices = (authOverrides = {}) => {
  mockUseAuth.mockReturnValue({
    token: 'test-token',
    ...authOverrides,
  });

  return render(
    <AuthProvider>
      <Devices />
    </AuthProvider>
  );
};

describe('Devices', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.OpenWearablesWidget = undefined;
    document.body.innerHTML = '';
  });

  const user = userEvent.setup();

  it('renders devices page with default state', () => {
    renderDevices();
    expect(screen.getByText('Источники здоровья')).toBeInTheDocument();
    expect(screen.getByText('Подключить источники здоровья')).toBeInTheDocument();
    expect(screen.getByText('Нет подключённых источников')).toBeInTheDocument();
  });

  it('shows connect button', () => {
    renderDevices();
    expect(
      screen.getByText('Подключить источники здоровья')
    ).toBeInTheDocument();
  });

  it('loads external script on mount when token exists', () => {
    renderDevices();
    const script = document.getElementById('open-wearables-widget-script');
    expect(script).toBeInTheDocument();
  });

  it('does not load script when no token', () => {
    renderDevices({ token: null });
    const script = document.getElementById('open-wearables-widget-script');
    expect(script).not.toBeInTheDocument();
  });

  it('handles widget connected message', async () => {
    vi.spyOn(api, 'getProviders').mockResolvedValueOnce({ providers: [] });
    renderDevices();
    window.OpenWearablesWidget = {
      init: vi.fn(),
    };

    await act(async () => {
      const event = new MessageEvent('message', {
        data: { type: 'OPEN_WEARABLES_CONNECTED' },
        origin: 'https://openwearables.com',
      });
      window.dispatchEvent(event);
    });

    await waitFor(() => {
      expect(screen.getByText(/Успешно подключено/)).toBeInTheDocument();
    });
  });

  it('handles widget error message', async () => {
    renderDevices();

    await act(async () => {
      const event = new MessageEvent('message', {
        data: {
          type: 'OPEN_WEARABLES_ERROR',
          data: { message: 'Connection failed' },
        },
        origin: 'https://openwearables.com',
      });
      window.dispatchEvent(event);
    });

    await waitFor(() => {
      expect(screen.getByText(/Connection failed/)).toBeInTheDocument();
    });
  });

  it('ignores messages from disallowed origins', async () => {
    renderDevices();

    await act(async () => {
      const event = new MessageEvent('message', {
        data: { type: 'OPEN_WEARABLES_CONNECTED' },
        origin: 'https://evil.com',
      });
      window.dispatchEvent(event);
    });

    expect(screen.queryByText(/Успешно подключено/)).not.toBeInTheDocument();
  });

  it('shows error when widget fails to load', async () => {
    renderDevices();
    const script = document.getElementById('open-wearables-widget-script');
    if (script) {
      await act(async () => {
        script.onerror();
      });
    }

    await waitFor(() => {
      expect(
        screen.getByText(/Не удалось загрузить виджет Open Wearables/)
      ).toBeInTheDocument();
    });
  });

  it('shows empty providers message when none connected', async () => {
    vi.spyOn(api, 'getProviders').mockResolvedValueOnce({ providers: [] });
    renderDevices();

    await act(async () => {
      const event = new MessageEvent('message', {
        data: { type: 'OPEN_WEARABLES_CONNECTED' },
        origin: 'https://openwearables.com',
      });
      window.dispatchEvent(event);
    });

    await waitFor(() => {
      expect(
        screen.getByText('Нет подключённых источников')
      ).toBeInTheDocument();
    });
  });

  it('displays provider source as fallback when source_name is missing', async () => {
    vi.spyOn(api, 'getProviders').mockResolvedValueOnce({
      providers: [
        {
          source: 'google',
          connected_at: '2024-01-01',
        },
      ],
    });
    renderDevices();

    await act(async () => {
      const event = new MessageEvent('message', {
        data: { type: 'OPEN_WEARABLES_CONNECTED' },
        origin: 'https://openwearables.com',
      });
      window.dispatchEvent(event);
    });

    await waitFor(() => {
      expect(screen.getByText('google')).toBeInTheDocument();
    });
  });

  it('displays providers after loading', async () => {
    vi.spyOn(api, 'getProviders').mockResolvedValueOnce({
      providers: [
        {
          source: 'google',
          source_name: 'Google Fit',
          connected_at: '2024-01-01',
        },
      ],
    });
    renderDevices();

    await act(async () => {
      const event = new MessageEvent('message', {
        data: { type: 'OPEN_WEARABLES_CONNECTED' },
        origin: 'https://openwearables.com',
      });
      window.dispatchEvent(event);
    });

    await waitFor(() => {
      expect(screen.getByText('Google Fit')).toBeInTheDocument();
    });
  });

  it('disconnects provider on click', async () => {
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true);
    vi.spyOn(api, 'getProviders')
      .mockResolvedValueOnce({
        providers: [
          {
            source: 'google',
            source_name: 'Google Fit',
            connected_at: '2024-01-01',
          },
        ],
      })
      .mockResolvedValueOnce({ providers: [] });
    vi.spyOn(api, 'disconnectIntegration').mockResolvedValueOnce(undefined);
    renderDevices();

    await act(async () => {
      const event = new MessageEvent('message', {
        data: { type: 'OPEN_WEARABLES_CONNECTED' },
        origin: 'https://openwearables.com',
      });
      window.dispatchEvent(event);
    });

    await waitFor(() => {
      expect(screen.getByText('Google Fit')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Отключить'));

    await waitFor(() => {
      expect(screen.queryByText('Google Fit')).not.toBeInTheDocument();
    });
  });

  it('shows error when disconnect fails', async () => {
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true);
    const alertMock = vi.spyOn(window, 'alert').mockImplementation(() => {});
    vi.spyOn(api, 'getProviders').mockResolvedValueOnce({
      providers: [
        {
          source: 'google',
          source_name: 'Google Fit',
          connected_at: '2024-01-01',
        },
      ],
    });
    vi.spyOn(api, 'disconnectIntegration').mockRejectedValueOnce(
      new Error('Network error')
    );
    renderDevices();

    await act(async () => {
      const event = new MessageEvent('message', {
        data: { type: 'OPEN_WEARABLES_CONNECTED' },
        origin: 'https://openwearables.com',
      });
      window.dispatchEvent(event);
    });

    await waitFor(() => {
      expect(screen.getByText('Google Fit')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Отключить'));

    expect(alertMock).toHaveBeenCalledWith('Ошибка отключения: Network error');
  });

  it('handles widget closed message', async () => {
    renderDevices();

    await act(async () => {
      const event = new MessageEvent('message', {
        data: { type: 'OPEN_WEARABLES_CLOSED' },
        origin: 'https://openwearables.com',
      });
      window.dispatchEvent(event);
    });

    await waitFor(() => {
      expect(
        screen.getByText('Подключить источники здоровья')
      ).toBeInTheDocument();
    });
  });

  it('shows loading state when connecting', async () => {
    renderDevices();
    window.OpenWearablesWidget = undefined;

    const connectButton = screen.getByText('Подключить источники здоровья');
    await user.click(connectButton);

    expect(screen.getByText('Подключение...')).toBeInTheDocument();
    expect(connectButton).toBeDisabled();
  });
});
