import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthProvider } from '../../contexts/AuthContext';
import TwoFASetup from './TwoFASetup';

const mockUseTwoFA = vi.fn();
vi.mock('./useTwoFA', () => ({
  useTwoFA: () => mockUseTwoFA(),
}));

describe('TwoFASetup', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state', () => {
    mockUseTwoFA.mockReturnValue({
      loading: true,
      enabled: false,
      status: null,
      qrCode: '',
      secret: '',
      backupCodes: [],
      setupCode: '',
      setupError: '',
      setupSuccess: '',
      disableCode: '',
      disableError: '',
      panelVisible: false,
      setSetupCode: vi.fn(),
      setDisableCode: vi.fn(),
      setPanelVisible: vi.fn(),
      handleEnable: vi.fn(),
      handleConfirmSetup: vi.fn(),
      handleDisable: vi.fn(),
    });
    render(
      <AuthProvider>
        <TwoFASetup />
      </AuthProvider>
    );
    expect(screen.getByText('Загрузка статуса 2FA...')).toBeDefined();
  });

  it('renders enable 2FA button when disabled', () => {
    mockUseTwoFA.mockReturnValue({
      loading: false,
      enabled: false,
      status: { enabled: false },
      qrCode: '',
      secret: '',
      backupCodes: [],
      setupCode: '',
      setupError: '',
      setupSuccess: '',
      disableCode: '',
      disableError: '',
      panelVisible: false,
      setSetupCode: vi.fn(),
      setDisableCode: vi.fn(),
      setPanelVisible: vi.fn(),
      handleEnable: vi.fn(),
      handleConfirmSetup: vi.fn(),
      handleDisable: vi.fn(),
    });
    render(
      <AuthProvider>
        <TwoFASetup />
      </AuthProvider>
    );
    expect(screen.getByText('Включить 2FA')).toBeDefined();
  });

  it('renders disable 2FA button when enabled', () => {
    mockUseTwoFA.mockReturnValue({
      loading: false,
      enabled: true,
      status: { enabled: true, backup_codes_remaining: 8 },
      qrCode: '',
      secret: '',
      backupCodes: [],
      setupCode: '',
      setupError: '',
      setupSuccess: '',
      disableCode: '',
      disableError: '',
      panelVisible: false,
      setSetupCode: vi.fn(),
      setDisableCode: vi.fn(),
      setPanelVisible: vi.fn(),
      handleEnable: vi.fn(),
      handleConfirmSetup: vi.fn(),
      handleDisable: vi.fn(),
    });
    render(
      <AuthProvider>
        <TwoFASetup />
      </AuthProvider>
    );
    const buttons = screen.getAllByText('Отключить 2FA');
    expect(buttons.length).toBeGreaterThanOrEqual(1);
  });
});
