import { useState } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { deleteProfile } from '../../utils/api';
import './ProfileModals.css';

export default function DeleteProfileModal({ onClose }) {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const { logout } = useAuth();

  const handleDelete = async () => {
    if (!password) {
      setError('Введите пароль');
      return;
    }
    if (!confirm('Вы уверены? Это действие нельзя отменить.')) return;
    setSubmitting(true);
    try {
      await deleteProfile(password);
      logout();
      window.location.href = '/';
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleOverlayKeyDown = (e) => {
    if (e.key === 'Escape') onClose();
  };

  return (
    <div className='modal'>
      <div className='modal-overlay' onClick={onClose} onKeyDown={handleOverlayKeyDown} />
      <div className='modal-content'>
        <h3 style={{ color: 'var(--accent)' }}>Удаление аккаунта</h3>
        <p className='delete-warning'>
          Это действие необратимо. Все ваши данные, тренировки и достижения
          будут удалены.
        </p>
        <div className='form-group'>
          <label>Введите пароль для подтверждения</label>
          <input
            type='password'
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder='Текущий пароль'
          />
          <div className='field-error'>{error}</div>
        </div>
        <div className='modal-actions'>
          <button type='button' className='btn-secondary' onClick={onClose}>
            Отмена
          </button>
          <button
            type='button'
            className='btn-danger'
            onClick={handleDelete}
            disabled={submitting}
          >
            {submitting ? 'Удаление...' : 'Удалить аккаунт'}
          </button>
        </div>
      </div>
    </div>
  );
}
