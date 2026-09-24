import { useNavigate } from 'react-router-dom';
import { useApp } from '../contexts/AppContext';

export default function Register() {
  const {
    state,
    t,
    doRegister,
    socialLogin,
    toggleHighContrast,
    toggleLanguage,
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
        <p>{t('Создайте аккаунт', 'Create an account')}</p>
      </div>
      <div className='field'>
        <label htmlFor='regEmail'>{t('Электронная почта', 'Email')}</label>
        <input id='regEmail' type='email' placeholder='example@mail.com' />
      </div>
      <div className='field'>
        <label htmlFor='regPassword'>{t('Пароль', 'Password')}</label>
        <input id='regPassword' type='password' placeholder='••••••••' />
      </div>
      <div className='field'>
        <label htmlFor='regConfirm'>
          {t('Подтверждение пароля', 'Confirm password')}
        </label>
        <input id='regConfirm' type='password' placeholder='••••••••' />
      </div>
      <label className='consent-row'>
        <input id='regConsent' type='checkbox' />{' '}
        <span>
          {t(
            'Я согласен(на) с Пользовательским соглашением и Политикой конфиденциальности.',
            'I agree to the Terms of Use and Privacy Policy.'
          )}{' '}
          <button
            type='button'
            className='legal-link'
            onClick={() => navigate('/terms')}
          >
            {t('Соглашение', 'Terms')}
          </button>{' '}
          ·{' '}
          <button
            type='button'
            className='legal-link'
            onClick={() => navigate('/consent')}
          >
            {t('Персональные данные', 'Personal Data')}
          </button>{' '}
          ·{' '}
          <button
            type='button'
            className='legal-link'
            onClick={() => navigate('/privacy')}
          >
            {t('Политика', 'Privacy')}
          </button>
        </span>
      </label>
      <button type='button' className='primary full' onClick={doRegister}>
        {t('Зарегистрироваться', 'Register')}
      </button>
      <div className='auth-link'>
        {t('Уже есть аккаунт?', 'Already have an account?')}{' '}
        <button type='button' onClick={() => go('login')}>
          {t('Войти', 'Login')}
        </button>
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
