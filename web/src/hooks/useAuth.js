import { useCallback } from 'react';
import * as api from '../services/api';

export function useAuth({ state, update, notify }) {
  const doLogin = useCallback(async () => {
    const email = document.getElementById('loginEmail')?.value.trim();
    const pass = document.getElementById('loginPassword')?.value.trim();
    if (!email || !pass) {
      notify('Неверный email или пароль');
      return;
    }
    try {
      const data = await api.login(email, pass);
      if (data?.status !== 'ok') {
        notify('Неверный email или пароль');
        return;
      }
      update({
        profile: { ...state.profile, email, password: pass },
        registered: true,
        guest: false,
        screen: 'home',
      });
      window.location.href = '/home';
    } catch {
      notify('Ошибка входа');
    }
  }, [state.profile, update, notify]);

  const logout = useCallback(async () => {
    await api.logout();
    update({ registered: false, guest: false, screen: 'login' });
    window.location.href = '/';
  }, [update]);

  const doRegister = useCallback(() => {
    const email = document.getElementById('regEmail')?.value.trim();
    const pass = document.getElementById('regPassword')?.value;
    const confirm = document.getElementById('regConfirm')?.value;
    try {
      api.register(email, pass, confirm);
    } catch (e) {
      notify(e.message);
      return;
    }
    if (!document.getElementById('regConsent')?.checked) {
      notify('Подтвердите согласие с документами.');
      return;
    }
    update({
      guest: false,
      registrationData: { email, password: pass, confirm, code: '' },
      profile: { ...state.profile, email },
      registered: true,
      screen: 'login',
    });
    notify('Регистрация успешна. Проверьте email для подтверждения.');
  }, [state.profile, update, notify]);

  const forgotPassword = useCallback(async () => {
    update({ screen: 'reset', resetStep: 0 });
  }, [update]);

  const submitResetEmail = useCallback(async () => {
    const email = document.getElementById('resetEmail')?.value.trim();
    if (!email?.includes('@')) {
      notify('Введите корректный email');
      return;
    }
    try {
      await api.submitResetEmail(email);
      update({ resetEmail: email, resetStep: 1 });
      notify('Код отправлен на email');
    } catch {
      notify('Ошибка отправки кода');
    }
  }, [update, notify]);

  const submitResetCode = useCallback(() => {
    const code = document.getElementById('resetCode')?.value.trim();
    try {
      api.submitResetCode(code);
    } catch (e) {
      notify(e.message);
      return;
    }
    update({ resetStep: 2 });
  }, [update, notify]);

  const submitNewPassword = useCallback(async () => {
    const pass = document.getElementById('newPassword')?.value || '';
    const confirmPass =
      document.getElementById('newPasswordConfirm')?.value || '';
    if (pass.length < 6) {
      notify('Пароль должен содержать минимум 6 символов');
      return;
    }
    if (pass !== confirmPass) {
      notify('Пароли не совпадают');
      return;
    }
    try {
      await api.submitNewPassword(
        state.resetEmail,
        document.getElementById('resetCode')?.value.trim(),
        pass
      );
      update({
        profile: { ...state.profile, password: pass },
        screen: 'login',
      });
      notify('Пароль изменён. Теперь можно войти.');
    } catch {
      notify('Ошибка сброса пароля');
    }
  }, [state.profile, state.resetEmail, update, notify]);

  const continueAsGuest = useCallback(() => {
    update({ guest: true, registered: false, screen: 'home' });
    window.location.href = '/home';
  }, [update]);

  const socialLogin = useCallback(
    (provider) => {
      try {
        const result = api.socialLogin(provider);
        if (result.redirect) {
          window.location.href = result.redirect;
          return;
        }
      } catch (e) {
        notify(e.message);
      }
    },
    [notify]
  );

  const confirmEmail = useCallback(
    async (token) => {
      try {
        const data = await api.confirmEmail(token);
        if (data?.status === 'ok') {
          notify('Email подтверждён');
        } else {
          notify(data?.message || 'Ошибка подтверждения');
        }
      } catch {
        notify('Ошибка подтверждения email');
      }
    },
    [notify]
  );

  const setup2FA = useCallback(async () => {
    try {
      const data = await api.setup2FA();
      if (data) {
        update({
          twoFactorSetup: {
            qrCodeUrl: data.qr_code_url || '',
            qrCodeBase64: data.qr_code_base64 || '',
            secret: data.secret || '',
            backupCodes: data.backup_codes || [],
          },
          screen: 'twofa-setup',
        });
      }
    } catch {
      notify('Не удалось настроить 2FA');
    }
  }, [update, notify]);

  const confirm2FA = useCallback(
    async (passcode, tempSecret, backupCodes) => {
      try {
        const data = await api.confirm2FA(passcode, tempSecret, backupCodes);
        if (data?.success) {
          notify('2FA успешно настроена');
          update({ twoFactorSetup: null, screen: 'profile' });
        } else {
          notify(data?.message || 'Ошибка подтверждения 2FA');
        }
      } catch {
        notify('Не удалось подтвердить 2FA');
      }
    },
    [update, notify]
  );

  const verify2FA = useCallback(
    async (tempToken, passcode, isBackupCode = false) => {
      try {
        const data = await api.verify2FA(tempToken, passcode, isBackupCode);
        if (data?.status === 'ok') {
          update({
            registered: true,
            guest: false,
            screen: 'home',
            twoFactorTempToken: null,
          });
        } else {
          notify('Неверный код 2FA');
        }
      } catch {
        notify('Ошибка проверки 2FA');
      }
    },
    [update, notify]
  );

  const disable2FA = useCallback(
    async (passcode) => {
      try {
        const data = await api.disable2FA(passcode);
        if (data?.success) {
          update({ profile: { ...state.profile, twoFactor: false } });
          notify('2FA отключена');
        } else {
          notify(data?.message || 'Ошибка отключения 2FA');
        }
      } catch {
        notify('Не удалось отключить 2FA');
      }
    },
    [state.profile, update, notify]
  );

  const load2FAStatus = useCallback(async () => {
    try {
      const data = await api.load2FAStatus();
      if (data) {
        update({
          profile: {
            ...state.profile,
            twoFactor: data.enabled,
            backupCodesRemaining: data.backup_codes_remaining,
          },
        });
      }
    } catch {
      // ignore
    }
  }, [state.profile, update]);

  return {
    doLogin,
    logout,
    doRegister,
    forgotPassword,
    submitResetEmail,
    submitResetCode,
    submitNewPassword,
    continueAsGuest,
    socialLogin,
    confirmEmail,
    setup2FA,
    confirm2FA,
    verify2FA,
    disable2FA,
    load2FAStatus,
  };
}
