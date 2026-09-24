import { useNavigate } from 'react-router-dom';
import { useApp } from '../contexts/AppContext';

export default function Login() {
  const {
    state,
    t,
    doLogin,
    continueAsGuest,
    socialLogin,
    toggleHighContrast,
    toggleLanguage,
    forgotPassword,
    go,
  } = useApp();
  const navigate = useNavigate();

  return (
    <section className='auth'>
      <div className='auth-brand'>
        <img
          className='splash-logo'
          src='https://uploads.onecompiler.io/44temhxkf/1785230799105/fit%20pulse.jpg'
          alt='FitPulse'
        />
        <h1>FitPulse</h1>
        <p>{t('С возвращением', 'Welcome back')}</p>
      </div>
      <div className='field'>
        <label htmlFor='loginEmail'>{t('Электронная почта', 'Email')}</label>
        <input id='loginEmail' type='email' defaultValue='mih@example.com' />
      </div>
      <div className='field'>
        <label htmlFor='loginPassword'>{t('Пароль', 'Password')}</label>
        <input id='loginPassword' type='password' defaultValue='password' />
      </div>
      <button type='button' className='primary full' onClick={doLogin}>
        {t('Войти', 'Login')}
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
          onClick={() => navigate('/terms')}
        >
          {t('Пользовательское соглашение', 'Terms of Use')}
        </button>
        ,{' '}
        <button
          type='button'
          className='legal-link'
          onClick={() => navigate('/consent')}
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
          onClick={() => navigate('/privacy')}
        >
          {t('Политику конфиденциальности', 'Privacy Policy')}
        </button>{' '}
        .
      </div>
      <div className='socials'>
        <button
          type='button'
          className='google'
          onClick={() => socialLogin('google')}
        >
          <i className='fab fa-google'></i> {t('Google', 'Google')}
        </button>
      </div>
      <div className='auth-link' style={{ marginTop: 8 }}>
        <button
          type='button'
          className='secondary auth-control'
          onClick={toggleHighContrast}
          style={{ fontSize: 9 }}
        >
          <i className='fas fa-eye'></i>{' '}
          {t('Режим высокой контрастности', 'High contrast mode')}
        </button>
        <button
          type='button'
          className='secondary auth-control lang-control'
          onClick={toggleLanguage}
          style={{ fontSize: 9, marginLeft: 6 }}
        >
          <i className='fas fa-globe'></i>{' '}
          {state.language === 'ru' ? 'EN' : 'RU'}
        </button>
      </div>
    </section>
  );
}
