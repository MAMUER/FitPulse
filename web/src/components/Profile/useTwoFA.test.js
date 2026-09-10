import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useTwoFA } from './useTwoFA';

const mockGet2FAStatus = vi.fn();
const mockSetup2FA = vi.fn();
const mockConfirm2FA = vi.fn();
const mockDisable2FA = vi.fn();

vi.mock('../../utils/api', () => ({
  get2FAStatus: (...args) => mockGet2FAStatus(...args),
  setup2FA: (...args) => mockSetup2FA(...args),
  confirm2FA: (...args) => mockConfirm2FA(...args),
  disable2FA: (...args) => mockDisable2FA(...args),
}));

describe('useTwoFA', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads 2FA status', async () => {
    mockGet2FAStatus.mockResolvedValue({ enabled: false });
    const { result } = renderHook(() => useTwoFA());
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });
    expect(result.current.enabled).toBe(false);
  });

  it('enables 2FA', async () => {
    mockGet2FAStatus.mockResolvedValue({ enabled: false });
    mockSetup2FA.mockResolvedValue({
      qr_code_base64: 'data:image/png;base64,abc',
      secret: 'SECRET123',
      backup_codes: ['code1', 'code2'],
    });
    const { result } = renderHook(() => useTwoFA());
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });
    await act(async () => {
      await result.current.handleEnable();
    });
    expect(result.current.qrCode).toBe('data:image/png;base64,abc');
    expect(result.current.backupCodes).toEqual(['code1', 'code2']);
  });

  it('validates setup code format', async () => {
    mockGet2FAStatus.mockResolvedValue({ enabled: false });
    const { result } = renderHook(() => useTwoFA());
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });
    await act(async () => {
      result.current.setSetupCode('12345');
      await result.current.handleConfirmSetup();
    });
    expect(result.current.setupError).toBe('Введите 6-значный код');
  });
});
