import { useState } from 'react';
import { useApp } from '../contexts/AppContext';

export default function TwoFASetup() {
  const { confirm2FA, t } = useApp();
  const [passcode, setPasscode] = useState('');
  const [backupCodes] = useState([]);
  const [secret, setSecret] = useState('');
  const [qrCodeBase64] = useState('');
  const [loading, setLoading] = useState(false);

  const start = async () => {
    setLoading(true);
    await confirm2FA(passcode, secret, backupCodes);
    setLoading(false);
  };

  return (
    <section className='twofa'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Настройка 2FA', 'Setup 2FA')}</div>
          <div className='panel-sub'>
            {t('Сканируйте QR-код и введите код', 'Scan QR and enter code')}
          </div>
        </div>
        <div className='panel-body'>
          {qrCodeBase64 && (
            <img
              src={`data:image/png;base64,${qrCodeBase64}`}
              alt='QR'
              style={{ maxWidth: 220 }}
            />
          )}
          <div className='field'>
            <label>{t('Секрет', 'Secret')}</label>
            <input
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              readOnly
            />
          </div>
          <div className='field'>
            <label>{t('Код подтверждения', 'Confirmation code')}</label>
            <input
              value={passcode}
              onChange={(e) => setPasscode(e.target.value)}
              maxLength={6}
            />
          </div>
          <button
            className='primary full'
            onClick={start}
            disabled={loading || !passcode}
          >
            {loading
              ? t('Проверка...', 'Verifying...')
              : t('Подтвердить', 'Confirm')}
          </button>
        </div>
      </div>
    </section>
  );
}
