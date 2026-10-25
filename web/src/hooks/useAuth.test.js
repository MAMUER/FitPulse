import { act, renderHook } from '@testing-library/react';
import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from 'vitest';
import * as api from '../services/api';
import { useAuth } from './useAuth';

vi.mock('../services/api', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  register: vi.fn(),
  submitResetEmail: vi.fn(),
  submitResetCode: vi.fn(),
  submitNewPassword: vi.fn(),
  socialLogin: vi.fn(),
  confirmEmail: vi.fn(),
  setup2FA: vi.fn(),
  confirm2FA: vi.fn(),
  verify2FA: vi.fn(),
  disable2FA: vi.fn(),
  load2FAStatus: vi.fn(),
}));

const INPUT_IDS = [
  'loginEmail',
  'loginPassword',
  'regEmail',
  'regPassword',
  'regConfirm',
  'regConsent',
  'resetEmail',
  'resetCode',
  'newPassword',
  'newPasswordConfirm',
];

const state = { profile: { name: 'Имя' }, resetEmail: 'user@example.com' };

let update;
let notify;
let result;

function setValue(id, value) {
  const el = document.getElementById(id);
  if (el) el.value = value;
}

function setChecked(id, checked) {
  const el = document.getElementById(id);
  if (el) el.checked = checked;
}

const originalLocation = window.location;

beforeAll(() => {
  Object.defineProperty(window, 'location', {
    configurable: true,
    writable: true,
    value: { href: '' },
  });
});

afterAll(() => {
  Object.defineProperty(window, 'location', {
    configurable: true,
    writable: true,
    value: originalLocation,
  });
});

beforeEach(() => {
  vi.resetAllMocks();
  window.location.href = '';
  update = vi.fn();
  notify = vi.fn();
  for (const id of INPUT_IDS) {
    const el = document.createElement('input');
    el.id = id;
    document.body.appendChild(el);
  }
  result = renderHook(() => useAuth({ state, update, notify })).result;
});

afterEach(() => {
  for (const id of INPUT_IDS) {
    document.getElementById(id)?.remove();
  }
});

describe('useAuth', () => {
  it('returns all auth callbacks', () => {
    expect(Object.keys(result.current)).toEqual([
      'doLogin',
      'logout',
      'doRegister',
      'forgotPassword',
      'submitResetEmail',
      'submitResetCode',
      'submitNewPassword',
      'continueAsGuest',
      'socialLogin',
      'confirmEmail',
      'setup2FA',
      'confirm2FA',
      'verify2FA',
      'disable2FA',
      'load2FAStatus',
    ]);
  });

  it('doLogin notifies when fields are empty', async () => {
    await act(async () => {
      await result.current.doLogin();
    });
    expect(notify).toHaveBeenCalledWith('Неверный email или пароль');
    expect(api.login).not.toHaveBeenCalled();
  });

  it('doLogin logs in and navigates home', async () => {
    setValue('loginEmail', ' user@example.com ');
    setValue('loginPassword', ' secret ');
    api.login.mockResolvedValue({ status: 'ok' });
    await act(async () => {
      await result.current.doLogin();
    });
    expect(api.login).toHaveBeenCalledWith('user@example.com', 'secret');
    expect(update).toHaveBeenCalledWith({
      profile: {
        ...state.profile,
        email: 'user@example.com',
        password: 'secret',
      },
      registered: true,
      guest: false,
      screen: 'home',
    });
    expect(window.location.href).toBe('/home');
  });

  it('doLogin notifies when credentials are rejected', async () => {
    setValue('loginEmail', 'user@example.com');
    setValue('loginPassword', 'secret');
    api.login.mockResolvedValue({ status: 'error' });
    await act(async () => {
      await result.current.doLogin();
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Неверный email или пароль');
  });

  it('doLogin notifies on api failure', async () => {
    setValue('loginEmail', 'user@example.com');
    setValue('loginPassword', 'secret');
    api.login.mockRejectedValue(new Error('network'));
    await act(async () => {
      await result.current.doLogin();
    });
    expect(notify).toHaveBeenCalledWith('Ошибка входа');
  });

  it('logout resets state and navigates to root', async () => {
    await act(async () => {
      await result.current.logout();
    });
    expect(api.logout).toHaveBeenCalled();
    expect(update).toHaveBeenCalledWith({
      registered: false,
      guest: false,
      screen: 'login',
    });
    expect(window.location.href).toBe('/');
  });

  it('doRegister notifies when consent is missing', () => {
    setValue('regEmail', 'user@example.com');
    setValue('regPassword', 'secret123');
    setValue('regConfirm', 'secret123');
    result.current.doRegister();
    expect(api.register).toHaveBeenCalledWith(
      'user@example.com',
      'secret123',
      'secret123'
    );
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Подтвердите согласие с документами.');
  });

  it('doRegister stores registration data on success', () => {
    setValue('regEmail', 'user@example.com');
    setValue('regPassword', 'secret123');
    setValue('regConfirm', 'secret123');
    setChecked('regConsent', true);
    result.current.doRegister();
    expect(update).toHaveBeenCalledWith({
      guest: false,
      registrationData: {
        email: 'user@example.com',
        password: 'secret123',
        confirm: 'secret123',
        code: '',
      },
      profile: { ...state.profile, email: 'user@example.com' },
      registered: true,
      screen: 'login',
    });
    expect(notify).toHaveBeenCalledWith(
      'Регистрация успешна. Проверьте email для подтверждения.'
    );
  });

  it('doRegister surfaces api validation errors', () => {
    setValue('regEmail', 'user@example.com');
    setValue('regPassword', 'secret123');
    setValue('regConfirm', 'secret123');
    api.register.mockImplementation(() => {
      throw new Error('Пароли не совпадают');
    });
    result.current.doRegister();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Пароли не совпадают');
  });

  it('forgotPassword moves to reset step', async () => {
    await act(async () => {
      await result.current.forgotPassword();
    });
    expect(update).toHaveBeenCalledWith({ screen: 'reset', resetStep: 0 });
  });

  it('submitResetEmail rejects invalid email', async () => {
    setValue('resetEmail', 'not-an-email');
    await act(async () => {
      await result.current.submitResetEmail();
    });
    expect(api.submitResetEmail).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Введите корректный email');
  });

  it('submitResetEmail sends code and advances step', async () => {
    setValue('resetEmail', ' user@example.com ');
    api.submitResetEmail.mockResolvedValue({});
    await act(async () => {
      await result.current.submitResetEmail();
    });
    expect(api.submitResetEmail).toHaveBeenCalledWith('user@example.com');
    expect(update).toHaveBeenCalledWith({
      resetEmail: 'user@example.com',
      resetStep: 1,
    });
    expect(notify).toHaveBeenCalledWith('Код отправлен на email');
  });

  it('submitResetEmail notifies on failure', async () => {
    setValue('resetEmail', 'user@example.com');
    api.submitResetEmail.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.submitResetEmail();
    });
    expect(notify).toHaveBeenCalledWith('Ошибка отправки кода');
  });

  it('submitResetCode surfaces api validation errors', () => {
    api.submitResetCode.mockImplementation(() => {
      throw new Error('Введите код');
    });
    result.current.submitResetCode();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Введите код');
  });

  it('submitResetCode advances to password step', () => {
    setValue('resetCode', '123456');
    result.current.submitResetCode();
    expect(api.submitResetCode).toHaveBeenCalledWith('123456');
    expect(update).toHaveBeenCalledWith({ resetStep: 2 });
  });

  it('submitNewPassword validates length', async () => {
    setValue('newPassword', '123');
    setValue('newPasswordConfirm', '123');
    setValue('resetCode', '123456');
    await act(async () => {
      await result.current.submitNewPassword();
    });
    expect(api.submitNewPassword).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith(
      'Пароль должен содержать минимум 6 символов'
    );
  });

  it('submitNewPassword uses empty strings when fields are missing', async () => {
    for (const id of ['newPassword', 'newPasswordConfirm']) {
      document.getElementById(id)?.remove();
    }
    await act(async () => {
      await result.current.submitNewPassword();
    });
    expect(api.submitNewPassword).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith(
      'Пароль должен содержать минимум 6 символов'
    );
  });

  it('submitNewPassword validates confirmation', async () => {
    setValue('newPassword', 'secret123');
    setValue('newPasswordConfirm', 'other123');
    setValue('resetCode', '123456');
    await act(async () => {
      await result.current.submitNewPassword();
    });
    expect(notify).toHaveBeenCalledWith('Пароли не совпадают');
  });

  it('submitNewPassword saves new password', async () => {
    setValue('newPassword', 'secret123');
    setValue('newPasswordConfirm', 'secret123');
    setValue('resetCode', '123456');
    api.submitNewPassword.mockResolvedValue({});
    await act(async () => {
      await result.current.submitNewPassword();
    });
    expect(api.submitNewPassword).toHaveBeenCalledWith(
      'user@example.com',
      '123456',
      'secret123'
    );
    expect(update).toHaveBeenCalledWith({
      profile: { ...state.profile, password: 'secret123' },
      screen: 'login',
    });
    expect(notify).toHaveBeenCalledWith('Пароль изменён. Теперь можно войти.');
  });

  it('submitNewPassword notifies on failure', async () => {
    setValue('newPassword', 'secret123');
    setValue('newPasswordConfirm', 'secret123');
    setValue('resetCode', '123456');
    api.submitNewPassword.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.submitNewPassword();
    });
    expect(notify).toHaveBeenCalledWith('Ошибка сброса пароля');
  });

  it('continueAsGuest enters guest mode', () => {
    result.current.continueAsGuest();
    expect(update).toHaveBeenCalledWith({
      guest: true,
      registered: false,
      screen: 'home',
    });
    expect(window.location.href).toBe('/home');
  });

  it('socialLogin redirects for supported provider', () => {
    api.socialLogin.mockReturnValue({ redirect: '/api/v1/auth/yandex' });
    result.current.socialLogin('yandex');
    expect(api.socialLogin).toHaveBeenCalledWith('yandex');
    expect(window.location.href).toBe('/api/v1/auth/yandex');
  });

  it('socialLogin notifies for unsupported provider', () => {
    api.socialLogin.mockImplementation(() => {
      throw new Error('Интеграция google не поддерживается');
    });
    result.current.socialLogin('google');
    expect(notify).toHaveBeenCalledWith('Интеграция google не поддерживается');
  });

  it('socialLogin does nothing without redirect', () => {
    api.socialLogin.mockReturnValue({});
    result.current.socialLogin('yandex');
    expect(window.location.href).toBe('');
  });

  it('confirmEmail notifies on success', async () => {
    api.confirmEmail.mockResolvedValue({ status: 'ok' });
    await act(async () => {
      await result.current.confirmEmail('token');
    });
    expect(api.confirmEmail).toHaveBeenCalledWith('token');
    expect(notify).toHaveBeenCalledWith('Email подтверждён');
  });

  it('confirmEmail notifies with backend message', async () => {
    api.confirmEmail.mockResolvedValue({
      status: 'bad',
      message: 'Ссылка истекла',
    });
    await act(async () => {
      await result.current.confirmEmail('token');
    });
    expect(notify).toHaveBeenCalledWith('Ссылка истекла');
  });

  it('confirmEmail falls back to generic message', async () => {
    api.confirmEmail.mockResolvedValue({ status: 'bad' });
    await act(async () => {
      await result.current.confirmEmail('token');
    });
    expect(notify).toHaveBeenCalledWith('Ошибка подтверждения');
  });

  it('confirmEmail notifies on failure', async () => {
    api.confirmEmail.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.confirmEmail('token');
    });
    expect(notify).toHaveBeenCalledWith('Ошибка подтверждения email');
  });

  it('setup2FA stores qr data and opens setup screen', async () => {
    api.setup2FA.mockResolvedValue({
      qr_code_url: 'https://qr',
      qr_code_base64: 'base64',
      secret: 'SECRET',
      backup_codes: ['a', 'b'],
    });
    await act(async () => {
      await result.current.setup2FA();
    });
    expect(update).toHaveBeenCalledWith({
      twoFactorSetup: {
        qrCodeUrl: 'https://qr',
        qrCodeBase64: 'base64',
        secret: 'SECRET',
        backupCodes: ['a', 'b'],
      },
      screen: 'twofa-setup',
    });
  });

  it('setup2FA notifies on failure', async () => {
    api.setup2FA.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.setup2FA();
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Не удалось настроить 2FA');
  });

  it('setup2FA ignores missing 2fa fields', async () => {
    api.setup2FA.mockResolvedValue({
      qr_code_url: null,
      qr_code_base64: undefined,
    });
    await act(async () => {
      await result.current.setup2FA();
    });
    expect(update).toHaveBeenCalledWith({
      twoFactorSetup: {
        qrCodeUrl: '',
        qrCodeBase64: '',
        secret: '',
        backupCodes: [],
      },
      screen: 'twofa-setup',
    });
  });

  it('setup2FA handles null result from api', async () => {
    api.setup2FA.mockResolvedValue(null);
    await act(async () => {
      await result.current.setup2FA();
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).not.toHaveBeenCalled();
  });

  it('confirm2FA notifies on success', async () => {
    api.confirm2FA.mockResolvedValue({ success: true });
    await act(async () => {
      await result.current.confirm2FA('123456', 'SECRET', ['backup']);
    });
    expect(api.confirm2FA).toHaveBeenCalledWith('123456', 'SECRET', ['backup']);
    expect(update).toHaveBeenCalledWith({
      twoFactorSetup: null,
      screen: 'profile',
    });
    expect(notify).toHaveBeenCalledWith('2FA успешно настроена');
  });

  it('confirm2FA notifies with backend message', async () => {
    api.confirm2FA.mockResolvedValue({
      success: false,
      message: 'Неверный код',
    });
    await act(async () => {
      await result.current.confirm2FA('000000');
    });
    expect(notify).toHaveBeenCalledWith('Неверный код');
  });

  it('confirm2FA falls back to generic message', async () => {
    api.confirm2FA.mockResolvedValue({ success: false });
    await act(async () => {
      await result.current.confirm2FA('000000');
    });
    expect(notify).toHaveBeenCalledWith('Ошибка подтверждения 2FA');
  });

  it('confirm2FA notifies on failure', async () => {
    api.confirm2FA.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.confirm2FA('000000');
    });
    expect(notify).toHaveBeenCalledWith('Не удалось подтвердить 2FA');
  });

  it('verify2FA completes login on success', async () => {
    api.verify2FA.mockResolvedValue({ status: 'ok' });
    await act(async () => {
      await result.current.verify2FA('temp', '123456', true);
    });
    expect(api.verify2FA).toHaveBeenCalledWith('temp', '123456', true);
    expect(update).toHaveBeenCalledWith({
      registered: true,
      guest: false,
      screen: 'home',
      twoFactorTempToken: null,
    });
  });

  it('verify2FA notifies on invalid code', async () => {
    api.verify2FA.mockResolvedValue({ status: 'bad' });
    await act(async () => {
      await result.current.verify2FA('temp', '000000');
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Неверный код 2FA');
  });

  it('verify2FA notifies on failure', async () => {
    api.verify2FA.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.verify2FA('temp', '000000');
    });
    expect(notify).toHaveBeenCalledWith('Ошибка проверки 2FA');
  });

  it('disable2FA updates profile on success', async () => {
    api.disable2FA.mockResolvedValue({ success: true });
    await act(async () => {
      await result.current.disable2FA('123456');
    });
    expect(api.disable2FA).toHaveBeenCalledWith('123456');
    expect(update).toHaveBeenCalledWith({
      profile: { ...state.profile, twoFactor: false },
    });
    expect(notify).toHaveBeenCalledWith('2FA отключена');
  });

  it('disable2FA notifies with backend message', async () => {
    api.disable2FA.mockResolvedValue({
      success: false,
      message: 'Неверный код',
    });
    await act(async () => {
      await result.current.disable2FA('000000');
    });
    expect(notify).toHaveBeenCalledWith('Неверный код');
  });

  it('disable2FA falls back to generic message', async () => {
    api.disable2FA.mockResolvedValue({ success: false });
    await act(async () => {
      await result.current.disable2FA('000000');
    });
    expect(notify).toHaveBeenCalledWith('Ошибка отключения 2FA');
  });

  it('disable2FA notifies on failure', async () => {
    api.disable2FA.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.disable2FA('000000');
    });
    expect(notify).toHaveBeenCalledWith('Не удалось отключить 2FA');
  });

  it('load2FAStatus updates profile flags', async () => {
    api.load2FAStatus.mockResolvedValue({
      enabled: true,
      backup_codes_remaining: 3,
    });
    await act(async () => {
      await result.current.load2FAStatus();
    });
    expect(update).toHaveBeenCalledWith({
      profile: {
        ...state.profile,
        twoFactor: true,
        backupCodesRemaining: 3,
      },
    });
  });

  it('load2FAStatus ignores empty payload', async () => {
    api.load2FAStatus.mockResolvedValue(null);
    await act(async () => {
      await result.current.load2FAStatus();
    });
    expect(update).not.toHaveBeenCalled();
  });

  it('load2FAStatus swallows errors', async () => {
    api.load2FAStatus.mockRejectedValue(new Error('boom'));
    await act(async () => {
      await result.current.load2FAStatus();
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).not.toHaveBeenCalled();
  });
});
