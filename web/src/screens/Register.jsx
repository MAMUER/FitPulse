import AuthLayout from '../components/AuthLayout';
import Captcha from '../components/Captcha';
import { useApp } from '../contexts/AppContext';
import { setCaptchaToken } from '../utils/backendRequest';

export default function Register() {
  const { t, doRegister, socialLogin, go, openLegal } = useApp();

  return (
    <AuthLayout>
      <div className='field'>
        <label htmlFor='regName'>{t('Имя', 'Name')}</label>
        <input
          id='regName'
          type='text'
          autoComplete='name'
          placeholder={t('Как к вам обращаться', 'What should we call you')}
        />
      </div>
      <div className='field'>
        <label htmlFor='regEmail'>{t('Электронная почта', 'Email')}</label>
        <input
          id='regEmail'
          type='email'
          autoComplete='email'
          placeholder='example@mail.com'
        />
      </div>
      <div className='field'>
        <label htmlFor='regPassword'>{t('Пароль', 'Password')}</label>
        <input
          id='regPassword'
          type='password'
          autoComplete='new-password'
          placeholder={t('Минимум 6 символов', 'At least 6 characters')}
        />
      </div>
      <div className='field'>
        <label htmlFor='regConfirm'>
          {t('Подтверждение пароля', 'Confirm password')}
        </label>
        <input
          id='regConfirm'
          type='password'
          autoComplete='new-password'
          placeholder={t('Повторите пароль', 'Repeat password')}
        />
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
            onClick={() => openLegal('terms')}
          >
            {t('Соглашение', 'Terms')}
          </button>{' '}
          ·{' '}
          <button
            type='button'
            className='legal-link'
            onClick={() => openLegal('consent')}
          >
            {t('Персональные данные', 'Personal Data')}
          </button>{' '}
          ·{' '}
          <button
            type='button'
            className='legal-link'
            onClick={() => openLegal('privacy')}
          >
            {t('Политика', 'Privacy')}
          </button>
        </span>
      </label>
      <label className='consent-row'>
        <input id='regConsentSpecial' type='checkbox' />{' '}
        <span>
          {t(
            'Я даю согласие на обработку специальных категорий персональных данных (сведения о здоровье, менструальном цикле) в соответствии с Политикой конфиденциальности.',
            'I consent to the processing of special categories of personal data (health information, menstrual cycle) in accordance with the Privacy Policy.'
          )}
        </span>
      </label>
      <p className='medical-disclaimer'>
        {t(
          'Это не медицинский совет. При заболеваниях или травмах проконсультируйтесь с врачом.',
          'This is not medical advice. Consult a doctor if you have medical conditions or injuries.'
        )}
      </p>
      <Captcha onToken={setCaptchaToken} />
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
          className='yandex'
          onClick={() => socialLogin('yandex')}
        >
          <i className='fab fa-yandex'></i> {t('Яндекс', 'Yandex')}
        </button>
      </div>
    </AuthLayout>
  );
}
