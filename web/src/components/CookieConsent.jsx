import { useState, useEffect } from 'react';
import { useApp } from '../contexts/AppContext';

export default function CookieConsent() {
  const { t } = useApp();
  const [visible, setVisible] = useState(false);
  const [accepted, setAccepted] = useState(() => {
    try {
      return localStorage.getItem('cookie-consent') === 'accepted';
    } catch {
      return true;
    }
  });

  useEffect(() => {
    if (!accepted) {
      setVisible(true);
    }
  }, [accepted]);

  const handleAccept = () => {
    try {
      localStorage.setItem('cookie-consent', 'accepted');
    } catch {
      // ignore storage errors
    }
    setAccepted(true);
    setVisible(false);
  };

  const handleDecline = () => {
    try {
      localStorage.setItem('cookie-consent', 'declined');
    } catch {
      // ignore storage errors
    }
    setAccepted(true);
    setVisible(false);
  };

  if (!visible) return null;

  return (
    <div className='cookie-consent'>
      <div className='cookie-consent-body'>
        <p>
          {t(
            'Мы используем файлы cookie для улучшения работы сервиса. Продолжая использовать FitPulse, вы соглашаетесь с нашей ',
            'We use cookies to improve the service. By continuing to use FitPulse, you agree to our '
          )}
          <a className='legal-link' href='/privacy' target='_blank' rel='noreferrer'>
            {t('Политикой конфиденциальности', 'Privacy Policy')}
          </a>
          .
        </p>
        <div className='cookie-consent-actions'>
          <button type='button' className='primary' onClick={handleAccept}>
            {t('Принять', 'Accept')}
          </button>
          <button type='button' className='secondary' onClick={handleDecline}>
            {t('Отклонить', 'Decline')}
          </button>
        </div>
      </div>
    </div>
  );
}
