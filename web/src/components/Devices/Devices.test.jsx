import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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

import Devices from './Devices';

describe('Devices', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuth.mockReturnValue({ token: 'test-token' });
  });

  it('renders devices view', () => {
    render(<Devices />);
    expect(screen.getByText('Источники здоровья')).toBeDefined();
  });

  it('shows no providers when empty', async () => {
    mockGetProviders.mockResolvedValue({ providers: [] });
    render(<Devices />);
    expect(await screen.findByText('Нет подключённых источников')).toBeDefined();
  });

  it('renders connect button', () => {
    render(<Devices />);
    expect(screen.getByText('Подключить источники здоровья')).toBeDefined();
  });

  it('disconnects a provider', async () => {
    const user = userEvent.setup();
    mockGetProviders.mockResolvedValue({
      providers: [
        {
          source: 'google',
          source_name: 'Google Fit',
          connected_at: '2024-01-01',
        },
      ],
    });
    mockDisconnectIntegration.mockResolvedValue({});
    window.confirm = vi.fn(() => true);
    render(<Devices />);
    window.dispatchEvent(
      new MessageEvent('message', {
        data: { type: 'OPEN_WEARABLES_CONNECTED' },
        origin: 'https://openwearables.com',
      })
    );
    expect(await screen.findByText('Google Fit')).toBeDefined();
    await user.click(screen.getByText('Отключить'));
    expect(mockDisconnectIntegration).toHaveBeenCalledWith('google');
  });
});
