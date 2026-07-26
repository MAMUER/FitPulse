import { describe, expect, it } from 'vitest';
import {
  calculateBMI,
  validateAge,
  validateEmail,
  validateHeight,
  validateName,
  validateNickname,
  validatePassword,
  validateWeight,
} from './validators';

describe('validators', () => {
  describe('validateEmail', () => {
    it('returns error for invalid email', () => {
      expect(validateEmail('invalid')).toBe('Некорректный формат email');
      expect(validateEmail('test@')).toBe('Некорректный формат email');
      expect(validateEmail('@domain.com')).toBe('Некорректный формат email');
    });

    it('returns empty string for valid email', () => {
      expect(validateEmail('test@example.com')).toBe('');
      expect(validateEmail('user.name@domain.org')).toBe('');
    });
  });

  describe('validatePassword', () => {
    it('returns error for empty password', () => {
      const result = validatePassword('');
      expect(result.error).toBe('Введите пароль');
    });

    it('returns error for short password', () => {
      const result = validatePassword('123');
      expect(result.error).toBe('Минимум 8 символов');
    });

    it('returns checks object for password without uppercase', () => {
      const result = validatePassword('password123');
      expect(result.error).toBe('');
      expect(result.checks.upper).toBe(false);
      expect(result.checks.length).toBe(true);
    });

    it('returns checks object for password without lowercase', () => {
      const result = validatePassword('PASSWORD123');
      expect(result.error).toBe('');
      expect(result.checks.lower).toBe(false);
    });

    it('returns checks object for password without number', () => {
      const result = validatePassword('Password');
      expect(result.error).toBe('');
      expect(result.checks.digit).toBe(false);
    });

    it('returns success for valid password', () => {
      const result = validatePassword('Password123');
      expect(result.error).toBe('');
      expect(result.checks.length).toBe(true);
      expect(result.checks.upper).toBe(true);
      expect(result.checks.lower).toBe(true);
      expect(result.checks.digit).toBe(true);
    });
  });

  describe('validateName', () => {
    it('returns error for empty name', () => {
      expect(validateName('')).toBe('Введите имя');
    });

    it('returns error for too short name', () => {
      expect(validateName('A')).toBe('Минимум 2 символа');
    });

    it('returns error for too long name', () => {
      expect(validateName('A'.repeat(101))).toBe('Максимум 100 символов');
    });

    it('returns error for name with numbers', () => {
      expect(validateName('John123')).toBe('Только буквы');
    });

    it('returns empty string for valid name', () => {
      expect(validateName('Иван')).toBe('');
      expect(validateName('John')).toBe('');
    });
  });

  describe('validateNickname', () => {
    it('returns error for empty nickname', () => {
      expect(validateNickname('')).toBe('Никнейм обязателен');
      expect(validateNickname('   ')).toBe('Никнейм обязателен');
    });

    it('returns error for too short nickname', () => {
      expect(validateNickname('A')).toBe('Минимум 2 символа');
    });

    it('returns error for too long nickname', () => {
      expect(validateNickname('A'.repeat(31))).toBe('Максимум 30 символов');
    });

    it('returns empty string for valid nickname', () => {
      expect(validateNickname('john_doe')).toBe('');
      expect(validateNickname('User123')).toBe('');
    });
  });

  describe('validateAge', () => {
    it('returns empty string for empty value', () => {
      expect(validateAge('')).toBe('');
    });

    it('returns error for non-numeric value', () => {
      expect(validateAge('abc')).toBe('Только целые цифры');
    });

    it('returns error for age below 18', () => {
      expect(validateAge('17')).toBe('От 18 до 100');
    });

    it('returns error for age above 100', () => {
      expect(validateAge('101')).toBe('От 18 до 100');
    });

    it('returns empty string for valid age', () => {
      expect(validateAge('25')).toBe('');
      expect(validateAge('50')).toBe('');
    });
  });

  describe('validateHeight', () => {
    it('returns empty string for empty value', () => {
      expect(validateHeight('')).toBe('');
    });

    it('returns error for non-numeric value', () => {
      expect(validateHeight('abc')).toBe('Только целые цифры');
    });

    it('returns error for height below 50', () => {
      expect(validateHeight('49')).toBe('От 50 до 300 см');
    });

    it('returns error for height above 300', () => {
      expect(validateHeight('301')).toBe('От 50 до 300 см');
    });

    it('returns empty string for valid height', () => {
      expect(validateHeight('175')).toBe('');
      expect(validateHeight('180')).toBe('');
    });
  });

  describe('validateWeight', () => {
    it('returns empty string for empty value', () => {
      expect(validateWeight('')).toBe('');
    });

    it('returns error for invalid number format', () => {
      expect(validateWeight('abc')).toBe('Число (например, 70.5)');
    });

    it('returns error for weight below 20', () => {
      expect(validateWeight('19')).toBe('От 20 до 500 кг');
    });

    it('returns error for weight above 500', () => {
      expect(validateWeight('501')).toBe('От 20 до 500 кг');
    });

    it('returns empty string for valid weight', () => {
      expect(validateWeight('70.5')).toBe('');
      expect(validateWeight('80')).toBe('');
    });
  });

  describe('calculateBMI', () => {
    it('returns null for invalid input', () => {
      expect(calculateBMI(0, 70)).toBeNull();
      expect(calculateBMI(175, 0)).toBeNull();
      expect(calculateBMI(0, 0)).toBeNull();
    });

    it('calculates BMI correctly for normal weight', () => {
      const result = calculateBMI(175, 70);
      expect(result).not.toBeNull();
      expect(result.bmi).toBe('22.9');
      expect(result.category).toBe('Нормальный вес');
    });

    it('calculates BMI correctly for underweight', () => {
      const result = calculateBMI(175, 50);
      expect(result).not.toBeNull();
      expect(result.category).toBe('Недостаточный вес');
      expect(result.recommendedGoal).toBe('muscle_gain');
    });

    it('calculates BMI correctly for overweight', () => {
      const result = calculateBMI(175, 90);
      expect(result).not.toBeNull();
      expect(result.category).toBe('Избыточный вес');
      expect(result.recommendedGoal).toBe('weight_loss');
    });

    it('calculates BMI correctly for obesity', () => {
      const result = calculateBMI(175, 110);
      expect(result).not.toBeNull();
      expect(result.category).toBe('Ожирение');
      expect(result.recommendedGoal).toBe('weight_loss');
    });
  });
});
