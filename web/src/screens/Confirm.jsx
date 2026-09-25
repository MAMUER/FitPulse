import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useApp } from '../contexts/AppContext';

export default function Confirm() {
  const { t, notify, confirmEmail } = useApp();
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') || '';
  const [status, setStatus] = useState('loading');
  const [message, setMessage] = useState('');

  let statusText;
  if (status === 'loading') {
    statusText = t('Подтверждение...', 'Confirming...');
  } else if (status === 'success') {
    statusText = t('Email подтверждён', 'Email confirmed');
  } else {
    statusText = t('Ошибка подтверждения', 'Confirmation error');
  }

  useEffect(() => {
    if (!token) {
      setStatus('error');
      setMessage(t('Токен не указан', 'Token is missing'));
      return;
    }
    confirmEmail(token)
      .then((data) => {
        if (data?.status === 'ok') {
          setStatus('success');
          setMessage(t('Email подтверждён', 'Email confirmed'));
        } else {
          setStatus('error');
          setMessage(
            data?.message || t('Ошибка подтверждения', 'Confirmation error')
          );
        }
      })
      .catch(() => {
        setStatus('error');
        setMessage(t('Ошибка подтверждения', 'Confirmation error'));
      });
  }, [token, t, confirmEmail]);

  return (
    <section className='auth'>
      <div className='auth-brand'>
        <h1>FitPulse</h1>
        <p>{statusText}</p>
        {message && <p>{message}</p>}
      </div>
    </section>
  );
}
