import { useState } from 'react';
import { changeEmail } from '../../utils/api';
import './ProfileModals.css';

export default function ChangeEmailModal({ onClose }) {
  const [newEmail, setNewEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const isValidEmail = (value) => {
    const at = value.indexOf('@');
    if (at < 0) return false;
    const domain = value.slice(at + 1);
    if (!domain?.includes('.')) return false;
    const [local] = value.split('@');
    if (!local) return false;
    return !value.includes(' ');
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    if (!newEmail || !password) {
      setError('Заполните все поля');
      return;
    }
    if (!isValidEmail(newEmail)) {
      setError('Некорректный email');
      return;
    }
    setSubmitting(true);
    try {
      await changeEmail(newEmail, password);
      onClose();
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
      <button
        type='button'
        className='modal-overlay'
        onClick={onClose}
        onKeyDown={handleOverlayKeyDown}
        aria-label='Закрыть'
      />
      <div className='modal-content'>
        <h3>Сменить почту</h3>
        <form onSubmit={handleSubmit}>
          <div className='form-group'>
            <label htmlFor='newEmail'>Новый email</label>
            <input
              id='newEmail'
              type='email'
              value={newEmail}
              onChange={(e) => setNewEmail(e.target.value)}
              required
            />
            <div className='field-error'>{error}</div>
          </div>
          <div className='form-group'>
            <label htmlFor='currentPassword'>Текущий пароль</label>
            <input
              id='currentPassword'
              type='password'
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>
          <div className='modal-actions'>
            <button type='button' className='btn-secondary' onClick={onClose}>
              Отмена
            </button>
            <button type='submit' className='btn-primary' disabled={submitting}>
              {submitting ? 'Сохранение...' : 'Сохранить новую почту'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
