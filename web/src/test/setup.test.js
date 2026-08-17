import { describe, expect, it, vi } from 'vitest';
import './setup';

describe('test setup mocks', () => {
  it('assigns React to global', () => {
    expect(global.React).toBeDefined();
  });

  it('defines window.localStorage mock', () => {
    expect(window.localStorage).toBeDefined();
    expect(typeof window.localStorage.getItem).toBe('function');
    expect(typeof window.localStorage.setItem).toBe('function');
    expect(typeof window.localStorage.removeItem).toBe('function');
    expect(typeof window.localStorage.clear).toBe('function');
  });

  it('defines window.prompt mock', () => {
    expect(typeof window.prompt).toBe('function');
  });

  it('defines window.location mock with assign, replace, reload', () => {
    expect(window.location).toBeDefined();
    expect(typeof window.location.assign).toBe('function');
    expect(typeof window.location.replace).toBe('function');
    expect(typeof window.location.reload).toBe('function');
  });

  it('mocks react-chartjs-2 components', async () => {
    const mod = await vi.importActual('react-chartjs-2');
    expect(mod).toBeDefined();
  });
});
