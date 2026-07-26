import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useAuth } from '../../contexts/AuthContext';
import { register, verify2FA } from '../../utils/api';
import {
  validateEmail,
  validateLoginPassword,
  validateName,
  validatePassword,
} from '../../utils/validators';
import './Auth.css';

export default function AuthScreen() {
  const [searchParams] = useSearchParams();
  const { login } = useAuth();
  const [mode, setMode] = useState('login'); // login, register, verify, login2fa
  const [errors, setErrors] = useState({});
  const [formData, setFormData] = useState({
    email: '',
    password: '',
    name: '',
    totpCode: '',
    backupCode: '',
  });
  const [twoFATempToken, setTwoFATempToken] = useState(null);
  const [generalError, setGeneralError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');
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
      setMode('verify');
      setFormData((f) => ({ ...f, verifyToken: confirmToken }));
    }
  }, [searchParams]);

  const setField = (field, value) => {
    setFormData((f) => ({ ...f, [field]: value }));
    setErrors((e) => ({ ...e, [field]: '' }));
    setGeneralError('');
  };

  const validateLogin = () => {
    const errs = {};
    const emailErr = validateEmail(formData.email);
    if (emailErr) errs.email = emailErr;
    const passErr = validateLoginPassword(formData.password);
    if (passErr) errs.password = passErr;
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const validateRegister = () => {
    const errs = {};
    const nameErr = validateName(formData.name);
    if (nameErr) errs.name = nameErr;
    const emailErr = validateEmail(formData.email);
    if (emailErr) errs.email = emailErr;
    const passResult = validatePassword(formData.password);
    if (passResult.error) errs.password = passResult.error;
    setPasswordChecks(passResult.checks || {});
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleLogin = async (e) => {
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
        setTwoFATempToken(data.temp_token);
        setMode('login2fa');
      }
    } catch (err) {
      setGeneralError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleRegister = async (e) => {
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
      setSuccessMessage(
        data.message || 'Регистрация успешна. Подтвердите email.'
      );
      setMode('verify');
    } catch (err) {
      setGeneralError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleLogin2FA = async (e) => {
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
  };

  const updatePasswordChecks = (value) => {
    const checks = {
      length: value.length >= 8,
      upper: /[A-ZА-ЯЁ]/.test(value),
      lower: /[a-zа-яё]/.test(value),
      digit: /\d/.test(value),
    };
    setPasswordChecks(checks);
    return checks;
  };

  return (
    <div className='auth-screen'>
      <div className='auth-container'>
        <div className='auth-logo'>
          <div className='logo-icon'>💓</div>
          <h1>FitPulse</h1>
          <p>Ваш персональный AI-тренер</p>
        </div>

        {mode === 'login' && (
          <form className='auth-form' onSubmit={handleLogin} noValidate>
            <div className='field'>
              <input
                type='email'
                placeholder='Email'
                value={formData.email}
                onChange={(e) => setField('email', e.target.value)}
                autoComplete='email'
                required
                maxLength={254}
                inputMode='email'
                className={
                  errors.email ? 'invalid' : formData.email ? 'valid' : ''
                }
              />
              <div className='field-error'>{errors.email || ''}</div>
            </div>
            <div className='field'>
              <input
                type='password'
                placeholder='Пароль'
                value={formData.password}
                onChange={(e) => setField('password', e.target.value)}
                autoComplete='current-password'
                required
                maxLength={128}
                className={errors.password ? 'invalid' : ''}
              />
              <div className='field-error'>{errors.password || ''}</div>
            </div>
            <div className='auth-error hidden'>{generalError}</div>
            <button type='submit' className='btn-primary' disabled={submitting}>
              {submitting ? 'Вход...' : 'Войти'}
            </button>
            <p className='auth-switch'>
              Нет аккаунта?{' '}
              <a
                href='#'
                onClick={(e) => {
                  e.preventDefault();
                  setMode('register');
                }}
              >
                Создать
              </a>
            </p>
          </form>
        )}

        {mode === 'register' && (
          <form className='auth-form' onSubmit={handleRegister} noValidate>
            <div className='field'>
              <input
                type='text'
                placeholder='Имя'
                value={formData.name}
                onChange={(e) => setField('name', e.target.value)}
                autoComplete='name'
                required
                maxLength={100}
                minLength={2}
                className={
                  errors.name ? 'invalid' : formData.name ? 'valid' : ''
                }
              />
              <div className='field-error'>{errors.name || ''}</div>
            </div>
            <div className='field'>
              <input
                type='email'
                placeholder='Email'
                value={formData.email}
                onChange={(e) => setField('email', e.target.value)}
                autoComplete='email'
                required
                maxLength={254}
                inputMode='email'
                className={
                  errors.email ? 'invalid' : formData.email ? 'valid' : ''
                }
              />
              <div className='field-error'>{errors.email || ''}</div>
            </div>
            <div className='field'>
              <input
                type='password'
                placeholder='Пароль (мин. 8 символов)'
                value={formData.password}
                onChange={(e) => {
                  setField('password', e.target.value);
                  updatePasswordChecks(e.target.value);
                }}
                autoComplete='new-password'
                required
                minLength={8}
                maxLength={128}
                className={errors.password ? 'invalid' : ''}
              />
              <div className='field-error'>{errors.password || ''}</div>
              <div
                className={`password-hint ${formData.password ? '' : 'hidden'}`}
              >
                <span
                  className={`hint-item ${passwordChecks.length ? 'pass' : ''}`}
                >
                  {passwordChecks.length ? '✓' : '✗'} 8+ символов
                </span>
                <span
                  className={`hint-item ${passwordChecks.upper ? 'pass' : ''}`}
                >
                  {passwordChecks.upper ? '✓' : '✗'} Заглавная буква
                </span>
                <span
                  className={`hint-item ${passwordChecks.lower ? 'pass' : ''}`}
                >
                  {passwordChecks.lower ? '✓' : '✗'} Строчная буква
                </span>
                <span
                  className={`hint-item ${passwordChecks.digit ? 'pass' : ''}`}
                >
                  {passwordChecks.digit ? '✓' : '✗'} Цифра
                </span>
              </div>
            </div>
            <div className='auth-error hidden'>{generalError}</div>
            <button
              type='submit'
              className='btn-primary'
              disabled={
                submitting || !!Object.values(passwordChecks).find((v) => !v)
              }
            >
              {submitting ? 'Создание...' : 'Создать аккаунт'}
            </button>
            <p className='auth-switch'>
              Уже есть?{' '}
              <a
                href='#'
                onClick={(e) => {
                  e.preventDefault();
                  setMode('login');
                }}
              >
                Войти
              </a>
            </p>
          </form>
        )}

        {mode === 'login2fa' && (
          <form className='auth-form' onSubmit={handleLogin2FA} noValidate>
            <h2>Двухфакторная аутентификация</h2>
            <p className='verify-text'>
              Введите код из приложения-аутентификатора.
            </p>
            <div className='field'>
              <input
                type='text'
                placeholder='6-значный код'
                value={formData.totpCode}
                onChange={(e) => setField('totpCode', e.target.value)}
                maxLength={6}
                inputMode='numeric'
                autoComplete='one-time-code'
              />
              <div className='field-error'></div>
            </div>
            <div className='field'>
              <input
                type='text'
                placeholder='Резервный код xxxx-xxxx'
                value={formData.backupCode}
                onChange={(e) => setField('backupCode', e.target.value)}
                maxLength={9}
              />
              <div className='field-error'></div>
            </div>
            <div className='auth-error'>{generalError}</div>
            <button type='submit' className='btn-primary' disabled={submitting}>
              {submitting ? 'Вход...' : 'Войти'}
            </button>
            <button
              type='button'
              className='btn-secondary'
              onClick={() => setField('backupCode', formData.totpCode)}
            >
              Использовать резервный код
            </button>
            <p className='auth-switch'>
              <a
                href='#'
                onClick={(e) => {
                  e.preventDefault();
                  setMode('login');
                  setTwoFATempToken(null);
                }}
              >
                ← Вернуться ко входу
              </a>
            </p>
          </form>
        )}

        {mode === 'verify' && (
          <div className='auth-form verify-form'>
            <div className='verify-icon'>📧</div>
            <h2>Проверьте почту</h2>
            <p className='verify-text'>
              Мы отправили письмо на{' '}
              <strong>{formData.email || 'ваш email'}</strong>
            </p>
            <p className='verify-text'>
              Перейдите по ссылке из письма, чтобы подтвердить email и войти.
            </p>
            {successMessage && (
              <div className='auth-success'>{successMessage}</div>
            )}
            {generalError && <div className='auth-error'>{generalError}</div>}
            <p className='auth-switch'>
              <a
                href='#'
                onClick={(e) => {
                  e.preventDefault();
                  setMode('login');
                }}
              >
                ← Вернуться ко входу
              </a>
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
