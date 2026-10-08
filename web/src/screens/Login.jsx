import AuthLayout from '../components/AuthLayout';
import { useApp } from '../contexts/AppContext';

export default function Login() {
  const { t, doLogin, continueAsGuest, socialLogin, forgotPassword, go } =
    useApp();

  return (
    <AuthLayout>
      <div className='field'>
        <label htmlFor='loginEmail'>{t('Электронная почта', 'Email')}</label>
        <input id='loginEmail' type='email' defaultValue='mih@example.com' />
      </div>
      <div className='field'>
        <label htmlFor='loginPassword'>{t('Пароль', 'Password')}</label>
        <input id='loginPassword' type='password' defaultValue='password' />
      </div>
      <button type='button' className='primary full' onClick={doLogin}>
        {t('Войти', 'Sign in')}
      </button>
      <div className='auth-link'>
        {t('Нет аккаунта?', "Don't have an account?")}{' '}
        <button type='button' onClick={() => go('register')}>
          {t('Зарегистрироваться', 'Register')}
        </button>
      </div>
      <div className='auth-link'>
        <button type='button' onClick={forgotPassword}>
          {t('Забыли пароль?', 'Forgot password?')}
        </button>
      </div>
      <button
        type='button'
        className='secondary full'
        onClick={continueAsGuest}
      >
        {t('Продолжить как гость', 'Continue as guest')}
      </button>
      <div className='auth-link'>
        {t('Продолжая, вы принимаете', 'By continuing, you accept')}{' '}
        <button
          type='button'
          className='legal-link'
          onClick={() => go('legal', 'terms')}
        >
          {t('Пользовательское соглашение', 'Terms of Use')}
        </button>
        ,{' '}
        <button
          type='button'
          className='legal-link'
          onClick={() => go('legal', 'consent')}
        >
          {t(
            'соглашение об использовании персональных данных',
            'Personal Data Agreement'
          )}
        </button>{' '}
        {t('и', 'and')}{' '}
        <button
          type='button'
          className='legal-link'
          onClick={() => go('legal', 'privacy')}
        >
          {t('Политику конфиденциальности', 'Privacy Policy')}
        </button>{' '}
        .
      </div>
      <div className='socials'>
        <button
          type='button'
          className='yandex'
          onClick={() => socialLogin('yandex')}
        >
          <i className='fab fa-yandex'></i> {t('Яндекс', 'Yandex')}
        </button>
      </div>
    </AuthLayout>
  );
}
