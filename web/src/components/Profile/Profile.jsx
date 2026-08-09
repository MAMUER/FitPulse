import { useCallback, useEffect, useState } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { getProfile, updateProfile } from '../../utils/api';
import {
  calculateBMI,
  validateAge,
  validateHeight,
  validateNickname,
  validateWeight,
} from '../../utils/validators';
import ChangeEmailModal from './ChangeEmailModal';
import ChangePasswordModal from './ChangePasswordModal';
import DeleteProfileModal from './DeleteProfileModal';
import TwoFASetup from './TwoFASetup';
import './Profile.css';

export default function Profile() {
  const { refreshProfile } = useAuth();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [errors, setErrors] = useState({});
  const [toast, setToast] = useState('');
  const [showPasswordModal, setShowPasswordModal] = useState(false);
  const [showEmailModal, setShowEmailModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const [form, setForm] = useState({
    nickname: '',
    age: '',
    gender: '',
    height: '',
    weight: '',
    fitness: '',
    nutrition: '',
    allergies: '',
    contraindications: '',
    goal: '',
  });

  const loadProfile = useCallback(async () => {
    try {
      const data = await getProfile();
      const p = data.profile || data;
      setForm({
        nickname: p.full_name || p.nickname || '',
        age: p.age || '',
        gender: p.gender || '',
        height: p.height_cm || '',
        weight: p.weight_kg || '',
        fitness: p.fitness_level || '',
        nutrition: p.nutrition || '',
        allergies: (p.allergies || []).join(', '),
        contraindications: (p.contraindications || []).join(', '),
        goal: p.goals?.[0] || '',
      });
    } catch (e) {
      console.error('Failed to load profile:', e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  const setField = (field, value) => {
    setForm((f) => ({ ...f, [field]: value }));
    setErrors((e) => ({ ...e, [field]: '' }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const errs = {};
    const nickErr = validateNickname(form.nickname);
    if (nickErr) errs.nickname = nickErr;
    const ageErr = validateAge(form.age);
    if (ageErr) errs.age = ageErr;
    const heightErr = validateHeight(form.height);
    if (heightErr) errs.height = heightErr;
    const weightErr = validateWeight(form.weight);
    if (weightErr) errs.weight = weightErr;

    setErrors(errs);
    if (Object.keys(errs).length > 0) {
      setToast('Исправьте ошибки в полях');
      return;
    }

    setSaving(true);
    try {
      const data = {
        full_name: form.nickname.trim(),
        age: form.age ? Number.parseInt(form.age, 10) : null,
        gender: form.gender || null,
        height_cm: form.height ? Number.parseInt(form.height, 10) : null,
        weight_kg: form.weight ? Number.parseFloat(form.weight) : null,
        fitness_level: form.fitness || null,
        nutrition: form.nutrition || null,
        goals: form.goal ? [form.goal] : [],
        allergies: form.allergies
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
        contraindications: form.contraindications
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
      };
      await updateProfile(data);
      setToast('Профиль сохранён');
      refreshProfile();
    } catch (err) {
      setToast(`Ошибка: ${err.message}`);
    } finally {
      setSaving(false);
      setTimeout(() => setToast(''), 3000);
    }
  };

  if (loading) return <div className='loading'>Загрузка профиля...</div>;

  return (
    <div className='view active profile-view'>
      <form className='profile-form' onSubmit={handleSubmit}>
        <div className='form-section'>
          <h3>Основное</h3>
          <div className='form-group'>
            <label htmlFor='nickname'>Никнейм *</label>
            <input
              type='text'
              id='nickname'
              value={form.nickname}
              onChange={(e) => setField('nickname', e.target.value)}
              placeholder='Ваш никнейм'
              maxLength={30}
              className={errors.nickname ? 'invalid' : ''}
            />
            <div className='field-error'>{errors.nickname || ''}</div>
          </div>
          <div className='form-row'>
            <div className='form-group'>
              <label htmlFor='age'>Возраст</label>
              <input
                type='number'
                id='age'
                value={form.age}
                onChange={(e) => setField('age', e.target.value)}
                placeholder='30'
                min='18'
                max='100'
                inputMode='numeric'
                className={errors.age ? 'invalid' : ''}
              />
              <div className='field-error'>{errors.age || ''}</div>
            </div>
            <div className='form-group'>
              <label htmlFor='gender'>Пол</label>
              <select
                id='gender'
                value={form.gender}
                onChange={(e) => setField('gender', e.target.value)}
              >
                <option value=''>—</option>
                <option value='male'>Мужской</option>
                <option value='female'>Женский</option>
              </select>
            </div>
          </div>
        </div>

        <div className='form-section'>
          <h3>Параметры тела</h3>
          <div className='form-row'>
            <div className='form-group'>
              <label htmlFor='height'>Рост, см</label>
              <input
                type='number'
                id='height'
                value={form.height}
                onChange={(e) => setField('height', e.target.value)}
                placeholder='175'
                min='50'
                max='300'
                inputMode='numeric'
                className={errors.height ? 'invalid' : ''}
              />
              <div className='field-error'>{errors.height || ''}</div>
            </div>
            <div className='form-group'>
              <label htmlFor='weight'>Вес, кг</label>
              <input
                type='number'
                id='weight'
                value={form.weight}
                onChange={(e) => setField('weight', e.target.value)}
                placeholder='70'
                min='20'
                max='500'
                step='0.1'
                inputMode='decimal'
                className={errors.weight ? 'invalid' : ''}
              />
              <div className='field-error'>{errors.weight || ''}</div>
            </div>
          </div>
          {form.height &&
            form.weight &&
            (() => {
              const bmi = calculateBMI(
                Number.parseFloat(form.height),
                Number.parseFloat(form.weight)
              );
              if (bmi) {
                return (
                  <div className='bmi-hint'>
                    <strong>ИМТ:</strong> {bmi.bmi} ({bmi.category})<br />
                    <span style={{ color: 'var(--blue)' }}>
                      {bmi.recommendation}
                    </span>
                  </div>
                );
              }
              return null;
            })()}
        </div>

        <div className='form-section'>
          <h3>Образ жизни</h3>
          <div className='form-group'>
            <label htmlFor='fitness'>Уровень подготовки</label>
            <select
              id='fitness'
              value={form.fitness}
              onChange={(e) => setField('fitness', e.target.value)}
            >
              <option value=''>—</option>
              <option value='beginner'>
                Начинающий (менее 1 тренировки в неделю)
              </option>
              <option value='intermediate'>
                Средний (1-3 тренировки в неделю)
              </option>
              <option value='advanced'>
                Продвинутый (более 3 тренировок в неделю)
              </option>
            </select>
          </div>
          <div className='form-group'>
            <label htmlFor='nutrition'>Тип питания</label>
            <select
              id='nutrition'
              value={form.nutrition}
              onChange={(e) => setField('nutrition', e.target.value)}
            >
              <option value=''>—</option>
              <option value='balanced'>Сбалансированное</option>
              <option value='high_protein'>Высокобелковое</option>
              <option value='vegetarian'>Вегетарианское</option>
              <option value='vegan'>Веганское</option>
              <option value='keto'>Кето</option>
              <option value='paleo'>Палео</option>
            </select>
          </div>
        </div>

        <div className='form-section'>
          <h3>Здоровье и предпочтения</h3>
          <div className='form-group'>
            <label htmlFor='allergies'>
              Аллергии или непереносимость продуктов (через запятую)
            </label>
            <input
              type='text'
              id='allergies'
              value={form.allergies}
              onChange={(e) => setField('allergies', e.target.value)}
              placeholder='Например: орехи, лактоза, глютен'
            />
          </div>
          <div className='form-group'>
            <label htmlFor='contraindications'>
              Медицинские противопоказания (для тренировок)
            </label>
            <input
              type='text'
              id='contraindications'
              value={form.contraindications}
              onChange={(e) => setField('contraindications', e.target.value)}
              placeholder='Например: проблемы с коленями, астма'
            />
          </div>
        </div>

        <div className='form-section'>
          <h3>Основная цель</h3>
          <div className='goals-grid'>
            {['weight_loss', 'muscle_gain', 'endurance', 'flexibility'].map(
              (goal) => (
                <label
                  key={goal}
                  className={`goal-chip ${form.goal === goal ? 'selected' : ''}`}
                >
                  <input
                    type='radio'
                    name='goal'
                    value={goal}
                    checked={form.goal === goal}
                    onChange={(e) => setField('goal', e.target.value)}
                  />
                  {goal === 'weight_loss' && 'Похудение'}
                  {goal === 'muscle_gain' && 'Набор мышц'}
                  {goal === 'endurance' && 'Выносливость'}
                  {goal === 'flexibility' && 'Гибкость'}
                </label>
              )
            )}
          </div>
        </div>

        <button type='submit' className='btn-primary' disabled={saving}>
          {saving ? 'Сохранение...' : 'Сохранить'}
        </button>
      </form>

      <div className='form-section' style={{ marginTop: 24 }}>
        <h3>Безопасность</h3>
        <button
          type='button'
          className='btn-secondary'
          onClick={() => setShowPasswordModal(true)}
          style={{ marginBottom: 12 }}
        >
          Сменить пароль
        </button>
        <button
          type='button'
          className='btn-secondary'
          onClick={() => setShowEmailModal(true)}
          style={{ marginBottom: 12 }}
        >
          Сменить почту
        </button>
        <div
          style={{
            marginTop: 16,
            paddingTop: 16,
            borderTop: '1px solid var(--border)',
          }}
        >
          <h4>Двухфакторная аутентификация</h4>
          <TwoFASetup />
        </div>
      </div>

      <div className='form-section danger-zone' style={{ marginTop: 24 }}>
        <h3 style={{ color: 'var(--red)' }}>Опасная зона</h3>
        <p
          style={{
            fontSize: 13,
            color: 'var(--text-secondary)',
            marginBottom: 12,
          }}
        >
          Удаление профиля необратимо. Все ваши данные, тренировки и достижения
          будут удалены.
        </p>
        <button
          type='button'
          className='btn-danger'
          onClick={() => setShowDeleteModal(true)}
        >
          Удалить аккаунт
        </button>
      </div>

      {toast && <div className='toast success'>{toast}</div>}

      {showPasswordModal && (
        <ChangePasswordModal onClose={() => setShowPasswordModal(false)} />
      )}
      {showEmailModal && (
        <ChangeEmailModal onClose={() => setShowEmailModal(false)} />
      )}
      {showDeleteModal && (
        <DeleteProfileModal onClose={() => setShowDeleteModal(false)} />
      )}
    </div>
  );
}
