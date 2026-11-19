import { beforeEach, describe, expect, it } from 'vitest';
import {
  EN_TEXT,
  getMonthName,
  MONTHS_EN,
  MONTHS_RU,
  setLanguage,
  t,
} from './i18n';

beforeEach(() => {
  localStorage.clear();
  // По умолчанию тесты работают на русском.
  setLanguage('ru');
});

describe('t', () => {
  it('returns russian text by default', () => {
    expect(t('Привет', 'Hello')).toBe('Привет');
  });

  it('returns english after switching language', () => {
    setLanguage('en');
    expect(t('Привет', 'Hello')).toBe('Hello');
    expect(t('Питание', 'Nutrition')).toBe('Nutrition');
  });

  it('switches back to russian', () => {
    setLanguage('en');
    expect(t('Привет', 'Hello')).toBe('Hello');
    setLanguage('ru');
    expect(t('Привет', 'Hello')).toBe('Привет');
  });

  it('persists the selected language to localStorage', () => {
    setLanguage('en');
    expect(localStorage.getItem('fitpulse_lang')).toBe('en');
    setLanguage('ru');
    expect(localStorage.getItem('fitpulse_lang')).toBe('ru');
  });

  it('keeps russian when localStorage is unavailable', () => {
    const orig = globalThis.localStorage;
    delete globalThis.localStorage;
    try {
      setLanguage('ru');
      expect(t('Привет', 'Hello')).toBe('Привет');
      expect(t('Питание', 'Nutrition')).toBe('Питание');
    } finally {
      globalThis.localStorage = orig;
    }
  });

  it('normalizes unknown language codes to russian', () => {
    setLanguage('fr');
    expect(t('Привет', 'Hello')).toBe('Привет');
  });
});

describe('getMonthName', () => {
  it('returns localized month names', () => {
    expect(getMonthName(0, 'en')).toBe('January');
    expect(getMonthName(0, 'ru')).toBe('Январь');
    expect(getMonthName(6, 'en')).toBe('July');
    expect(getMonthName(6, 'ru')).toBe('Июль');
    expect(getMonthName(11)).toBe('Декабрь');
  });

  it('returns an empty string for unknown indexes', () => {
    expect(getMonthName(12, 'en')).toBe('');
    expect(getMonthName(12, 'ru')).toBe('');
    expect(getMonthName(-1, 'en')).toBe('');
  });
});

describe('setLanguage', () => {
  it('stores the selected language', () => {
    setLanguage('en');
    expect(localStorage.getItem('fitpulse_lang')).toBe('en');
    setLanguage('ru');
    expect(localStorage.getItem('fitpulse_lang')).toBe('ru');
  });
});

describe('translation data', () => {
  it('exposes month lists', () => {
    expect(MONTHS_RU).toHaveLength(12);
    expect(MONTHS_EN).toHaveLength(12);
    expect(MONTHS_RU[0]).toBe('Январь');
    expect(MONTHS_EN[0]).toBe('January');
  });

  it('exposes english translations', () => {
    expect(EN_TEXT.Привет).toBe('Hello');
    expect(EN_TEXT.Главная).toBe('Home');
  });
});
