import { act, renderHook } from '@testing-library/react';
import { createElement } from 'react';
import { MemoryRouter, useLocation } from 'react-router-dom';
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
  createCriticalSession: vi.fn(),
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
  'regName',
  'regEmail',
  'regPassword',
  'regConfirm',
  'regConsent',
  'regConsentSpecial',
  'resetEmail',
  'resetCode',
  'newPassword',
  'newPasswordConfirm',
];

const state = { profile: { name: 'Имя' }, resetEmail: 'user@example.com' };

function Wrapper({ children }) {
  return createElement(MemoryRouter, { initialEntries: ['/login'] }, children);
}

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
  // 2FA-эндпоинты требуют critical session — по умолчанию отдаём токен.
  api.createCriticalSession.mockResolvedValue('crit-1');
  window.location.href = '';
  update = vi.fn();
  notify = vi.fn();
  for (const id of INPUT_IDS) {
    const el = document.createElement('input');
    el.id = id;
    document.body.appendChild(el);
  }
  result = renderHook(
    () => ({
      ...useAuth({ state, update, notify }),
      location: useLocation(),
    }),
    { wrapper: Wrapper }
  ).result;
});

afterEach(() => {
  for (const id of INPUT_IDS) {
    document.getElementById(id)?.remove();
  }
});

describe('useAuth', () => {
  it('returns all auth callbacks', () => {
    expect(
      Object.keys(result.current).filter((key) => key !== 'location')
    ).toEqual([
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
      authenticated: true,
      screen: 'home',
    });
    expect(result.current.location.pathname).toBe('/home');
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
    expect(notify).toHaveBeenCalledWith('Неверный email или пароль');
  });

  it('logout resets state and navigates to login', async () => {
    await act(async () => {
      await result.current.logout();
    });
    expect(api.logout).toHaveBeenCalled();
    expect(update).toHaveBeenCalledWith({
      registered: false,
      guest: false,
      authenticated: false,
      screen: 'login',
    });
    expect(result.current.location.pathname).toBe('/login');
  });

  it('doRegister notifies when consent is missing', async () => {
    setValue('regEmail', 'user@example.com');
    setValue('regPassword', 'secret123');
    setValue('regConfirm', 'secret123');
    await act(async () => {
      await result.current.doRegister();
    });
    expect(api.register).not.toHaveBeenCalled();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Подтвердите согласие с документами.');
  });

  it('doRegister posts registration data on success', async () => {
    setValue('regEmail', 'user@example.com');
    setValue('regPassword', 'secret123');
    setValue('regConfirm', 'secret123');
    setValue('regName', 'Alice');
    setChecked('regConsent', true);
    setChecked('regConsentSpecial', true);
    api.register.mockResolvedValue({ status: 'ok' });
    await act(async () => {
      await result.current.doRegister();
    });
    expect(api.register).toHaveBeenCalledWith(
      'user@example.com',
      'secret123',
      'Alice'
    );
    expect(update).toHaveBeenCalledWith({
      guest: false,
      registrationData: {
        email: 'user@example.com',
        password: 'secret123',
        confirm: 'secret123',
        code: '',
      },
      profile: { ...state.profile, email: 'user@example.com' },
      screen: 'login',
    });
    expect(result.current.location.pathname).toBe('/login');
  });

  it('doRegister surfaces api validation errors', async () => {
    setValue('regEmail', 'user@example.com');
    setValue('regPassword', 'secret123');
    setValue('regConfirm', 'secret123');
    setChecked('regConsent', true);
    setChecked('regConsentSpecial', true);
    api.register.mockRejectedValue(new Error('Пароли не совпадают'));
    await act(async () => {
      await result.current.doRegister();
    });
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Пароли не совпадают');
  });

  it('forgotPassword moves to reset step and navigates to /reset', async () => {
    await act(async () => {
      await result.current.forgotPassword();
    });
    expect(update).toHaveBeenCalledWith({ screen: 'reset', resetStep: 0 });
    expect(result.current.location.pathname).toBe('/reset');
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
    expect(notify).toHaveBeenCalledWith(
      'Не удалось отправить код. Проверьте, что email зарегистрирован.'
    );
  });

  it('submitResetCode rejects a non-6-digit code', () => {
    setValue('resetCode', '123');
    result.current.submitResetCode();
    expect(update).not.toHaveBeenCalled();
    expect(notify).toHaveBeenCalledWith('Код должен состоять из 6 цифр');
  });

  it('submitResetCode advances to password step', () => {
    setValue('resetCode', '123456');
    result.current.submitResetCode();
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

  it('continueAsGuest redirects to login (guest mode removed)', () => {
    act(() => result.current.continueAsGuest());
    expect(result.current.location.pathname).toBe('/login');
  });

  it('socialLogin redirects to the yandex oauth endpoint', () => {
    // jsdom не реализует навигацию — подменяем location, чтобы проверить переход.
    const original = window.location;
    const assign = vi.fn();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { assign },
    });
    api.socialLogin.mockReturnValue({ redirect: '/api/v1/auth/yandex' });

    result.current.socialLogin('yandex');

    expect(assign).toHaveBeenCalledWith('/api/v1/auth/yandex');
    expect(notify).not.toHaveBeenCalled();

    Object.defineProperty(window, 'location', {
      configurable: true,
      value: original,
    });
  });

  it('socialLogin notifies for other providers', () => {
    result.current.socialLogin('google');
    expect(notify).toHaveBeenCalledWith('Этот способ входа пока не настроен.');
  });

  it('socialLogin notifies when the backend returns no redirect', () => {
    api.socialLogin.mockReturnValue({});
    result.current.socialLogin('yandex');
    expect(notify).toHaveBeenCalledWith('Этот способ входа пока не настроен.');
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
    expect(api.confirm2FA).toHaveBeenCalledWith(
      '123456',
      'SECRET',
      ['backup'],
      'crit-1'
    );
    expect(api.createCriticalSession).toHaveBeenCalled();
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
    expect(api.disable2FA).toHaveBeenCalledWith('123456', 'crit-1');
    expect(api.createCriticalSession).toHaveBeenCalled();
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
