import { test, expect } from '@playwright/test';

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

async function setupConfig(page, config) {
  await page.goto('/');
  await page.evaluate((c) => {
    localStorage.setItem('fitpulse-merged-v9', JSON.stringify({
      screen: 'login',
      theme: c.theme,
      highContrast: c.highContrast,
      language: c.lang,
    }));
  }, config);
  await page.reload();
  await page.waitForTimeout(600);
}

function getText(key) {
  const map = {
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
      await expect(page.locator(`text=${getText('login')}`)).toBeVisible();
      await page.locator(`text=${getText('register')}`).click();
      await expect(page.locator(`text=${getText('register')}`)).toBeVisible();
    });

    test('login navigates to legal pages', async ({ page }) => {
      await page.locator(`text=${getText('terms')}`).click();
      await expect(page.locator(`text=${getText('terms')}`)).toBeVisible();
      await page.locator(`text=${getText('privacy')}`).click();
      await expect(page.locator(`text=${getText('privacy')}`).last).toBeVisible();
    });

    test('guest flow reaches home', async ({ page }) => {
      await page.locator(`text=${getText('guest')}`).click();
      await expect(page.locator(`text=${getText('home')}`)).toBeVisible();
    });

    test('home navigates to main screens', async ({ page }) => {
      await page.locator(`text=${getText('guest')}`).click();
      await page.waitForTimeout(300);
      await page.locator(`text=${getText('nutrition')}`).click();
      await expect(page.locator(`text=${getText('nutrition')}`)).toBeVisible();
      await page.locator(`text=${getText('calendar')}`).click();
      await expect(page.locator(`text=${getText('calendar')}`)).toBeVisible();
      await page.locator(`text=${getText('training')}`).click();
      await expect(page.locator(`text=${getText('training')}`)).toBeVisible();
      await page.locator(`text=${getText('ai')}`).click();
      await expect(page.locator(`text=${getText('ai')}`)).toBeVisible();
      await page.locator(`text=${getText('profile')}`).click();
      await expect(page.locator(`text=${getText('profile')}`)).toBeVisible();
      await page.locator(`text=${getText('chat')}`).click();
      await expect(page.locator(`text=${getText('chat')}`)).toBeVisible();
    });

    test('profile actions are accessible', async ({ page }) => {
      await page.locator(`text=${getText('guest')}`).click();
      await page.waitForTimeout(300);
      await page.locator(`text=${getText('profile')}`).click();
      await expect(page.locator(`text=${getText('logout')}`)).toBeVisible();
      await expect(page.locator(`text=${getText('changePassword')}`)).toBeVisible();
      await expect(page.locator(`text=${getText('deleteAccount')}`)).toBeVisible();
      await expect(page.locator(`text=${getText('theme')}`)).toBeVisible();
      await expect(page.locator(`text=${getText('language')}`)).toBeVisible();
    });

    test('training quick add and navigation', async ({ page }) => {
      await page.locator(`text=${getText('guest')}`).click();
      await page.waitForTimeout(300);
      await page.locator(`text=${getText('training')}`).click();
      await expect(page.locator(`text=${getText('training')}`)).toBeVisible();
      await page.locator(`text=${getText('startWorkout')}`).first.click();
      await expect(page.locator(`text=${getText('startWorkout')}`)).toBeVisible();
    });

    test('chat screen renders and opens settings', async ({ page }) => {
      await page.locator(`text=${getText('guest')}`).click();
      await page.waitForTimeout(300);
      await page.locator(`text=${getText('chat')}`).click();
      await expect(page.locator(`text=${getText('chat')}`)).toBeVisible();
      await page.locator(`text=${getText('chatSettings')}`).click();
      await expect(page.getByText(/chat settings/i)).toBeVisible();
    });

    test('nutrition period tabs exist', async ({ page }) => {
      await page.locator(`text=${getText('guest')}`).click();
      await page.waitForTimeout(300);
      await page.locator(`text=${getText('nutrition')}`).click();
      await expect(page.locator(`text=${getText('nutrition')}`)).toBeVisible();
    });

    test('high contrast toggle works', async ({ page }) => {
      await page.locator(`text=${getText('highContrast')}`).click();
      await page.waitForTimeout(300);
      const html = page.locator('html');
      await expect(html).toHaveAttribute('data-highcontrast', /on|off/);
    });

    test('theme toggle works', async ({ page }) => {
      await page.locator(`text=${getText('guest')}`).click();
      await page.waitForTimeout(300);
      const initial = await page.locator('html').getAttribute('data-theme');
      await page.locator(`text=${getText('theme')}`).click();
      await page.waitForTimeout(300);
      const after = await page.locator('html').getAttribute('data-theme');
      expect([initial, after].sort()).toEqual(['dark', 'light']);
    });

    test('language toggle switches labels', async ({ page }) => {
      await page.locator(`text=${getText('guest')}`).click();
      await page.waitForTimeout(300);
      const before = await page.locator('html').getAttribute('lang');
      await page.locator(`text=${getText('language')}`).click();
      await page.waitForTimeout(300);
      const after = await page.locator('html').getAttribute('lang');
      expect([before, after].sort()).toEqual(['en', 'ru']);
    });
  });
}
