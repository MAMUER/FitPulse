import AuthLayout from '../components/AuthLayout';
import { useApp } from '../contexts/AppContext';

export default function Register() {
  const {
    t,
    doRegister,
    socialLogin,
    go,
  } = useApp();

  return (
    <AuthLayout>
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
            onClick={() => go('legal', 'terms')}
          >
            {t('Соглашение', 'Terms')}
          </button>{' '}
          ·{' '}
          <button
            type='button'
            className='legal-link'
            onClick={() => go('legal', 'consent')}
          >
            {t('Персональные данные', 'Personal Data')}
          </button>{' '}
          ·{' '}
          <button
            type='button'
            className='legal-link'
            onClick={() => go('legal', 'privacy')}
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
          {t('Войти', 'Sign in')}
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
    </AuthLayout>
  );
}
