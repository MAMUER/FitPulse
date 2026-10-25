import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useApp } from '../contexts/AppContext';
import Integrations from './Integrations';

vi.mock('../contexts/AppContext', () => ({
  useApp: vi.fn(),
}));

describe('Integrations', () => {
  it('loads, disconnects and handles errors', async () => {
    const loadIntegrationProviders = vi.fn().mockResolvedValue([
      {
        source: 'withings',
        source_name: 'Withings',
        connected: true,
        connected_at: '2026-01-01T00:00:00Z',
      },
    ]);
    const disconnectIntegration = vi
      .fn()
      .mockResolvedValueOnce({})
      .mockRejectedValueOnce(new Error('fail'));
    const notify = vi.fn();
    useApp.mockReturnValue({
      notify,
      t: (ru, _en) => ru,
      loadIntegrationProviders,
      disconnectIntegration,
    });

    render(<Integrations />);
    await screen.findByText('Отключить');
    fireEvent.click(screen.getByText('Отключить'));
    await waitFor(() =>
      expect(disconnectIntegration).toHaveBeenCalledWith('withings')
    );
    await waitFor(() =>
      expect(notify).toHaveBeenCalledWith('Интеграция отключена')
    );
  });

  it('shows empty when no providers', async () => {
    const loadIntegrationProviders = vi.fn().mockResolvedValue([]);
    const disconnectIntegration = vi.fn();
    useApp.mockReturnValue({
      notify: vi.fn(),
      t: (ru, _en) => ru,
      loadIntegrationProviders,
      disconnectIntegration,
    });
    render(<Integrations />);
    await screen.findByText('Нет доступных интеграций');
    expect(screen.queryByText('Отключить')).not.toBeInTheDocument();
  });

  it('handles load error', async () => {
    const loadIntegrationProviders = vi.fn().mockRejectedValue(new Error('fail'));
    const disconnectIntegration = vi.fn();
    useApp.mockReturnValue({
      notify: vi.fn(),
      t: (ru, _en) => ru,
      loadIntegrationProviders,
      disconnectIntegration,
    });
    render(<Integrations />);
    await screen.findByText('Нет доступных интеграций');
    expect(loadIntegrationProviders).toHaveBeenCalled();
  });

  it('notifies on toggle failure', async () => {
    const loadIntegrationProviders = vi.fn().mockResolvedValue([
      { source: 'withings', source_name: 'Withings', connected: true, connected_at: null },
    ]);
    const disconnectIntegration = vi.fn().mockRejectedValue(new Error('fail'));
    const notify = vi.fn();
    useApp.mockReturnValue({
      notify,
      t: (ru, _en) => ru,
      loadIntegrationProviders,
      disconnectIntegration,
    });

    render(<Integrations />);
    await screen.findByText('Отключить');
    fireEvent.click(screen.getByText('Отключить'));
    await waitFor(() =>
      expect(notify).toHaveBeenCalledWith('Не удалось изменить интеграцию')
    );
  });
});
