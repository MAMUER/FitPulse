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
});

describe('t', () => {
  it('returns russian text by default', () => {
    expect(t('Привет', 'Hello')).toBe('Привет');
  });

  it('uses the saved fallback language', () => {
    setLanguage('en');
    expect(localStorage.getItem('fitpulse_lang')).toBe('en');
    expect(t('Привет', 'Hello')).toBe('Hello');
    expect(t('Питание', 'Nutrition')).toBe('Nutrition');
  });

  it('uses the language from merged state', () => {
    localStorage.setItem(
      'fitpulse-merged-v9',
      JSON.stringify({ language: 'en', screen: 'home' })
    );
    setLanguage('ru');
    expect(t('Привет', 'Hello')).toBe('Hello');
  });

  it('keeps russian when merged state has no language', () => {
    localStorage.setItem(
      'fitpulse-merged-v9',
      JSON.stringify({ screen: 'home' })
    );
    expect(t('Привет', 'Hello')).toBe('Привет');
  });

  it('falls back to saved language when merged state is invalid', () => {
    localStorage.setItem('fitpulse-merged-v9', '{not-json');
    setLanguage('en');
    expect(t('Привет', 'Hello')).toBe('Hello');
  });

  it('keeps russian when merged state is invalid and no fallback exists', () => {
    localStorage.setItem('fitpulse-merged-v9', '{not-json');
    expect(t('Привет', 'Hello')).toBe('Привет');
  });

  it('uses ru when localStorage is unavailable', () => {
    const orig = globalThis.localStorage;
    // eslint-disable-next-line no-unused-expressions
    delete globalThis.localStorage;
    try {
      expect(t('Привет', 'Hello')).toBe('Привет');
      expect(t('Питание', 'Nutrition')).toBe('Питание');
    } finally {
      globalThis.localStorage = orig;
    }
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
