import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { confirmEmail as apiConfirmEmail } from '../../utils/api';
import './Confirm.css';

export default function Confirm({ token: tokenProp }) {
  const [status, setStatus] = useState('loading');
  const [message, setMessage] = useState('');
  const navigate = useNavigate();

  const readToken = useCallback(() => {
    if (tokenProp) return tokenProp;
    try {
      return new URLSearchParams(window.location.search).get('token');
    } catch {
      return null;
    }
  }, [tokenProp]);

  useEffect(() => {
    const token = readToken();
    if (!token) {
      setStatus('error');
      setMessage(
        'Токен подтверждения не найден. Проверьте письмо и попробуйте снова.'
      );
      return;
    }

    apiConfirmEmail(token)
      .then(() => {
        setStatus('success');
        setMessage(
          'Email успешно подтверждён! Теперь вы можете войти в систему.'
        );
      })
      .catch((err) => {
        setStatus('error');
        setMessage(err.message || 'Ошибка подтверждения');
      });
  }, [readToken]);

  return (
    <div className='confirm-page'>
      <div className='confirm-container'>
        <h1>Подтверждение email</h1>
        {status === 'loading' && (
          <>
            <p>Пожалуйста, подождите...</p>
            <div className='spinner' />
          </>
        )}
        {status === 'success' && (
          <div className='result success'>{message}</div>
        )}
        {status === 'error' && <div className='result error'>{message}</div>}
        {status !== 'loading' && (
          <div className='back-link'>
            <button
              type='button'
              onClick={() => {
                navigate('/');
              }}
            >
              ← Вернуться ко входу
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
