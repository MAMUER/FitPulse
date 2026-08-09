import { render, screen } from '@testing-library/react';
import { act, fireEvent, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthProvider, useAuth } from '../../contexts/AuthContext';
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

  it('renders devices page', () => {
    renderDevices();
    expect(screen.getByText('Источники здоровья')).toBeInTheDocument();
  });

  it('shows connect button', () => {
    renderDevices();
    expect(screen.getByText('Подключить источники здоровья')).toBeInTheDocument();
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
        data: { type: 'OPEN_WEARABLES_ERROR', data: { message: 'Connection failed' } },
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
      expect(screen.getByText(/Не удалось загрузить виджет Open Wearables/)).toBeInTheDocument();
    });
  });

  it('shows empty providers message when none connected', async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      json: () => Promise.resolve({ providers: [] }),
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
      expect(screen.getByText('Нет подключённых источников')).toBeInTheDocument();
    });
  });

  it('displays providers after loading', async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      json: () =>
        Promise.resolve({
          providers: [
            { source: 'google', source_name: 'Google Fit', connected_at: '2024-01-01' },
          ],
        }),
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
});
