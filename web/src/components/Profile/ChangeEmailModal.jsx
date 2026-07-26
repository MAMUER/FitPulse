import { useState } from 'react';
import { changeEmail } from '../../utils/api';
import './ProfileModals.css';

export default function ChangeEmailModal({ onClose }) {
  const [newEmail, setNewEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    if (!newEmail || !password) {
      setError('Заполните все поля');
      return;
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(newEmail)) {
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

  return (
    <div className='modal'>
      <div className='modal-overlay' onClick={onClose} />
      <div className='modal-content'>
        <h3>Сменить почту</h3>
        <form onSubmit={handleSubmit}>
          <div className='form-group'>
            <label>Новый email</label>
            <input
              type='email'
              value={newEmail}
              onChange={(e) => setNewEmail(e.target.value)}
              required
            />
            <div className='field-error'>{error}</div>
          </div>
          <div className='form-group'>
            <label>Текущий пароль</label>
            <input
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
