import { useState } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { deleteProfile } from '../../utils/api';
import Modal from './Modal';
import './ProfileModals.css';

export default function DeleteProfileModal({ onClose }) {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const { logout } = useAuth();

  const handleDelete = async () => {
    if (!password) {
      setError('Введите пароль для подтверждения удаления аккаунта.');
      return;
    }
    setSubmitting(true);
    try {
      await deleteProfile(password);
      logout();
      window.location.href = '/';
    } catch (err) {
      setError(
        err.message ||
          'Не удалось удалить аккаунт. Проверьте пароль и попробуйте снова.'
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      onClose={onClose}
      ariaLabel='Удаление аккаунта'
      ariaDescribedby='delete-modal-desc'
    >
      <h3 style={{ color: 'var(--accent)' }}>Удаление аккаунта</h3>
      <p className='delete-warning' id='delete-modal-desc'>
        Это действие необратимо. Все ваши данные, тренировки и достижения будут
        удалены.
      </p>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          handleDelete();
        }}
      >
        <div className='form-group'>
          <label htmlFor='deletePassword'>
            Введите пароль для подтверждения
          </label>
          <input
            id='deletePassword'
            type='password'
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder='Текущий пароль'
            aria-invalid={!!error}
            aria-describedby={error ? 'delete-password-error' : undefined}
          />
          <div className='field-error' id='delete-password-error' role='alert'>
            {error}
          </div>
        </div>
        <div className='modal-actions'>
          <button type='button' className='btn-secondary' onClick={onClose}>
            Отмена
          </button>
          <button type='submit' className='btn-danger' disabled={submitting}>
            {submitting ? 'Удаление...' : 'Удалить аккаунт'}
          </button>
        </div>
      </form>
    </Modal>
  );
}
