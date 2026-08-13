import { describe, it, expect } from 'vitest';
import { EXERCISE_NAME_MAP } from './exerciseNames';

describe('EXERCISE_NAME_MAP', () => {
  it('exports exercise name mappings', () => {
    expect(EXERCISE_NAME_MAP).toBeDefined();
    expect(typeof EXERCISE_NAME_MAP).toBe('object');
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
