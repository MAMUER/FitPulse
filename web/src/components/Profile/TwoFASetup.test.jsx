import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../../utils/api';
import TwoFASetup from './TwoFASetup';

vi.mock('../../utils/api', () => ({
  get2FAStatus: vi.fn(),
  setup2FA: vi.fn(),
  confirm2FA: vi.fn(),
  disable2FA: vi.fn(),
}));

describe('TwoFASetup', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const user = userEvent.setup();

  it('shows loading state initially', () => {
    api.get2FAStatus.mockImplementation(() => new Promise(() => {}));
    render(<TwoFASetup />);
    expect(screen.getByText('Загрузка статуса 2FA...')).toBeInTheDocument();
  });

  it('renders component with useTwoFA hook', async () => {
    api.get2FAStatus.mockResolvedValue({ enabled: false });
    render(<TwoFASetup />);
    await waitFor(() => {
      expect(screen.getByText('Не включена')).toBeInTheDocument();
    });
    expect(screen.getByText('Включить 2FA')).toBeInTheDocument();
  });

  it('shows disabled state when 2FA is not enabled', async () => {
    api.get2FAStatus.mockResolvedValue({ enabled: false });
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText('Не включена')).toBeInTheDocument();
    });

    expect(screen.getByText('Включить 2FA')).toBeInTheDocument();
    expect(screen.queryByText('Отключить 2FA')).not.toBeInTheDocument();
  });

  it('shows enabled state when 2FA is enabled', async () => {
    api.get2FAStatus.mockResolvedValue({ enabled: true });
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText(/Включена/)).toBeInTheDocument();
    });

    expect(screen.getAllByText('Отключить 2FA').length).toBeGreaterThanOrEqual(
      1
    );
    expect(screen.queryByText('Включить 2FA')).not.toBeInTheDocument();
  });

  it('allows clicking the disable 2FA button when enabled', async () => {
    api.get2FAStatus.mockResolvedValue({ enabled: true });
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText(/Включена/)).toBeInTheDocument();
    });

    const topButton = document.querySelector('.twofa-section > div > button');
    await user.click(topButton);

    expect(topButton).toBeTruthy();
  });

  it('loads setup panel on enable click', async () => {
    api.get2FAStatus.mockResolvedValue({ enabled: false });
    api.setup2FA.mockResolvedValueOnce({
      qr_code_base64: 'data:image/png;base64,abc',
      secret: 'JBSWY3DPEHPK3PXP',
      backup_codes: ['123456', '789012'],
    });
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText('Включить 2FA')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Включить 2FA'));

    await waitFor(() => {
      expect(
        screen.getByText('Подтвердить и включить 2FA')
      ).toBeInTheDocument();
    });
    expect(screen.getByPlaceholderText('6-значный код')).toBeInTheDocument();
  });

  it('shows setup error on failure', async () => {
    api.get2FAStatus.mockResolvedValue({ enabled: false });
    api.setup2FA.mockRejectedValueOnce(new Error('setup failed'));
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText('Включить 2FA')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Включить 2FA'));

    await waitFor(() => {
      expect(screen.getByText('setup failed')).toBeInTheDocument();
    });
  });

  it('validates 6-digit code on confirm', async () => {
    api.get2FAStatus.mockResolvedValue({ enabled: false });
    api.setup2FA.mockResolvedValueOnce({
      qr_code_base64: 'data:image/png;base64,abc',
      secret: 'JBSWY3DPEHPK3PXP',
      backup_codes: ['123456', '789012'],
    });
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText('Включить 2FA')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Включить 2FA'));

    await waitFor(() => {
      expect(screen.getByPlaceholderText('6-значный код')).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText('6-значный код');
    await user.type(input, '123');

    await user.click(screen.getByText('Подтвердить и включить 2FA'));

    await waitFor(() => {
      expect(screen.getByText('Введите 6-значный код')).toBeInTheDocument();
    });
  });

  it('confirms 2FA successfully', async () => {
    api.get2FAStatus.mockResolvedValueOnce({ enabled: false });
    api.setup2FA.mockResolvedValueOnce({
      qr_code_base64: 'data:image/png;base64,abc',
      secret: 'JBSWY3DPEHPK3PXP',
      backup_codes: ['123456', '789012'],
    });
    api.confirm2FA.mockResolvedValueOnce(undefined);
    api.get2FAStatus.mockResolvedValueOnce({
      enabled: true,
      backup_codes_remaining: 5,
    });
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText('Включить 2FA')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Включить 2FA'));

    await waitFor(() => {
      expect(screen.getByPlaceholderText('6-значный код')).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText('6-значный код');
    await user.type(input, '123456');

    await user.click(screen.getByText('Подтвердить и включить 2FA'));

    await waitFor(() => {
      expect(screen.getByText(/Включена/)).toBeInTheDocument();
    });
    expect(screen.getAllByText('Отключить 2FA').length).toBeGreaterThanOrEqual(
      1
    );
  });

  it('shows disable error for empty code', async () => {
    api.get2FAStatus.mockResolvedValue({
      enabled: true,
      backup_codes_remaining: 5,
    });
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText(/Включена/)).toBeInTheDocument();
    });

    const disablePanel = document.getElementById('disable2FAPanel');
    await user.click(disablePanel.querySelector('button'));

    await waitFor(() => {
      expect(screen.getByText('Введите код 2FA')).toBeInTheDocument();
    });
  });

  it('disables 2FA successfully', async () => {
    let get2FACallCount = 0;
    api.get2FAStatus.mockImplementation(() => {
      get2FACallCount++;
      if (get2FACallCount === 1)
        return Promise.resolve({ enabled: true, backup_codes_remaining: 5 });
      return Promise.resolve({ enabled: false });
    });
    api.disable2FA.mockResolvedValueOnce(undefined);
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText(/Включена/)).toBeInTheDocument();
    });

    const disablePanel = document.getElementById('disable2FAPanel');
    const disableInput = disablePanel.querySelector('input');
    await user.type(disableInput, '123456');

    await user.click(disablePanel.querySelector('button'));

    await waitFor(() => {
      expect(screen.getByText('Не включена')).toBeInTheDocument();
    });
  });

  it('shows error when confirm 2FA fails', async () => {
    api.get2FAStatus.mockResolvedValue({ enabled: false });
    api.setup2FA.mockResolvedValueOnce({
      qr_code_base64: 'data:image/png;base64,abc',
      secret: 'JBSWY3DPEHPK3PXP',
      backup_codes: ['123456', '789012'],
    });
    api.confirm2FA.mockRejectedValueOnce(new Error('invalid code'));
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText('Включить 2FA')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Включить 2FA'));

    await waitFor(() => {
      expect(screen.getByPlaceholderText('6-значный код')).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText('6-значный код');
    await user.type(input, '123456');

    await user.click(screen.getByText('Подтвердить и включить 2FA'));

    await waitFor(() => {
      expect(screen.getByText('invalid code')).toBeInTheDocument();
    });
  });

  it('shows error when disable 2FA fails', async () => {
    api.get2FAStatus.mockResolvedValue({
      enabled: true,
      backup_codes_remaining: 5,
    });
    api.disable2FA.mockRejectedValueOnce(new Error('disable failed'));
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText(/Включена/)).toBeInTheDocument();
    });

    const disablePanel = document.getElementById('disable2FAPanel');
    const disableInput = disablePanel.querySelector('input');
    await user.type(disableInput, '123456');

    await user.click(disablePanel.querySelector('button'));

    await waitFor(() => {
      expect(screen.getByText('disable failed')).toBeInTheDocument();
    });
  });

  it('handles load status error gracefully', async () => {
    api.get2FAStatus.mockRejectedValueOnce(new Error('load failed'));
    render(<TwoFASetup />);

    await waitFor(() => {
      expect(screen.getByText('Не включена')).toBeInTheDocument();
    });
  });
});
