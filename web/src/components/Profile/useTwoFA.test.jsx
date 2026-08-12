import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useTwoFA } from './useTwoFA';

vi.mock('../../utils/api');

describe('useTwoFA', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads 2fa status successfully', async () => {
    const { get2FAStatus } = await import('../../utils/api');
    get2FAStatus.mockResolvedValue({ enabled: true });

    const { result } = renderHook(() => useTwoFA());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.enabled).toBe(true);
    expect(result.current.status).toEqual({ enabled: true });
  });

  it('handles load 2fa status error', async () => {
    const { get2FAStatus } = await import('../../utils/api');
    get2FAStatus.mockRejectedValue(new Error('Network error'));

    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const { result } = renderHook(() => useTwoFA());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      'Failed to load 2FA status:',
      expect.any(Error)
    );
    consoleSpy.mockRestore();
  });

  it('enables 2fa and sets qr code', async () => {
    const { get2FAStatus, setup2FA } = await import('../../utils/api');
    get2FAStatus.mockResolvedValue({ enabled: false });
    setup2FA.mockResolvedValue({
      qr_code_base64: 'base64qr',
      secret: 'abcd1234efgh5678',
      backup_codes: ['code1', 'code2'],
    });

    const { result } = renderHook(() => useTwoFA());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    result.current.handleEnable();

    await waitFor(() => {
      expect(result.current.qrCode).toBe('base64qr');
    });

    expect(result.current.secret).toBe('abcd 1234 efgh 5678');
    expect(result.current.backupCodes).toEqual(['code1', 'code2']);
    expect(result.current.panelVisible).toBe(true);
    expect(result.current.setupError).toBe('');
  });

  it('handles enable 2fa error', async () => {
    const { get2FAStatus, setup2FA } = await import('../../utils/api');
    get2FAStatus.mockResolvedValue({ enabled: false });
    setup2FA.mockRejectedValue(new Error('Setup failed'));

    const { result } = renderHook(() => useTwoFA());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    result.current.handleEnable();

    await waitFor(() => {
      expect(result.current.setupError).toBe('Setup failed');
    });
  });

  it('validates setup code format', async () => {
    const { get2FAStatus } = await import('../../utils/api');
    get2FAStatus.mockResolvedValue({ enabled: false });

    const { result } = renderHook(() => useTwoFA());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      result.current.setSetupCode('12345');
      result.current.secret = 'abcd1234efgh5678';
      result.current.backupCodes = ['code1'];
    });

    await act(async () => {
      await result.current.handleConfirmSetup();
    });

    expect(result.current.setupError).toBe('Введите 6-значный код');
  });

  it('confirms 2fa setup successfully', async () => {
    const { get2FAStatus, setup2FA, confirm2FA } = await import(
      '../../utils/api'
    );
    get2FAStatus.mockResolvedValue({ enabled: false });
    setup2FA.mockResolvedValue({
      qr_code_base64: 'base64qr',
      secret: 'abcd1234efgh5678',
      backup_codes: ['code1'],
    });
    confirm2FA.mockResolvedValue({});

    const { result } = renderHook(() => useTwoFA());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    result.current.handleEnable();

    await waitFor(() => {
      expect(result.current.qrCode).toBe('base64qr');
    });

    await act(async () => {
      result.current.setSetupCode('123456');
    });

    await act(async () => {
      await result.current.handleConfirmSetup();
    });

    await waitFor(() => {
      expect(result.current.panelVisible).toBe(false);
    });

    expect(result.current.setupSuccess).toBe(
      '2FA включена. Сохраните резервные коды в надёжном месте.'
    );
    expect(confirm2FA).toHaveBeenCalledWith('123456', 'abcd1234efgh5678', [
      'code1',
    ]);
  });

  it('handles confirm 2fa setup error', async () => {
    const { get2FAStatus, setup2FA, confirm2FA } = await import(
      '../../utils/api'
    );
    get2FAStatus.mockResolvedValue({ enabled: false });
    setup2FA.mockResolvedValue({
      qr_code_base64: 'base64qr',
      secret: 'abcd1234efgh5678',
      backup_codes: ['code1'],
    });
    confirm2FA.mockRejectedValue(new Error('Invalid code'));

    const { result } = renderHook(() => useTwoFA());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    result.current.handleEnable();

    await waitFor(() => {
      expect(result.current.qrCode).toBe('base64qr');
    });

    await act(async () => {
      result.current.setSetupCode('123456');
    });

    await act(async () => {
      await result.current.handleConfirmSetup();
    });

    expect(result.current.setupError).toBe('Invalid code');
  });

  it('validates disable code', async () => {
    const { get2FAStatus } = await import('../../utils/api');
    get2FAStatus.mockResolvedValue({ enabled: true });

    const { result } = renderHook(() => useTwoFA());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      await result.current.handleDisable();
    });

    expect(result.current.disableError).toBe('Введите код 2FA');
  });

  it('disables 2fa successfully', async () => {
    const { get2FAStatus, disable2FA } = await import('../../utils/api');
    get2FAStatus.mockResolvedValue({ enabled: true });
    disable2FA.mockResolvedValue({});

    const { result } = renderHook(() => useTwoFA());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      result.current.setDisableCode('123456');
    });

    await act(async () => {
      await result.current.handleDisable();
    });

    expect(result.current.disableCode).toBe('');
    expect(disable2FA).toHaveBeenCalledWith('123456');
  });

  it('handles disable 2fa error', async () => {
    const { get2FAStatus, disable2FA } = await import('../../utils/api');
    get2FAStatus.mockResolvedValue({ enabled: true });
    disable2FA.mockRejectedValue(new Error('Invalid code'));

    const { result } = renderHook(() => useTwoFA());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      result.current.setDisableCode('123456');
    });

    await act(async () => {
      await result.current.handleDisable();
    });

    expect(result.current.disableError).toBe('Invalid code');
  });
});
