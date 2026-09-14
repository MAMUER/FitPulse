import { useState } from 'react';
import { changePassword } from '../../utils/api';
import Modal from './Modal';
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
      setError('Заполните все поля. Текущий и новый пароль обязательны.');
      return;
    }
    if (newPass.length < 8) {
      setError('Новый пароль должен содержать минимум 8 символов.');
      return;
    }
    if (newPass !== confirm) {
      setError(
        'Пароли не совпадают. Проверьте ввод нового пароля и подтверждения.'
      );
      return;
    }
    setSubmitting(true);
    try {
      await changePassword(current, newPass);
      onClose();
    } catch (err) {
      setError(
        err.message ||
          'Не удалось сменить пароль. Проверьте текущий пароль и попробуйте снова.'
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      onClose={onClose}
      ariaLabel='Сменить пароль'
      ariaDescribedby='password-modal-desc'
    >
      <h3>Сменить пароль</h3>
      <p id='password-modal-desc' className='sr-only'>
        Форма смены пароля. Новый пароль должен содержать минимум 8 символов.
      </p>
      <form onSubmit={handleSubmit}>
        <div className='form-group'>
          <label htmlFor='currentPassword'>Текущий пароль</label>
          <input
            id='currentPassword'
            type='password'
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
            required
            aria-invalid={!!error}
          />
          <div className='field-error' role='alert'>
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
            aria-invalid={!!error}
            aria-describedby={error ? 'password-error' : 'password-hint'}
          />
          <div className='field-error' id='password-error' role='alert'>
            {error && newPass.length < 8 ? 'Минимум 8 символов' : ''}
          </div>
          <div id='password-hint' className='sr-only'>
            Новый пароль должен содержать минимум 8 символов.
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
            aria-invalid={!!error}
          />
          <div className='field-error' role='alert'>
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
    </Modal>
  );
}
