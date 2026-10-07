import { useState } from 'react';
import { useApp } from '../contexts/AppContext';

export default function Reset() {
  const { state, t, submitResetEmail, submitResetCode, submitNewPassword, go } =
    useApp();
  const step = state.resetStep || 0;
  const [code, setCode] = useState('');
  const [newPass, setNewPass] = useState('');
  const [confirmPass, setConfirmPass] = useState('');
  const [sending, setSending] = useState(false);
  const [resetting] = useState(false);

  if (step === 0) {
    return (
      <section className='auth'>
        <div className='auth-brand'>
          <h1>{t('Восстановление', 'Recovery')}</h1>
          <p>
            {t(
              'Укажи email для получения кода',
              'Enter your email to receive a code'
            )}
          </p>
        </div>
        <div className='field'>
          <label htmlFor='resetEmail'>Email</label>
          <input id='resetEmail' type='email' placeholder='example@mail.com' />
        </div>
        <button
          type='button'
          className='primary full'
          onClick={async () => {
            setSending(true);
            await submitResetEmail();
            setSending(false);
          }}
          disabled={sending}
        >
          {sending
            ? t('Отправка...', 'Sending...')
            : t('Получить код', 'Get code')}
        </button>
        <button
          type='button'
          className='secondary full'
          onClick={() => go('login')}
        >
          {t('Другие варианты входа', 'Other sign-in options')}
        </button>
      </section>
    );
  }
  if (step === 1) {
    return (
      <section className='auth'>
        <div className='auth-brand'>
          <h1>{t('Код из письма', 'Email code')}</h1>
          <p>{state.resetEmail || ''}</p>
        </div>
        <div className='field'>
          <label htmlFor='resetCode'>{t('Код', 'Code')}</label>
          <input
            id='resetCode'
            inputMode='numeric'
            maxLength={6}
            value={code}
            onChange={(e) => setCode(e.target.value)}
          />
        </div>
        <button
          type='button'
          className='primary full'
          onClick={submitResetCode}
        >
          {t('Продолжить', 'Continue')}
        </button>
        <button
          type='button'
          className='secondary full'
          onClick={() => go('login')}
        >
          {t('Назад', 'Back')}
        </button>
      </section>
    );
  }
  return (
    <section className='auth'>
      <div className='auth-brand'>
        <h1>{t('Новый пароль', 'New password')}</h1>
      </div>
      <div className='field'>
        <label htmlFor='newPassword'>{t('Новый пароль', 'New password')}</label>
        <input
          id='newPassword'
          type='password'
          value={newPass}
          onChange={(e) => setNewPass(e.target.value)}
        />
      </div>
      <div className='field'>
        <label htmlFor='newPasswordConfirm'>
          {t('Подтверждение пароля', 'Confirm password')}
        </label>
        <input
          id='newPasswordConfirm'
          type='password'
          value={confirmPass}
          onChange={(e) => setConfirmPass(e.target.value)}
        />
      </div>
      <button
        type='button'
        className='primary full'
        onClick={submitNewPassword}
        disabled={resetting}
      >
        {resetting ? t('Сохранение...', 'Saving...') : t('Сохранить', 'Save')}
      </button>
    </section>
  );
}
