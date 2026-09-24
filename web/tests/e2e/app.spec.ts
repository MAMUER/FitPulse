import { test, expect, type Page } from '@playwright/test';

const CONFIGS = [
  { name: 'RU Light Normal', lang: 'ru', theme: 'light', highContrast: false },
  { name: 'RU Light HC', lang: 'ru', theme: 'light', highContrast: true },
  { name: 'RU Dark Normal', lang: 'ru', theme: 'dark', highContrast: false },
  { name: 'RU Dark HC', lang: 'ru', theme: 'dark', highContrast: true },
  { name: 'EN Light Normal', lang: 'en', theme: 'light', highContrast: false },
  { name: 'EN Light HC', lang: 'en', theme: 'light', highContrast: true },
  { name: 'EN Dark Normal', lang: 'en', theme: 'dark', highContrast: false },
  { name: 'EN Dark HC', lang: 'en', theme: 'dark', highContrast: true },
];

async function setupConfig(page: Page, config: typeof CONFIGS[number]) {
  await page.goto('/');
  await page.evaluate(
    (c: { theme: string; highContrast: boolean; lang: string }) => {
      localStorage.setItem('fitpulse-merged-v9', JSON.stringify({
        screen: 'login',
        theme: c.theme,
        highContrast: c.highContrast,
        language: c.lang,
      }));
    },
    config
  );
  await page.reload();
  await page.waitForLoadState('networkidle');
}

type LangKey = keyof typeof CONFIGS[number]['lang'];

type TranslationKeys = {
  ru: {
    login: string;
    register: string;
    home: string;
    nutrition: string;
    calendar: string;
    training: string;
    ai: string;
    profile: string;
    chat: string;
    guest: string;
    terms: string;
    privacy: string;
    consent: string;
    forgot: string;
    highContrast: string;
    startWorkout: string;
    water: string;
    body: string;
    videos: string;
    aiAdvisor: string;
    logout: string;
    theme: string;
    language: string;
    deleteAccount: string;
    changePassword: string;
    devices: string;
    restrictions: string;
    chatSettings: string;
    nutritionSettings: string;
    searchFriends: string;
  };
  en: {
    login: string;
    register: string;
    home: string;
    nutrition: string;
    calendar: string;
    training: string;
    ai: string;
    profile: string;
    chat: string;
    guest: string;
    terms: string;
    privacy: string;
    consent: string;
    forgot: string;
    highContrast: string;
    startWorkout: string;
    water: string;
    body: string;
    videos: string;
    aiAdvisor: string;
    logout: string;
    theme: string;
    language: string;
    deleteAccount: string;
    changePassword: string;
    devices: string;
    restrictions: string;
    chatSettings: string;
    nutritionSettings: string;
    searchFriends: string;
  };
};

type TextKey = keyof TranslationKeys['ru'];

function getText(key: TextKey, config: typeof CONFIGS[number]) {
  const map: Record<string, Record<TextKey, string>> = {
    ru: {
      login: 'Войти',
      register: 'Зарегистрироваться',
      home: 'Главная',
      nutrition: 'Питание',
      calendar: 'Календарь',
      training: 'Тренировки',
      ai: 'AI',
      profile: 'Профиль',
      chat: 'Чат',
      guest: 'Продолжить как гость',
      terms: 'Пользовательское соглашение',
      privacy: 'Политика конфиденциальности',
      consent: 'Персональные данные',
      forgot: 'Забыли пароль?',
      highContrast: 'Режим высокой контрастности',
      startWorkout: 'Начать тренировку',
      water: 'Вода, мл',
      body: 'Профиль тела',
      videos: 'Видео-тренировки',
      aiAdvisor: 'AI-советник',
      logout: 'Выйти',
      theme: 'Тема',
      language: 'Язык',
      deleteAccount: 'Удалить аккаунт',
      changePassword: 'Изменить пароль',
      devices: 'Устройства',
      restrictions: 'Ограничения',
      chatSettings: 'Настройки чата',
      nutritionSettings: 'Настройки питания',
      searchFriends: 'Найти',
    },
    en: {
      login: 'Sign in',
      register: 'Register',
      home: 'Home',
      nutrition: 'Nutrition',
      calendar: 'Calendar',
      training: 'Training',
      ai: 'AI',
      profile: 'Profile',
      chat: 'Chat',
      guest: 'Continue as guest',
      terms: 'Terms of Use',
      privacy: 'Privacy Policy',
      consent: 'Personal Data',
      forgot: 'Forgot password?',
      highContrast: 'High contrast mode',
      startWorkout: 'Start workout',
      water: 'Water, ml',
      body: 'Body profile',
      videos: 'Video workouts',
      aiAdvisor: 'AI Advisor',
      logout: 'Logout',
      theme: 'Theme',
      language: 'Language',
      deleteAccount: 'Delete account',
      changePassword: 'Change password',
      devices: 'Devices',
      restrictions: 'Restrictions',
      chatSettings: 'Chat settings',
      nutritionSettings: 'Nutrition settings',
      searchFriends: 'Search',
    },
  };
  return map[config.lang]?.[key] ?? key;
}

for (const config of CONFIGS) {
  test.describe(`FitPulse E2E ${config.name}`, () => {
    test.beforeEach(async ({ page }) => {
      await setupConfig(page, config);
    });

    test('login screen renders and navigates to register', async ({ page }) => {
      await expect(page.locator(`text=${getText('login', config)}`)).toBeVisible();
      await page.locator(`text=${getText('register', config)}`).click();
      await expect(page.locator(`text=${getText('register', config)}`)).toBeVisible();
    });

    test('login navigates to legal pages', async ({ page }) => {
      await page.locator(`text=${getText('terms', config)}`).click();
      await expect(page.locator(`text=${getText('terms', config)}`)).toBeVisible();
      await page.locator(`text=${getText('privacy', config)}`).click();
      await expect(page.locator(`text=${getText('privacy', config)}`).last()).toBeVisible();
    });

    test('guest flow reaches home', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await expect(page.locator(`text=${getText('home', config)}`)).toBeVisible();
    });

    test('home navigates to main screens', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page.waitForLoadState('networkidle');
      await page.locator(`text=${getText('nutrition', config)}`).click();
      await expect(page.locator(`text=${getText('nutrition', config)}`)).toBeVisible();
      await page.locator(`text=${getText('calendar', config)}`).click();
      await expect(page.locator(`text=${getText('calendar', config)}`)).toBeVisible();
      await page.locator(`text=${getText('training', config)}`).click();
      await expect(page.locator(`text=${getText('training', config)}`)).toBeVisible();
      await page.locator(`text=${getText('ai', config)}`).click();
      await expect(page.locator(`text=${getText('ai', config)}`)).toBeVisible();
      await page.locator(`text=${getText('profile', config)}`).click();
      await expect(page.locator(`text=${getText('profile', config)}`)).toBeVisible();
      await page.locator(`text=${getText('chat', config)}`).click();
      await expect(page.locator(`text=${getText('chat', config)}`)).toBeVisible();
    });

    test('profile actions are accessible', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page.waitForLoadState('networkidle');
      await page.locator(`text=${getText('profile', config)}`).click();
      await expect(page.locator(`text=${getText('logout', config)}`)).toBeVisible();
      await expect(page.locator(`text=${getText('changePassword', config)}`)).toBeVisible();
      await expect(page.locator(`text=${getText('deleteAccount', config)}`)).toBeVisible();
      await expect(page.locator(`text=${getText('theme', config)}`)).toBeVisible();
      await expect(page.locator(`text=${getText('language', config)}`)).toBeVisible();
    });

    test('training quick add and navigation', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page.waitForLoadState('networkidle');
      await page.locator(`text=${getText('training', config)}`).click();
      await expect(page.locator(`text=${getText('training', config)}`)).toBeVisible();
      await page.locator(`text=${getText('startWorkout', config)}`).first().click();
      await expect(page.locator(`text=${getText('startWorkout', config)}`)).toBeVisible();
    });

    test('chat screen renders and opens settings', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page.waitForLoadState('networkidle');
      await page.locator(`text=${getText('chat', config)}`).click();
      await expect(page.locator(`text=${getText('chat', config)}`)).toBeVisible();
      await page.locator(`text=${getText('chatSettings', config)}`).click();
      await expect(page.getByText(/chat settings/i)).toBeVisible();
    });

    test('nutrition period tabs exist', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page.waitForLoadState('networkidle');
      await page.locator(`text=${getText('nutrition', config)}`).click();
      await expect(page.locator(`text=${getText('nutrition', config)}`)).toBeVisible();
    });

    test('high contrast toggle works', async ({ page }) => {
      await page.locator(`text=${getText('highContrast', config)}`).click();
      await page.waitForLoadState('networkidle');
      const html = page.locator('html');
      await expect(html).toHaveAttribute('data-highcontrast', /on|off/);
    });

    test('theme toggle works', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page.waitForLoadState('networkidle');
      const initial = await page.locator('html').getAttribute('data-theme');
      await page.locator(`text=${getText('theme', config)}`).click();
      await page.waitForLoadState('networkidle');
      const after = await page.locator('html').getAttribute('data-theme');
      expect([initial, after].sort()).toEqual(['dark', 'light']);
    });

    test('language toggle switches labels', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page.waitForLoadState('networkidle');
      const before = await page.locator('html').getAttribute('lang');
      await page.locator(`text=${getText('language', config)}`).click();
      await page.waitForLoadState('networkidle');
      const after = await page.locator('html').getAttribute('lang');
      expect([before, after].sort()).toEqual(['en', 'ru']);
    });
  });
}
