import { beforeEach, describe, expect, it, vi } from 'vitest';
import { backendRequest } from '../utils/backendRequest';
import * as api from './api';

vi.mock('../utils/backendRequest', () => ({ backendRequest: vi.fn() }));

const mocked = vi.mocked(backendRequest);

beforeEach(() => {
  mocked.mockClear();
  mocked.mockResolvedValue({ ok: true });
});

describe('auth', () => {
  it('login posts credentials and returns data', async () => {
    mocked.mockResolvedValue({ token: 'abc' });
    await expect(api.login('user@example.com', 'secret1')).resolves.toEqual({
      token: 'abc',
    });
    expect(mocked).toHaveBeenCalledWith('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email: 'user@example.com', password: 'secret1' }),
    });
  });

  it('login propagates errors', async () => {
    mocked.mockRejectedValue(new Error('Backend 500'));
    await expect(api.login('user@example.com', 'secret1')).rejects.toThrow(
      'Backend 500'
    );
  });

  it('logout posts and swallows errors', async () => {
    await expect(api.logout()).resolves.toBeUndefined();
    expect(mocked).toHaveBeenCalledWith('/api/v1/logout', { method: 'POST' });

    mocked.mockClear();
    mocked.mockRejectedValue(new Error('Backend 500'));
    await expect(api.logout()).resolves.toBeUndefined();
  });

  it('register validates fields', () => {
    expect(api.register('a@b.com', 'secret1', 'secret1')).toEqual({
      email: 'a@b.com',
      password: 'secret1',
      confirm: 'secret1',
    });
    expect(() => api.register('', 'secret1', 'secret1')).toThrow(
      'Заполните все поля'
    );
    expect(() => api.register('a@b.com', '', 'secret1')).toThrow(
      'Заполните все поля'
    );
    expect(() => api.register('a@b.com', 'secret1', '')).toThrow(
      'Заполните все поля'
    );
    expect(() => api.register('a@b.com', 'secret1', 'secret2')).toThrow(
      'Пароли не совпадают'
    );
    expect(() => api.register('a@b.com', '12345', '12345')).toThrow(
      'Пароль должен содержать минимум 6 символов'
    );
    expect(mocked).not.toHaveBeenCalled();
  });

  it('register accepts a six character password', () => {
    expect(api.register('a@b.com', '123456', '123456')).toEqual({
      email: 'a@b.com',
      password: '123456',
      confirm: '123456',
    });
  });

  it('forgotPassword validates email', () => {
    expect(api.forgotPassword('a@b.com')).toEqual({ email: 'a@b.com' });
    expect(() => api.forgotPassword('')).toThrow('Введите корректный email');
    expect(() => api.forgotPassword('ab.com')).toThrow(
      'Введите корректный email'
    );
    expect(() => api.forgotPassword()).toThrow('Введите корректный email');
  });

  it('submitResetEmail returns data', async () => {
    mocked.mockResolvedValue({ sent: true });
    await expect(api.submitResetEmail('a@b.com')).resolves.toEqual({
      sent: true,
    });
    expect(mocked).toHaveBeenCalledWith('/api/v1/auth/forgot-password', {
      method: 'POST',
      body: JSON.stringify({ email: 'a@b.com' }),
    });
  });

  it('submitResetEmail propagates errors', async () => {
    mocked.mockRejectedValue(new Error('Backend 500'));
    await expect(api.submitResetEmail('a@b.com')).rejects.toThrow(
      'Backend 500'
    );
  });

  it('submitResetCode validates code', () => {
    expect(api.submitResetCode('123456')).toEqual({ code: '123456' });
    expect(() => api.submitResetCode('')).toThrow('Введите код');
  });

  it('submitNewPassword validates and posts', async () => {
    mocked.mockResolvedValue({ done: true });
    await expect(
      api.submitNewPassword('a@b.com', '123456', 'newsecret')
    ).resolves.toEqual({ done: true });
    expect(mocked).toHaveBeenCalledWith('/api/v1/auth/reset', {
      method: 'POST',
      body: JSON.stringify({
        email: 'a@b.com',
        code: '123456',
        new_password: 'newsecret',
      }),
    });
    await expect(
      api.submitNewPassword('a@b.com', '123456', '12345')
    ).rejects.toThrow('Пароль должен содержать минимум 6 символов');
  });

  it('continueAsGuest returns guest payload', () => {
    expect(api.continueAsGuest()).toEqual({ guest: true });
  });

  it('socialLogin supports yandex only', () => {
    expect(api.socialLogin('yandex')).toEqual({
      redirect: '/api/v1/auth/yandex',
    });
    expect(() => api.socialLogin('google')).toThrow(
      'Интеграция google не поддерживается'
    );
  });

  it('confirmEmail posts token', async () => {
    mocked.mockResolvedValue({ confirmed: true });
    await expect(api.confirmEmail('tok')).resolves.toEqual({ confirmed: true });
    expect(mocked).toHaveBeenCalledWith('/api/v1/auth/confirm', {
      method: 'POST',
      body: JSON.stringify({ token: 'tok' }),
    });
  });
});

describe('two factor', () => {
  it('setup2FA posts and returns data', async () => {
    mocked.mockResolvedValue({ secret: 's' });
    await expect(api.setup2FA()).resolves.toEqual({ secret: 's' });
    expect(mocked).toHaveBeenCalledWith('/api/v1/auth/2fa/setup', {
      method: 'POST',
    });
  });

  it('confirm2FA posts passcode payload', async () => {
    mocked.mockResolvedValue({ ok: true });
    await expect(
      api.confirm2FA('111111', 'secret', ['code-1'])
    ).resolves.toEqual({ ok: true });
    expect(mocked).toHaveBeenCalledWith('/api/v1/auth/2fa/confirm', {
      method: 'POST',
      body: JSON.stringify({
        passcode: '111111',
        temp_secret: 'secret',
        backup_codes: ['code-1'],
      }),
    });
  });

  it('verify2FA defaults isBackupCode to false', async () => {
    mocked.mockResolvedValue({ ok: true });
    await expect(api.verify2FA('temp', '111111')).resolves.toEqual({
      ok: true,
    });
    expect(mocked).toHaveBeenCalledWith('/api/v1/auth/2fa/verify', {
      method: 'POST',
      body: JSON.stringify({
        temp_token: 'temp',
        passcode: '111111',
        is_backup_code: false,
      }),
    });
  });

  it('verify2FA accepts backup code flag', async () => {
    await api.verify2FA('temp', 'code-1', true);
    expect(mocked).toHaveBeenCalledWith('/api/v1/auth/2fa/verify', {
      method: 'POST',
      body: JSON.stringify({
        temp_token: 'temp',
        passcode: 'code-1',
        is_backup_code: true,
      }),
    });
  });

  it('disable2FA posts passcode', async () => {
    await api.disable2FA('111111');
    expect(mocked).toHaveBeenCalledWith('/api/v1/auth/2fa/disable', {
      method: 'POST',
      body: JSON.stringify({ passcode: '111111' }),
    });
  });

  it('load2FAStatus gets status', async () => {
    mocked.mockResolvedValue({ enabled: false });
    await expect(api.load2FAStatus()).resolves.toEqual({ enabled: false });
    expect(mocked).toHaveBeenCalledWith('/api/v1/auth/2fa/status');
  });
});

describe('profile', () => {
  it('loadProfile returns data', async () => {
    mocked.mockResolvedValue({ name: 'Михаил' });
    await expect(api.loadProfile()).resolves.toEqual({ name: 'Михаил' });
    expect(mocked).toHaveBeenCalledWith('/api/v1/profile');
  });

  it('saveProfile puts profile data', async () => {
    mocked.mockResolvedValue({ saved: true });
    await expect(api.saveProfile({ name: 'Тест' })).resolves.toEqual({
      saved: true,
    });
    expect(mocked).toHaveBeenCalledWith('/api/v1/profile', {
      method: 'PUT',
      body: JSON.stringify({ name: 'Тест' }),
    });
  });

  it('changePassword validates input', () => {
    expect(api.changePassword('old', 'newsecret', 'newsecret')).toEqual({
      oldPass: 'old',
      newPass: 'newsecret',
    });
    expect(() => api.changePassword('', 'newsecret', 'newsecret')).toThrow(
      'Введите старый пароль'
    );
    expect(() => api.changePassword('old', '12345', '12345')).toThrow(
      'Пароль должен содержать минимум 6 символов'
    );
    expect(() => api.changePassword('old', 'newsecret', 'other')).toThrow(
      'Пароли не совпадают'
    );
  });

  it('deleteAccount clears merged storage and returns true', async () => {
    localStorage.setItem('fitpulse-merged-v9', '{"screen":"home"}');
    await expect(api.deleteAccount()).resolves.toBe(true);
    expect(localStorage.getItem('fitpulse-merged-v9')).toBeNull();
  });
});

describe('biometrics and health', () => {
  it('loadBiometrics returns data', async () => {
    mocked.mockResolvedValue([{ value: 70 }]);
    await expect(api.loadBiometrics()).resolves.toEqual([{ value: 70 }]);
    expect(mocked).toHaveBeenCalledWith('/api/v1/biometrics');
  });

  it('addBiometric defaults device type to manual', async () => {
    await api.addBiometric('weight', 74.2);
    const [, init] = mocked.mock.calls[0];
    expect(JSON.parse(init.body)).toEqual({
      metric_type: 'weight',
      value: 74.2,
      timestamp: expect.any(String),
      device_type: 'manual',
    });
  });

  it('addBiometric accepts device type', async () => {
    await api.addBiometric('pulse', 72, 'watch');
    const [, init] = mocked.mock.calls[0];
    expect(JSON.parse(init.body).device_type).toBe('watch');
  });

  it('loadBodyComposition and saveBodyComposition', async () => {
    mocked.mockResolvedValue({ ok: true });
    await expect(api.loadBodyComposition()).resolves.toEqual({ ok: true });
    expect(mocked).toHaveBeenCalledWith('/api/v1/health/body-composition');

    mocked.mockClear();
    await api.saveBodyComposition({ fat: 18.4 });
    expect(mocked).toHaveBeenCalledWith('/api/v1/health/body-composition', {
      method: 'POST',
      body: JSON.stringify({ fat: 18.4 }),
    });
  });

  it('loadConditions, createCondition, deleteCondition', async () => {
    mocked.mockResolvedValue({ ok: true });
    await expect(api.loadConditions()).resolves.toEqual({ ok: true });
    expect(mocked).toHaveBeenCalledWith('/api/v1/health/conditions');

    mocked.mockClear();
    await api.createCondition({ name: 'Астма' });
    expect(mocked).toHaveBeenCalledWith('/api/v1/health/conditions', {
      method: 'POST',
      body: JSON.stringify({ name: 'Астма' }),
    });

    mocked.mockClear();
    await api.deleteCondition('c1');
    expect(mocked).toHaveBeenCalledWith('/api/v1/health/conditions/c1', {
      method: 'DELETE',
    });
  });

  it('menstrual cycles CRUD', async () => {
    mocked.mockResolvedValue({ ok: true });
    await expect(api.loadMenstrualCycles()).resolves.toEqual({ ok: true });
    expect(mocked).toHaveBeenCalledWith('/api/v1/health/menstrual-cycles');

    mocked.mockClear();
    await api.createMenstrualCycle({ start: '2026-01-01' });
    expect(mocked).toHaveBeenCalledWith('/api/v1/health/menstrual-cycles', {
      method: 'POST',
      body: JSON.stringify({ start: '2026-01-01' }),
    });

    mocked.mockClear();
    await api.updateMenstrualCycle('m1', { start: '2026-02-01' });
    expect(mocked).toHaveBeenCalledWith('/api/v1/health/menstrual-cycles/m1', {
      method: 'PUT',
      body: JSON.stringify({ start: '2026-02-01' }),
    });

    mocked.mockClear();
    await api.deleteMenstrualCycle('m1');
    expect(mocked).toHaveBeenCalledWith('/api/v1/health/menstrual-cycles/m1', {
      method: 'DELETE',
    });
  });
});

describe('training', () => {
  it('loadTrainingPlans returns data', async () => {
    mocked.mockResolvedValue([{ id: 1 }]);
    await expect(api.loadTrainingPlans()).resolves.toEqual([{ id: 1 }]);
    expect(mocked).toHaveBeenCalledWith('/api/v1/training/plans');
  });

  it('generatePlan sends defaults', async () => {
    mocked.mockResolvedValue({ plan: 'p1' });
    await expect(api.generatePlan()).resolves.toEqual({ plan: 'p1' });
    expect(mocked).toHaveBeenCalledWith('/api/v1/training/generate', {
      method: 'POST',
      body: JSON.stringify({
        duration_weeks: 4,
        available_days: [1, 3, 5],
        class: 'endurance_basic',
        confidence: 0.8,
      }),
    });
  });

  it('generatePlan sends custom params', async () => {
    await api.generatePlan({
      durationWeeks: 8,
      availableDays: [2, 4],
      class: 'power_pro',
      confidence: 0.5,
    });
    expect(mocked).toHaveBeenCalledWith('/api/v1/training/generate', {
      method: 'POST',
      body: JSON.stringify({
        duration_weeks: 8,
        available_days: [2, 4],
        class: 'power_pro',
        confidence: 0.5,
      }),
    });
  });

  it('getPlanDetails uses plan id in path', async () => {
    mocked.mockResolvedValue({ id: 42 });
    await expect(api.getPlanDetails(42)).resolves.toEqual({ id: 42 });
    expect(mocked).toHaveBeenCalledWith('/api/v1/training/plans/42');
  });

  it('loadProgress returns data', async () => {
    mocked.mockResolvedValue({ done: 4 });
    await expect(api.loadProgress()).resolves.toEqual({ done: 4 });
    expect(mocked).toHaveBeenCalledWith('/api/v1/training/progress');
  });

  it('completeWorkout sends defaults', async () => {
    mocked.mockResolvedValue({ xp: 40 });
    await expect(api.completeWorkout(1, 2)).resolves.toEqual({ xp: 40 });
    expect(mocked).toHaveBeenCalledWith('/api/v1/training/complete', {
      method: 'POST',
      body: JSON.stringify({
        plan_id: 1,
        workout_id: 2,
        rating: 5,
        feedback: '',
      }),
    });
  });

  it('completeWorkout sends rating and feedback', async () => {
    await api.completeWorkout(1, 2, 4, 'Отлично');
    expect(mocked).toHaveBeenCalledWith('/api/v1/training/complete', {
      method: 'POST',
      body: JSON.stringify({
        plan_id: 1,
        workout_id: 2,
        rating: 4,
        feedback: 'Отлично',
      }),
    });
  });

  it('loadAchievements returns data', async () => {
    mocked.mockResolvedValue([{ name: 'a' }]);
    await expect(api.loadAchievements()).resolves.toEqual([{ name: 'a' }]);
    expect(mocked).toHaveBeenCalledWith('/api/v1/achievements');
  });

  it('loadVideos returns data', async () => {
    mocked.mockResolvedValue([{ id: 1 }]);
    await expect(api.loadVideos()).resolves.toEqual([{ id: 1 }]);
    expect(mocked).toHaveBeenCalledWith('/api/v1/videos');
  });
});

describe('nutrition', () => {
  it('loadMeals returns data', async () => {
    mocked.mockResolvedValue([{ name: 'Овсянка' }]);
    await expect(api.loadMeals()).resolves.toEqual([{ name: 'Овсянка' }]);
    expect(mocked).toHaveBeenCalledWith('/api/v1/nutrition/meals');
  });

  it('createMeal uses provided time', async () => {
    mocked.mockResolvedValue({ id: 1 });
    await expect(api.createMeal('Овсянка', 360, '08:00')).resolves.toEqual({
      id: 1,
    });
    expect(mocked).toHaveBeenCalledWith('/api/v1/nutrition/meals', {
      method: 'POST',
      body: JSON.stringify({ name: 'Овсянка', calories: 360, time: '08:00' }),
    });
  });

  it('createMeal falls back to current time', async () => {
    await api.createMeal('Овсянка', 360);
    const [, init] = mocked.mock.calls[0];
    const body = JSON.parse(init.body);
    expect(body.name).toBe('Овсянка');
    expect(body.calories).toBe(360);
    expect(body.time).toMatch(/^\d{1,2}:\d{2}/);
  });

  it('removeMeal uses meal id in path', async () => {
    mocked.mockResolvedValue({ removed: true });
    await expect(api.removeMeal('m1')).resolves.toEqual({ removed: true });
    expect(mocked).toHaveBeenCalledWith('/api/v1/nutrition/meals/m1', {
      method: 'DELETE',
    });
  });
});

describe('calendar', () => {
  it('loadCalendarEvents returns data', async () => {
    mocked.mockResolvedValue([{ id: 1 }]);
    await expect(api.loadCalendarEvents()).resolves.toEqual([{ id: 1 }]);
    expect(mocked).toHaveBeenCalledWith('/api/v1/calendar/events');
  });

  it('createEvent posts event data', async () => {
    await api.createEvent({ title: 'Тренировка' });
    expect(mocked).toHaveBeenCalledWith('/api/v1/calendar/events', {
      method: 'POST',
      body: JSON.stringify({ title: 'Тренировка' }),
    });
  });

  it('updateEvent uses event id in path', async () => {
    await api.updateEvent('e1', 'Тренировка', '24 июля', 'training', 'Ноги');
    expect(mocked).toHaveBeenCalledWith('/api/v1/calendar/events/e1', {
      method: 'PUT',
      body: JSON.stringify({
        title: 'Тренировка',
        date: '24 июля',
        type: 'training',
        description: 'Ноги',
      }),
    });
  });

  it('deleteCalendarEvent uses event id in path', async () => {
    await api.deleteCalendarEvent('e1');
    expect(mocked).toHaveBeenCalledWith('/api/v1/calendar/events/e1', {
      method: 'DELETE',
    });
  });
});

describe('ai, survey and integrations', () => {
  it('sendAIMessage posts message', async () => {
    mocked.mockResolvedValue({ reply: 'Привет' });
    await expect(api.sendAIMessage('Привет')).resolves.toEqual({
      reply: 'Привет',
    });
    expect(mocked).toHaveBeenCalledWith('/api/v1/chat', {
      method: 'POST',
      body: JSON.stringify({ message: 'Привет' }),
    });
  });

  it('sendAIMessage propagates errors', async () => {
    mocked.mockRejectedValue(new Error('Backend 500'));
    await expect(api.sendAIMessage('Привет')).rejects.toThrow('Backend 500');
  });

  it('quickAI is sendAIMessage', () => {
    expect(api.quickAI).toBe(api.sendAIMessage);
  });

  it('saveSurvey and loadSurvey', async () => {
    mocked.mockResolvedValue({ ok: true });
    await expect(api.saveSurvey({ goals: [] })).resolves.toEqual({ ok: true });
    expect(mocked).toHaveBeenCalledWith('/api/v1/survey', {
      method: 'POST',
      body: JSON.stringify({ goals: [] }),
    });

    mocked.mockClear();
    await expect(api.loadSurvey()).resolves.toEqual({ ok: true });
    expect(mocked).toHaveBeenCalledWith('/api/v1/survey');
  });

  it('loadIntegrationProviders returns data', async () => {
    mocked.mockResolvedValue([{ id: 'apple' }]);
    await expect(api.loadIntegrationProviders()).resolves.toEqual([
      { id: 'apple' },
    ]);
    expect(mocked).toHaveBeenCalledWith('/api/v1/integrations/providers');
  });

  it('disconnectIntegration uses source in path', async () => {
    mocked.mockResolvedValue({ ok: true });
    await expect(api.disconnectIntegration('apple')).resolves.toEqual({
      ok: true,
    });
    expect(mocked).toHaveBeenCalledWith(
      '/api/v1/integrations/apple/disconnect',
      { method: 'POST' }
    );
  });
});
