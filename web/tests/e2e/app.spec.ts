import { expect, type Page, test } from '@playwright/test';

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

async function setupConfig(page: Page, config: (typeof CONFIGS)[number]) {
  // addInitScript выполняется до скриптов страницы, поэтому значение гарантированно
  // попадает в localStorage раньше, чем приложение успеет записать своё состояние.
  // Прежняя схема goto('/') -> evaluate(setItem) -> reload() была гонкой: эффект
  // persist приложения мог перезаписать ключ после evaluate, и тест оставался на
  // лендинге без кнопки «Продолжить как гость» (отсюда флаки в 4 воркерах).
  await page.addInitScript(
    (c: { theme: string; highContrast: boolean; lang: string }) => {
      localStorage.setItem(
        'fitpulse-merged-v1',
        JSON.stringify({
          screen: 'login',
          theme: c.theme,
          highContrast: c.highContrast,
          language: c.lang,
        })
      );
    },
    config
  );
  await page.goto('/');
}

type TranslationKeys = {
  ru: {
    login: string;
    register: string;
    home: string;
    nutrition: string;
    calendar: string;
    training: string;
    timer: string;
    ai: string;
    profile: string;
    guest: string;
    terms: string;
    privacy: string;
    privacyLink: string;
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
    timer: string;
    ai: string;
    profile: string;
    guest: string;
    terms: string;
    privacy: string;
    privacyLink: string;
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
    nutritionSettings: string;
    searchFriends: string;
  };
};

type TextKey = keyof TranslationKeys['ru'];

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// navButton находит кнопку в нижней навигации по её тексту. Иконки
// FontAwesome попадают в accessible name, из-за чего getByRole с exact:true
// не находит кнопку; textContent при этом содержит только подпись.
function navButton(page: Page, label: string) {
  return page
    .locator('.bottom-nav .nav')
    .filter({ hasText: new RegExp(`^\\s*${escapeRegExp(label)}\\s*$`) });
}

function getText(key: TextKey, config: (typeof CONFIGS)[number]) {
  const map: Record<string, Record<TextKey, string>> = {
    ru: {
      login: 'Войти',
      register: 'Зарегистрироваться',
      home: 'Главная',
      nutrition: 'Питание',
      calendar: 'Календарь',
      training: 'Тренировки',
      timer: 'Таймер',
      ai: 'AI',
      profile: 'Профиль',
      guest: 'Продолжить как гость',
      terms: 'Пользовательское соглашение',
      privacy: 'Политика конфиденциальности',
      // На экране логина ссылка стоит в винительном падеже.
      privacyLink: 'Политику конфиденциальности',
      consent: 'Персональные данные',
      forgot: 'Забыли пароль?',
      highContrast: 'Режим высокой контрастности',
      startWorkout: 'Начать',
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
      timer: 'Timer',
      ai: 'AI',
      profile: 'Profile',
      guest: 'Continue as guest',
      terms: 'Terms of Use',
      privacy: 'Privacy Policy',
      privacyLink: 'Privacy Policy',
      consent: 'Personal Data',
      forgot: 'Forgot password?',
      highContrast: 'High contrast mode',
      startWorkout: 'Start',
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
      await expect(
        page.locator(`text=${getText('login', config)}`)
      ).toBeVisible();
      await page.locator(`text=${getText('register', config)}`).click();
      await expect(
        page.locator(`text=${getText('register', config)}`)
      ).toBeVisible();
    });

    test('login navigates to legal pages', async ({ page }) => {
      const back = config.lang === 'ru' ? 'Назад' : 'Back';
      const title = page.locator('.panel-title');

      await page
        .locator(`text=${getText('terms', config)}`)
        .first()
        .click();
      await expect(title).toHaveText(getText('terms', config));

      await page.getByRole('button', { name: back }).click();
      await page
        .locator(`text=${getText('privacyLink', config)}`)
        .first()
        .click();
      await expect(title).toHaveText(getText('privacy', config));
    });

    test('guest flow reaches home', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await expect(
        page.locator(`text=${getText('home', config)}`)
      ).toBeVisible();
    });

    test('home navigates to main screens', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      for (const key of ['nutrition', 'calendar', 'training', 'ai'] as const) {
        const label = getText(key, config);
        // Кнопки навигации содержат иконку FontAwesome, поэтому точное
        // сопоставление по accessible name ("AI" ловил "Training") не работает.
        // Сопоставляем текст кнопки в .bottom-nav регуляркой.
        await navButton(page, label).click();
        await expect(page.locator('.bottom-nav .nav.active')).toContainText(
          label
        );
      }
    });

    test('profile actions are accessible', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page
        .getByRole('button', { name: getText('profile', config) })
        .click();
      const settings = page.locator('.settings .setting');
      // Кнопка смены пароля — это .secondary.full, а не .setting, поэтому ищем
      // её по тексту напрямую (в блоке настроек и в опасной зоне).
      await expect(
        page.locator(`text=${getText('changePassword', config)}`).first()
      ).toBeVisible();
      await expect(
        settings.filter({ hasText: getText('theme', config) }).first()
      ).toBeVisible();
      await expect(
        settings.filter({ hasText: getText('language', config) }).first()
      ).toBeVisible();
      // The delete-account button lives in the danger block, not in .settings.
      await expect(
        page.locator(`text=${getText('deleteAccount', config)}`).first()
      ).toBeVisible();
    });

    test('training screen renders for guests (start hidden)', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page
        .locator('.bottom-nav .nav')
        .filter({ hasText: getText('training', config) })
        .click();
      await expect(
        page.getByRole('heading', {
          name: getText('training', config),
          exact: true,
        })
      ).toBeVisible();
      // Гостю доступен просмотр, но не запуск тренировок (guest lockdown):
      // кнопка «Начать» скрыта, пока пользователь не авторизован.
      await expect(
        page.locator(`text=${getText('startWorkout', config)}`).first()
      ).toHaveCount(0);
    });

    test('nutrition period tabs exist', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page
        .locator('.bottom-nav .nav')
        .filter({ hasText: getText('nutrition', config) })
        .click();
      await expect(
        page.getByRole('heading', {
          name: getText('nutrition', config),
          exact: true,
        })
      ).toBeVisible();
    });

    test('high contrast toggle works', async ({ page }) => {
      await page.locator(`text=${getText('highContrast', config)}`).click();
      const html = page.locator('html');
      await expect(html).toHaveAttribute('data-highcontrast', /on|off/);
    });

    test('theme toggle works', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page
        .getByRole('button', { name: getText('profile', config) })
        .click();
      const html = page.locator('html');
      const initial = await html.getAttribute('data-theme');
      await page
        .locator('.settings .setting')
        .filter({ hasText: getText('theme', config) })
        .click();
      await expect(html).toHaveAttribute(
        'data-theme',
        initial === 'dark' ? 'light' : 'dark'
      );
      const after = await html.getAttribute('data-theme');
      expect([initial, after].sort()).toEqual(['dark', 'light']);
    });

    test('language toggle switches labels', async ({ page }) => {
      await page.locator(`text=${getText('guest', config)}`).click();
      await page
        .getByRole('button', { name: getText('profile', config) })
        .click();
      const html = page.locator('html');
      const before = await html.getAttribute('lang');
      await page
        .locator('.settings .setting')
        .filter({ hasText: getText('language', config) })
        .click();
      await expect(html).toHaveAttribute('lang', before === 'ru' ? 'en' : 'ru');
      const after = await html.getAttribute('lang');
      expect([before, after].sort()).toEqual(['en', 'ru']);
    });
  });
}
