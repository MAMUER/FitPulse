import { useState } from 'react';
import { changePassword } from '../../utils/api';
import './ProfileModals.css';

export default function ChangePasswordModal({ onClose }) {
  const [current, setCurrent] = useState('');
  const [newPass, setNewPass] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    if (!current || !newPass) {
      setError('Заполните все поля');
      return;
    }
    if (newPass.length < 8) {
      setError('Пароль минимум 8 символов');
      return;
    }
    if (newPass !== confirm) {
      setError('Пароли не совпадают');
      return;
    }
    setSubmitting(true);
    try {
      await changePassword(current, newPass);
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
        <h3>Сменить пароль</h3>
        <form onSubmit={handleSubmit}>
          <div className='form-group'>
            <label htmlFor='currentPassword'>Текущий пароль</label>
            <input
              id='currentPassword'
              type='password'
              value={current}
              onChange={(e) => setCurrent(e.target.value)}
              required
            />
            <div className='field-error'>
              {error && !current ? 'Введите текущий пароль' : ''}
            </div>
          </div>
          <div className='form-group'>
            <label htmlFor='newPassword'>Новый пароль</label>
            <input
              id='newPassword'
              type='password'
              value={newPass}
              onChange={(e) => setNewPass(e.target.value)}
              required
              minLength={8}
            />
            <div className='field-error'>
              {error && newPass.length < 8 ? 'Минимум 8 символов' : ''}
            </div>
          </div>
          <div className='form-group'>
            <label htmlFor='confirmPassword'>Подтверждение пароля</label>
            <input
              id='confirmPassword'
              type='password'
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              required
            />
            <div className='field-error'>
              {error && confirm && newPass !== confirm
                ? 'Пароли не совпадают'
                : ''}
            </div>
          </div>
          <div className='modal-actions'>
            <button type='button' className='btn-secondary' onClick={onClose}>
              Отмена
            </button>
            <button type='submit' className='btn-primary' disabled={submitting}>
              {submitting ? 'Сохранение...' : 'Сохранить новый пароль'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
