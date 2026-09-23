import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useApp } from '../contexts/AppContext';
import { backendRequest } from '../utils/backendRequest';

export default function Confirm() {
  const { t, notify } = useApp();
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') || '';
  const [status, setStatus] = useState('loading');
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (!token) {
      setStatus('error');
      setMessage(t('Токен не указан', 'Token is missing'));
      return;
    }
    backendRequest('/api/v1/auth/confirm', {
      method: 'POST',
      body: JSON.stringify({ token }),
    })
      .then((data) => {
        if (data && data.status === 'ok') {
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
  }, [token, t, notify]);

  return (
    <section className='auth'>
      <div className='auth-brand'>
        <h1>FitPulse</h1>
        <p>
          {status === 'loading'
            ? t('Подтверждение...', 'Confirming...')
            : status === 'success'
              ? t('Email подтверждён', 'Email confirmed')
              : t('Ошибка подтверждения', 'Confirmation error')}
        </p>
        {message && <p>{message}</p>}
      </div>
    </section>
  );
}
