import { useState } from 'react';
import AuthLayout from '../components/AuthLayout';
import Captcha from '../components/Captcha';
import { useApp } from '../contexts/AppContext';
import { setCaptchaToken } from '../utils/backendRequest';

// Глазик: absolute-позиционирование, чтобы не зависеть от CSS.
const eyeButtonStyle = {
  position: 'absolute',
  right: 8,
  top: '50%',
  transform: 'translateY(-50%)',
  background: 'none',
  border: 'none',
  cursor: 'pointer',
  color: 'inherit',
  padding: 4,
};

export default function Register() {
  const { t, doRegister, socialLogin, go, openLegal } = useApp();
  const [showPass, setShowPass] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

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
        <div style={{ position: 'relative' }}>
          <input
            id='regPassword'
            type={showPass ? 'text' : 'password'}
            autoComplete='new-password'
            placeholder={t('Минимум 8 символов', 'At least 8 characters')}
            style={{ paddingRight: 40 }}
          />
          <button
            type='button'
            onClick={() => setShowPass((v) => !v)}
            aria-label={
              showPass
                ? t('Скрыть пароль', 'Hide password')
                : t('Показать пароль', 'Show password')
            }
            style={eyeButtonStyle}
          >
            <i className={`fas ${showPass ? 'fa-eye-slash' : 'fa-eye'}`} />
          </button>
        </div>
        <small className='pw-hint'>
          {t(
            'Минимум 8 символов. Буквы, цифры и спецсимволы повышают надёжность.',
            'At least 8 characters. Letters, digits and symbols make it stronger.'
          )}
        </small>
      </div>
      <div className='field'>
        <label htmlFor='regConfirm'>
          {t('Подтверждение пароля', 'Confirm password')}
        </label>
        <div style={{ position: 'relative' }}>
          <input
            id='regConfirm'
            type={showConfirm ? 'text' : 'password'}
            autoComplete='new-password'
            placeholder={t('Повторите пароль', 'Repeat password')}
            style={{ paddingRight: 40 }}
          />
          <button
            type='button'
            onClick={() => setShowConfirm((v) => !v)}
            aria-label={
              showConfirm
                ? t('Скрыть пароль', 'Hide password')
                : t('Показать пароль', 'Show password')
            }
            style={eyeButtonStyle}
          >
            <i className={`fas ${showConfirm ? 'fa-eye-slash' : 'fa-eye'}`} />
          </button>
        </div>
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
          'FitPulse не является медицинским сервисом и не ставит диагнозов. При заболеваниях или травмах проконсультируйтесь с врачом.',
          'FitPulse is not a medical service and does not provide diagnoses. Consult a doctor for medical conditions or injuries.'
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
