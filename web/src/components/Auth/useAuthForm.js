import { useCallback, useEffect, useState } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { register, verify2FA } from '../../utils/api';
import {
  validateEmail,
  validateLoginPassword,
  validateName,
  validatePassword,
} from '../../utils/validators';

const INITIAL_FORM = {
  email: '',
  password: '',
  name: '',
  totpCode: '',
  backupCode: '',
};

export function useAuthForm({ searchParams, onModeChange, onSuccessMessage }) {
  const { login } = useAuth();
  const [formData, setFormData] = useState(INITIAL_FORM);
  const [errors, setErrors] = useState({});
  const [generalError, setGeneralError] = useState('');
  const [passwordChecks, setPasswordChecks] = useState({
    length: false,
    upper: false,
    lower: false,
    digit: false,
  });
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const confirmToken = searchParams.get('token');
    if (confirmToken) {
      onModeChange?.('verify');
      setFormData((f) => ({ ...f, verifyToken: confirmToken }));
    }
  }, [searchParams, onModeChange]);

  const setField = useCallback((field, value) => {
    setFormData((f) => ({ ...f, [field]: value }));
    setErrors((e) => ({ ...e, [field]: '' }));
    setGeneralError('');
  }, []);

  const getFieldClass = useCallback(
    (fieldName) => {
      if (errors[fieldName]) return 'invalid';
      return formData[fieldName] ? 'valid' : '';
    },
    [errors, formData]
  );

  const validateLogin = useCallback(() => {
    const errs = {};
    const emailErr = validateEmail(formData.email);
    if (emailErr) errs.email = emailErr;
    const passErr = validateLoginPassword(formData.password);
    if (passErr) errs.password = passErr;
    setErrors(errs);
    return Object.keys(errs).length === 0;
  }, [formData.email, formData.password]);

  const validateRegister = useCallback(() => {
    const errs = {};
    const nameErr = validateName(formData.name);
    if (nameErr) errs.name = nameErr;
    const emailErr = validateEmail(formData.email);
    if (emailErr) errs.email = emailErr;
    const passResult = validatePassword(formData.password);
    if (passResult.error) errs.password = passResult.error;
    /* istanbul ignore next */
    setPasswordChecks(passResult.checks || {});
    setErrors(errs);
    return Object.keys(errs).length === 0;
  }, [formData.email, formData.name, formData.password]);

  const handleLogin = useCallback(
    async (e) => {
      e.preventDefault();
      setGeneralError('');
      if (!validateLogin()) {
        setGeneralError('Проверьте введённые данные');
        return;
      }
      setSubmitting(true);
      try {
        const data = await login(formData.email, formData.password);
        if (data.requires_2fa && data.temp_token) {
          onModeChange?.('login2fa');
          return { requires2FA: true, tempToken: data.temp_token };
        }
      } catch (err) {
        setGeneralError(err.message);
      } finally {
        setSubmitting(false);
      }
      return null;
    },
    [formData.email, formData.password, login, validateLogin, onModeChange]
  );

  const handleRegister = useCallback(
    async (e) => {
      e.preventDefault();
      setGeneralError('');
      if (!validateRegister()) {
        setGeneralError('Проверьте введённые данные');
        return;
      }
      setSubmitting(true);
      try {
        const data = await register(
          formData.email,
          formData.password,
          formData.name
        );
        onSuccessMessage?.(
          data.message || 'Регистрация успешна. Подтвердите email.'
        );
        onModeChange?.('verify');
      } catch (err) {
        setGeneralError(err.message);
      } finally {
        setSubmitting(false);
      }
    },
    [
      formData.email,
      formData.name,
      formData.password,
      register,
      validateRegister,
      onModeChange,
      onSuccessMessage,
    ]
  );

  const handleLogin2FA = useCallback(
    async (e, twoFATempToken) => {
      e.preventDefault();
      setGeneralError('');
      const code = formData.totpCode || formData.backupCode;
      if (!code) {
        setGeneralError('Введите код');
        return;
      }
      setSubmitting(true);
      try {
        const isBackup = !!formData.backupCode;
        const data = await verify2FA(twoFATempToken, code, isBackup);
        login(data.access_token);
      } catch (err) {
        setGeneralError(err.message);
      } finally {
        setSubmitting(false);
      }
    },
    [formData.totpCode, formData.backupCode, login, verify2FA]
  );

  const updatePasswordChecks = useCallback((value) => {
    const checks = {
      length: value.length >= 8,
      upper: /[A-ZА-ЯЁ]/.test(value),
      lower: /[a-zа-яё]/.test(value),
      digit: /\d/.test(value),
    };
    setPasswordChecks(checks);
    return checks;
  }, []);

  return {
    formData,
    errors,
    generalError,
    passwordChecks,
    submitting,
    setField,
    getFieldClass,
    validateLogin,
    validateRegister,
    handleLogin,
    handleRegister,
    handleLogin2FA,
    updatePasswordChecks,
  };
}
