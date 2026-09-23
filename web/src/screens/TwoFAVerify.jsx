import { useState } from 'react';
import { useApp } from '../contexts/AppContext';

export default function TwoFAVerify() {
  const { verify2FA, t } = useApp();
  const [passcode, setPasscode] = useState('');
  const [backup, setBackup] = useState(false);
  const [loading, setLoading] = useState(false);
  const tempToken =
    new URLSearchParams(window.location.search).get('temp_token') || '';

  const submit = async (e) => {
    e.preventDefault();
    setLoading(true);
    await verify2FA(tempToken, passcode, backup);
    setLoading(false);
  };

  return (
    <section className='twofa'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>
            {t('Двухфакторная аутентификация', 'Two-Factor Authentication')}
          </div>
          <div className='panel-sub'>
            {t('Введите код из приложения', 'Enter code from app')}
          </div>
        </div>
        <div className='panel-body'>
          <form onSubmit={submit}>
            <div className='field'>
              <label>{t('Код', 'Code')}</label>
              <input
                value={passcode}
                onChange={(e) => setPasscode(e.target.value)}
                placeholder='123456'
                maxLength={6}
              />
            </div>
            <div className='field'>
              <label>
                <input
                  type='checkbox'
                  checked={backup}
                  onChange={(e) => setBackup(e.target.checked)}
                />
                {t('Это резервный код', 'This is a backup code')}
              </label>
            </div>
            <button
              className='primary full'
              type='submit'
              disabled={loading || !passcode}
            >
              {loading
                ? t('Проверка...', 'Verifying...')
                : t('Подтвердить', 'Verify')}
            </button>
          </form>
        </div>
      </div>
    </section>
  );
}
