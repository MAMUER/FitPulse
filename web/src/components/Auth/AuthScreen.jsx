import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useAuthForm } from './useAuthForm';
import './Auth.css';

export default function AuthScreen({ searchParams: searchParamsProp }) {
  const [routerSearchParams] = useSearchParams();
  const searchParams = searchParamsProp || routerSearchParams;
  const [mode, setMode] = useState('login');
  const [successMessage, setSuccessMessage] = useState('');
  const [twoFATempToken, setTwoFATempToken] = useState(null);

  const {
    formData,
    errors,
    generalError,
    passwordChecks,
    submitting,
    setField,
    getFieldClass,
    handleLogin,
    handleRegister,
    handleLogin2FA,
    updatePasswordChecks,
  } = useAuthForm({
    searchParams,
    onModeChange: setMode,
    onSuccessMessage: setSuccessMessage,
  });

  const handleLoginSubmit = async (e) => {
    const result = await handleLogin(e);
    if (result?.requires2FA) {
      setTwoFATempToken(result.tempToken);
    }
  };

  const handleLogin2FASubmit = (e) => {
    handleLogin2FA(e, twoFATempToken);
  };

  return (
    <div className='auth-screen'>
      <div className='auth-container'>
        <div className='auth-logo'>
          <div className='logo-icon'>💓</div>
          <h1>FitPulse</h1>
          <p>Ваш персональный AI-тренер</p>
        </div>

        <div
          className='auth-landing'
          style={{
            textAlign: 'center',
            color: 'var(--text-secondary)',
            fontSize: 15,
            lineHeight: 1.6,
            maxWidth: 520,
            margin: '0 auto 28px',
            padding: '0 20px',
          }}
        >
          <p style={{ marginBottom: 12 }}>
            FitPulse — это открытая платформа для фитнес- и health-трекинга. Мы
            помогаем отслеживать пульс, SpO2, шаги, сон и тренировки,
            синхронизировать данные с носимых устройств и получать
            персонализированные insights.
          </p>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))',
              gap: 10,
              textAlign: 'left',
              marginTop: 18,
            }}
          >
            {[
              '📊 Биометрия и активность',
              '⌚ Все основные бренды',
              '🤖 AI-планы тренировок',
              '🔒 End-to-end защита',
            ].map((feature) => (
              <div
                key={feature}
                style={{
                  background: 'var(--bg-card)',
                  borderRadius: 'var(--radius-md)',
                  padding: '12px 14px',
                  fontSize: 13,
                }}
              >
                {feature}
              </div>
            ))}
          </div>
          <p
            style={{
              marginTop: 16,
              fontSize: 13,
              color: 'var(--text-tertiary)',
            }}
          >
            Мы собираем только данные, необходимые для работы сервиса: учётные
            записи, биометрию с устройств, технические логи. Вы можете запросить
            копию или удаление данных в любой момент.
          </p>
        </div>

        {mode === 'login' && (
          <form className='auth-form' onSubmit={handleLoginSubmit} noValidate>
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
                className={getFieldClass('email')}
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
              <button
                type='button'
                className='link-button'
                onClick={() => setMode('register')}
              >
                Создать
              </button>
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
                className={getFieldClass('name')}
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
                className={getFieldClass('email')}
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
                submitting || Object.values(passwordChecks).some((v) => !v)
              }
            >
              {/* istanbul ignore next */}
              {submitting ? 'Создание...' : 'Создать аккаунт'}
            </button>
            <p className='auth-switch'>
              Уже есть?{' '}
              <button
                type='button'
                className='link-button'
                onClick={() => setMode('login')}
              >
                Войти
              </button>
            </p>
          </form>
        )}

        {mode === 'login2fa' && (
          <form
            className='auth-form'
            onSubmit={handleLogin2FASubmit}
            noValidate
          >
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
              <button
                type='button'
                className='link-button'
                onClick={() => {
                  setMode('login');
                  setTwoFATempToken(null);
                }}
              >
                ← Вернуться ко входу
              </button>
            </p>
          </form>
        )}

        {/* istanbul ignore next */}
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
            {/* istanbul ignore next */}
            {successMessage && (
              <div className='auth-success'>{successMessage}</div>
            )}
            {/* istanbul ignore next */}
            {generalError && <div className='auth-error'>{generalError}</div>}
            <p className='auth-switch'>
              <button
                type='button'
                className='link-button'
                onClick={() => setMode('login')}
              >
                ← Вернуться ко входу
              </button>
            </p>
          </div>
        )}

        <div
          style={{
            textAlign: 'center',
            color: 'var(--text-tertiary)',
            fontSize: 13,
            display: 'flex',
            gap: 16,
            justifyContent: 'center',
            flexWrap: 'wrap',
            padding: '0 24px 24px',
          }}
        >
          <a
            href='/privacy'
            style={{ color: 'inherit', textDecoration: 'none' }}
          >
            Политика конфиденциальности
          </a>
          <span>•</span>
          <a href='/terms' style={{ color: 'inherit', textDecoration: 'none' }}>
            Пользовательское соглашение
          </a>
        </div>
      </div>
    </div>
  );
}
