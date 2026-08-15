import { describe, expect, it } from 'vitest';
import { EXERCISE_NAME_MAP } from './exerciseNames';

describe('EXERCISE_NAME_MAP', () => {
  it('exports exercise name map constant', () => {
    expect(EXERCISE_NAME_MAP).toBeDefined();
    expect(typeof EXERCISE_NAME_MAP).toBe('object');
  });

  it('module exports a non-empty object', () => {
    expect(Object.keys(EXERCISE_NAME_MAP).length).toBeGreaterThan(0);
  });

  it('contains expected exercise keys', () => {
    expect(EXERCISE_NAME_MAP.jumping_jacks).toBe('Прыжки на месте');
    expect(EXERCISE_NAME_MAP.pushups).toBe('Отжимания');
    expect(EXERCISE_NAME_MAP.squats).toBe('Приседания');
    expect(EXERCISE_NAME_MAP.running).toBe('Бег');
    expect(EXERCISE_NAME_MAP.cycling).toBe('Велосипед');
  });

  it('has non-empty values for all entries', () => {
    for (const [key, value] of Object.entries(EXERCISE_NAME_MAP)) {
      expect(value).toBeTruthy();
      expect(typeof key).toBe('string');
    }
  });
});
