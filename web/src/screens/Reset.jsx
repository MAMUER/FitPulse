import { useApp } from '../contexts/AppContext';

export default function Reset() {
  const { state, t, submitResetEmail, submitResetCode, submitNewPassword, go } =
    useApp();
  const step = state.resetStep || 0;

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
          <label>Email</label>
          <input id='resetEmail' type='email' placeholder='example@mail.com' />
        </div>
        <button className='primary full' onClick={submitResetEmail}>
          {t('Получить код', 'Get code')}
        </button>
        <button className='secondary full' onClick={() => go('login')}>
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
          <label>123456</label>
          <input id='resetCode' inputMode='numeric' maxLength={6} />
        </div>
        <button className='primary full' onClick={submitResetCode}>
          {t('Продолжить', 'Continue')}
        </button>
        <button className='secondary full' onClick={() => go('login')}>
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
        <label>{t('Новый пароль', 'New password')}</label>
        <input id='newPassword' type='password' />
      </div>
      <div className='field'>
        <label>{t('Подтверждение пароля', 'Confirm password')}</label>
        <input id='newPasswordConfirm' type='password' />
      </div>
      <button className='primary full' onClick={submitNewPassword}>
        {t('Сохранить', 'Save')}
      </button>
    </section>
  );
}
