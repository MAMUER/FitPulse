import { useState } from 'react';
import { changeEmail } from '../../utils/api';
import Modal from './Modal';
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
      setError(
        'Заполните все поля. Email и текущий пароль обязательны для смены почты.'
      );
      return;
    }
    if (!isValidEmail(newEmail)) {
      setError('Некорректный email. Используйте формат name@example.com.');
      return;
    }
    setSubmitting(true);
    try {
      await changeEmail(newEmail, password);
      onClose();
    } catch (err) {
      setError(
        err.message ||
          'Не удалось сменить email. Проверьте текущий пароль и попробуйте снова.'
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      onClose={onClose}
      ariaLabel='Сменить почту'
      ariaDescribedby='email-modal-desc'
    >
      <h3>Сменить почту</h3>
      <p id='email-modal-desc' className='sr-only'>
        Форма смены email. После успешной смены новый адрес будет подтверждён
        автоматически.
      </p>
      <form onSubmit={handleSubmit}>
        <div className='form-group'>
          <label htmlFor='newEmail'>Новый email</label>
          <input
            id='newEmail'
            type='email'
            value={newEmail}
            onChange={(e) => setNewEmail(e.target.value)}
            required
            aria-invalid={!!error}
            aria-describedby={error ? 'email-error' : undefined}
          />
          <div className='field-error' id='email-error' role='alert'>
            {error}
          </div>
        </div>
        <div className='form-group'>
          <label htmlFor='currentPassword'>Текущий пароль</label>
          <input
            id='currentPassword'
            type='password'
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            aria-invalid={!!error}
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
    </Modal>
  );
}
