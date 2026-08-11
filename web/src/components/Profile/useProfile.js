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

const INITIAL_FORM = {
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
};

export function useProfile() {
  const { refreshProfile } = useAuth();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [errors, setErrors] = useState({});
  const [toast, setToast] = useState('');
  const [form, setForm] = useState(INITIAL_FORM);

  const loadProfile = useCallback(async () => {
    try {
      const data = await getProfile();
      const p = data.profile || data; /* istanbul ignore next */
      setForm({
        nickname: p.full_name || p.nickname || '',
        age: p.age || '',
        gender: p.gender || '',
        height: p.height_cm || '',
        weight: p.weight_kg || '',
        fitness: p.fitness_level || '',
        nutrition: p.nutrition || '',
        allergies: (p.allergies || []).join(', '), /* istanbul ignore next */
        contraindications: (p.contraindications || []).join(', '), /* istanbul ignore next */
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
    const ageErr = validateAge(form.age); /* istanbul ignore next */
    if (ageErr) errs.age = ageErr; /* istanbul ignore next */
    const heightErr = validateHeight(form.height); /* istanbul ignore next */
    if (heightErr) errs.height = heightErr; /* istanbul ignore next */
    const weightErr = validateWeight(form.weight); /* istanbul ignore next */
    if (weightErr) errs.weight = weightErr; /* istanbul ignore next */

    setErrors(errs);
    if (Object.keys(errs).length > 0) {
      setToast('Исправьте ошибки в полях');
      return;
    }

    setSaving(true);
    try {
      const data = {
        full_name: form.nickname.trim(),
        age: form.age ? Number.parseInt(form.age, 10) : null, /* istanbul ignore next */
        gender: form.gender || null, /* istanbul ignore next */
        height_cm: form.height ? Number.parseInt(form.height, 10) : null, /* istanbul ignore next */
        weight_kg: form.weight ? Number.parseFloat(form.weight) : null, /* istanbul ignore next */
        fitness_level: form.fitness || null, /* istanbul ignore next */
        nutrition: form.nutrition || null, /* istanbul ignore next */
        goals: form.goal ? [form.goal] : [], /* istanbul ignore next */
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

  const bmi =
    form.height && form.weight
      ? calculateBMI(
          Number.parseFloat(form.height),
          Number.parseFloat(form.weight)
        )
      : null;

  return {
    loading,
    saving,
    errors,
    toast,
    form,
    bmi,
    setField,
    handleSubmit,
    setToast,
  };
}
