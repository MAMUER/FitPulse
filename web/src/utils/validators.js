export function validateEmail(v) {
  if (!v) return 'Введите email';
  if (v.length > 254) return 'Email слишком длинный';
  if (!v.includes('@')) return 'Некорректный формат email';
  if (/\s/.test(v)) return 'Некорректный формат email';
  const [localPart, domain] = v.split('@');
  if (!localPart || !domain) return 'Некорректный формат email';
  if (!domain.includes('.')) return 'Некорректный формат email';
  const domainParts = domain.split('.');
  if (!domainParts.every((part) => part.length > 0)) {
    return 'Некорректный формат email';
  }
  return '';
}

export function validateLoginPassword(v) {
  if (!v) return 'Введите пароль';
  return '';
}

export function validatePassword(v) {
  const checks = {
    length: v.length >= 8,
    upper: /[A-ZА-ЯЁ]/.test(v),
    lower: /[a-zа-яё]/.test(v),
    digit: /\d/.test(v),
  };
  if (!v) return { error: 'Введите пароль', checks };
  if (!checks.length) return { error: 'Минимум 8 символов', checks };
  return { error: '', checks };
}

export function validateName(v) {
  if (!v) return 'Введите имя';
  if (v.length < 2) return 'Минимум 2 символа';
  if (v.length > 100) return 'Максимум 100 символов';
  if (!/^[A-Za-zА-Яа-яЁё\s-]+$/.test(v)) return 'Только буквы';
  return '';
}

export function validateNickname(v) {
  if (!v?.trim()) return 'Никнейм обязателен';
  if (v.trim().length < 2) return 'Минимум 2 символа';
  if (v.trim().length > 30) return 'Максимум 30 символов';
  if (!/^[A-Za-zА-Яа-яЁё0-9_\s-]+$/.test(v.trim()))
    return 'Только буквы, цифры, _ и -';
  return '';
}

export function validateAge(v) {
  if (!v) return '';
  if (!/^\d+$/.test(v)) return 'Только целые цифры';
  const n = Number.parseInt(v, 10);
  if (n < 18 || n > 100) return 'От 18 до 100';
  return '';
}

export function validateHeight(v) {
  if (!v) return '';
  if (!/^\d+$/.test(v)) return 'Только целые цифры';
  const n = Number.parseInt(v, 10);
  if (n < 50 || n > 300) return 'От 50 до 300 см';
  return '';
}

export function validateWeight(v) {
  if (!v) return '';
  if (!/^\d+(\.\d{1,2})?$/.test(v)) return 'Число (например, 70.5)';
  const n = Number.parseFloat(v);
  if (n < 20 || n > 500) return 'От 20 до 500 кг';
  return '';
}

export function calculateBMI(heightCm, weightKg) {
  if (heightCm > 0 && weightKg > 0) {
    const heightM = heightCm / 100;
    const bmi = weightKg / (heightM * heightM);
    let category = '';
    let recommendation = '';
    let recommendedGoal = '';

    if (bmi < 18.5) {
      category = 'Недостаточный вес';
      recommendation = 'Рекомендуется набор мышечной массы.';
      recommendedGoal = 'muscle_gain';
    } else if (bmi < 25) {
      category = 'Нормальный вес';
      recommendation = 'Ваш вес в норме. Можно выбрать любую цель.';
    } else if (bmi < 30) {
      category = 'Избыточный вес';
      recommendation = 'Рекомендуется снижение веса.';
      recommendedGoal = 'weight_loss';
    } else {
      category = 'Ожирение';
      recommendation = 'Настоятельно рекомендуется снижение веса.';
      recommendedGoal = 'weight_loss';
    }

    return { bmi: bmi.toFixed(1), category, recommendation, recommendedGoal };
  }
  return null;
}
