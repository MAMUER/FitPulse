import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { confirmEmail as apiConfirmEmail } from '../../utils/api';
import './Confirm.css';

export default function Confirm() {
  const [status, setStatus] = useState('loading'); // loading, success, error
  const [message, setMessage] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    const token = new URLSearchParams(window.location.search).get('token');
    if (!token) {
      setStatus('error');
      setMessage('Токен подтверждения не найден. Проверьте письмо и попробуйте снова.');
      return;
    }

    apiConfirmEmail(token)
      .then(() => {
        setStatus('success');
        setMessage('Email успешно подтверждён! Теперь вы можете войти в систему.');
      })
      .catch((err) => {
        setStatus('error');
        setMessage(err.message || 'Ошибка подтверждения');
      });
  }, []);

  return (
    <div className="confirm-page">
      <div className="confirm-container">
        <h1>Подтверждение email</h1>
        {status === 'loading' && (
          <>
            <p>Пожалуйста, подождите...</p>
            <div className="spinner" />
          </>
        )}
        {status === 'success' && (
          <div className="result success">{message}</div>
        )}
        {status === 'error' && (
          <div className="result error">{message}</div>
        )}
        {status !== 'loading' && (
          <div className="back-link">
            <a href="#" onClick={(e) => { e.preventDefault(); navigate('/'); }}>
              ← Вернуться ко входу
            </a>
          </div>
        )}
      </div>
    </div>
  );
}
