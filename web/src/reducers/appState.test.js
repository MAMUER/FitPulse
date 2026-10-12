import { beforeEach, describe, expect, it, vi } from 'vitest';
import { loadState } from '../services/storage';
import { defaultState, getInitialState } from './appState';

vi.mock('../services/storage', () => ({ loadState: vi.fn() }));

const mockedLoadState = vi.mocked(loadState);

beforeEach(() => {
  mockedLoadState.mockReset();
  mockedLoadState.mockReturnValue(null);
});

describe('getInitialState defaults', () => {
  it('returns defaults when nothing is saved', () => {
    const state = getInitialState();
    expect(state.screen).toBe('login');
    expect(state.theme).toBe('dark');
    expect(state.highContrast).toBe(false);
    expect(state.language).toBe('ru');
    expect(state.calendarMonthIndex).toBe(new Date().getMonth());
    expect(state.calendarYear).toBe(new Date().getFullYear());
    expect(state.profileImage).toBe('');
    expect(state.profile.name).toBe(defaultState.profile.name);
    expect(state.profile.email).toBe(defaultState.profile.email);
    expect(state.profile.friends).toBe(defaultState.profile.friends);
    expect(state.profile.devices).toEqual(defaultState.profile.devices);
    expect(state.calendarEvents).toEqual({});
    expect(state.customExercises).toEqual([]);
    expect(state.customMeals).toEqual([]);
    expect(state.waterIntake).toBe(0);
    expect(state.waterDate).toBe('');
    expect(state.points).toBe(1270);
    expect(state.level).toBe(5);
    expect(state.streak).toBe(12);
    expect(state.maxStreak).toBe(18);
    expect(state.friends).toEqual(defaultState.friends);
    expect(state.pinnedPlaces).toEqual(defaultState.pinnedPlaces);
    expect(state.hackIndex).toBe(0);
    expect(state.nutritionPeriod).toBe('day');
    expect(state.trainingPeriod).toBe('day');
    expect(state.survey).toEqual(defaultState.survey);
    expect(state.surveyCompleted).toBe(false);
    expect(state.surveyDeferred).toBe(false);
    expect(state.registered).toBe(false);
    expect(state.guest).toBe(false);
    expect(state.region).toBe('RU');
    expect(state.chatConsent).toBe(false);
    expect(state.chatSettings).toEqual(defaultState.chatSettings);
    expect(state.chatMessages).toEqual(defaultState.chatMessages);
    expect(state.storySeen).toEqual({});
    expect(state.twoFactorSetup).toBeNull();
    expect(state.twoFactorTempToken).toBeNull();
    expect(state.profileLoaded).toBe(false);
    expect(state.selectedPlanId).toBeNull();
  });

  it('uses the current month when the index is undefined', () => {
    mockedLoadState.mockReturnValue({ calendarYear: 2027 });
    const state = getInitialState();
    expect(state.calendarMonthIndex).toBe(new Date().getMonth());
    expect(state.calendarYear).toBe(2027);
  });
});

describe('getInitialState saved values', () => {
  it('merges every persisted field', () => {
    mockedLoadState.mockReturnValue({
      screen: 'home',
      theme: 'light',
      highContrast: true,
      language: 'en',
      calendarMonthIndex: 5,
      calendarYear: 2027,
      profileImage: 'data:image/png;base64,AAA',
      profileName: 'Тест',
      gender: 'Женский',
      email: 'test@example.com',
      phone: '+7 000 000 00 00',
      twoFactor: true,
      backupCodesRemaining: 3,
      devices: ['Apple Watch'],
      bio: 'Био',
      status: 'Активен',
      friendsCount: 7,
      calendarEvents: { '2027-06-10': [{ title: 'Тренировка' }] },
      customExercises: [{ name: 'Планка' }],
      customMeals: [{ name: 'Боул' }],
      waterIntake: 4,
      waterDate: '2027-06-10',
      points: 100,
      level: 2,
      streak: 5,
      maxStreak: 9,
      friends: [{ name: 'Анна' }],
      pinnedPlaces: [{ name: 'Дом' }],
      hackIndex: 2,
      nutritionPeriod: 'week',
      trainingPeriod: 'month',
      completedWorkouts: [{ id: 1 }],
      dailyCompleted: 3,
      xp: 50,
      trainingMinutes: 30,
      caloriesBurned: 100,
      biometrics: [{ value: 70 }],
      bodyComposition: [{ fat: 18.4 }],
      trainingPlans: [{ id: 'p1' }],
      selectedPlanId: 'p1',
      mealsList: [{ name: 'Овсянка' }],
      calendarEventsList: [{ id: 'e1' }],
      achievementsBackend: [{ id: 'a1' }],
      videos: [{ id: 'v1' }],
      conditions: [{ id: 'c1' }],
      menstrualCycles: [{ id: 'm1' }],
      aiClassification: { class: 'endurance_basic' },
      aiPlan: { weeks: 4 },
      aiDiet: { kcal: 2000 },
      loading: { profile: true },
      survey: { step: 2 },
      surveyCompleted: true,
      surveyDeferred: true,
      registered: true,
      guest: true,
      region: 'US',
      chatConsent: true,
      chatSettings: { allowGuest: true },
      chatMessages: [{ from: 'Я', text: 'Привет' }],
      storySeen: { 0: true },
      twoFactorSetup: { secret: 's' },
      twoFactorTempToken: 'temp',
    });

    const state = getInitialState();

    expect(state.screen).toBe('home');
    expect(state.theme).toBe('light');
    expect(state.highContrast).toBe(true);
    expect(state.language).toBe('en');
    expect(state.calendarMonthIndex).toBe(5);
    expect(state.calendarYear).toBe(2027);
    expect(state.profileImage).toBe('data:image/png;base64,AAA');
    expect(state.profile).toEqual({
      ...defaultState.profile,
      name: 'Тест',
      gender: 'Женский',
      email: 'test@example.com',
      phone: '+7 000 000 00 00',
      twoFactor: true,
      backupCodesRemaining: 3,
      devices: ['Apple Watch'],
      bio: 'Био',
      status: 'Активен',
      friends: 7,
    });
    expect(state.calendarEvents).toEqual({
      '2027-06-10': [{ title: 'Тренировка' }],
    });
    expect(state.customExercises).toEqual([{ name: 'Планка' }]);
    expect(state.customMeals).toEqual([{ name: 'Боул' }]);
    expect(state.waterIntake).toBe(4);
    expect(state.waterDate).toBe('2027-06-10');
    expect(state.points).toBe(100);
    expect(state.level).toBe(2);
    expect(state.streak).toBe(5);
    expect(state.maxStreak).toBe(9);
    expect(state.friends).toEqual([{ name: 'Анна' }]);
    expect(state.pinnedPlaces).toEqual([{ name: 'Дом' }]);
    expect(state.hackIndex).toBe(2);
    expect(state.nutritionPeriod).toBe('week');
    expect(state.trainingPeriod).toBe('month');
    expect(state.completedWorkouts).toEqual([{ id: 1 }]);
    expect(state.dailyCompleted).toBe(3);
    expect(state.xp).toBe(50);
    expect(state.trainingMinutes).toBe(30);
    expect(state.caloriesBurned).toBe(100);
    expect(state.biometrics).toEqual([{ value: 70 }]);
    expect(state.bodyComposition).toEqual([{ fat: 18.4 }]);
    expect(state.trainingPlans).toEqual([{ id: 'p1' }]);
    expect(state.selectedPlanId).toBe('p1');
    expect(state.mealsList).toEqual([{ name: 'Овсянка' }]);
    expect(state.calendarEventsList).toEqual([{ id: 'e1' }]);
    expect(state.achievementsBackend).toEqual([{ id: 'a1' }]);
    expect(state.videos).toEqual([{ id: 'v1' }]);
    expect(state.conditions).toEqual([{ id: 'c1' }]);
    expect(state.menstrualCycles).toEqual([{ id: 'm1' }]);
    expect(state.aiClassification).toEqual({ class: 'endurance_basic' });
    expect(state.aiPlan).toEqual({ weeks: 4 });
    expect(state.aiDiet).toEqual({ kcal: 2000 });
    expect(state.loading).toEqual({ profile: true });
    expect(state.survey).toEqual({ step: 2 });
    expect(state.surveyCompleted).toBe(true);
    expect(state.surveyDeferred).toBe(true);
    expect(state.registered).toBe(true);
    expect(state.guest).toBe(true);
    expect(state.region).toBe('US');
    expect(state.chatConsent).toBe(true);
    expect(state.chatSettings).toEqual({ allowGuest: true });
    expect(state.chatMessages).toEqual([{ from: 'Я', text: 'Привет' }]);
    expect(state.storySeen).toEqual({ 0: true });
    expect(state.twoFactorSetup).toEqual({ secret: 's' });
    expect(state.twoFactorTempToken).toBe('temp');
    expect(state.profileLoaded).toBe(false);
  });

  it('keeps a zero friends count and default profile fields', () => {
    mockedLoadState.mockReturnValue({ friendsCount: 0 });
    const state = getInitialState();
    expect(state.profile.friends).toBe(0);
    expect(state.profile.name).toBe(defaultState.profile.name);
  });
});

describe('getInitialState failures', () => {
  it('returns defaults when loading state throws', () => {
    mockedLoadState.mockImplementation(() => {
      throw new Error('storage unavailable');
    });
    expect(getInitialState()).toEqual(defaultState);
  });

  it('returns defaults for undefined saved fields', () => {
    mockedLoadState.mockReturnValue(undefined);
    const state = getInitialState();
    expect(state.screen).toBe('login');
    expect(state.theme).toBe('dark');
    expect(state.profile).toEqual(defaultState.profile);
  });
});
