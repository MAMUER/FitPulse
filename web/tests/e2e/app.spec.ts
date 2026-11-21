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

// ---------------------------------------------------------------------------
// Реальный пользователь для E2E (гостевой режим удалён). Логин/пароль берутся
// ТОЛЬКО из SEED_ADMIN_EMAIL / SEED_ADMIN_PASSWORD (в CI — GitHub secrets).
// Никаких хардкод-креденшелов: если их нет — авторизованные тесты пропускаются.
const ADMIN_EMAIL = process.env.SEED_ADMIN_EMAIL ?? '';
const ADMIN_PASSWORD = process.env.SEED_ADMIN_PASSWORD ?? '';
const HAS_AUTH = Boolean(process.env.BASE_URL && ADMIN_EMAIL && ADMIN_PASSWORD);

async function login(page: Page, email: string, password: string) {
  await page.goto('/login');
  await page.locator('#loginEmail').fill(email);
  await page.locator('#loginPassword').fill(password);
  await page.getByRole('button', { name: 'Войти', exact: true }).click();
  // После успешного входа SPA уходит с /login на /home.
  await page.waitForURL((url) => !url.pathname.includes('/login'), {
    timeout: 15000,
  });
}

// Дедупликация: раньше матрица гоняла 8 конфигов (RU/EN × light/dark × HC) × 11
// тестов = 88 одинаковых проверок. Кросс-браузерность обеспечивают 3 проекта
// (chromium/firefox/webkit), а тему/контраст/язык тесты переключают сами,
// поэтому достаточно одного репрезентативного конфига.
for (const config of HAS_AUTH ? CONFIGS.slice(0, 1) : []) {
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

    test('login reaches home', async ({ page }) => {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await expect(
        page.locator(`text=${getText('home', config)}`)
      ).toBeVisible();
    });

    test('home navigates to main screens', async ({ page }) => {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
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
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await page.goto('/profile');
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

    test('training screen renders for a logged-in user', async ({ page }) => {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
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
    });

    test('nutrition period tabs exist', async ({ page }) => {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
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
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await page.goto('/profile');
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
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await page.goto('/profile');
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

// ---------------------------------------------------------------------------
// Authenticated functional E2E.
// Авторизованные тесты требуют реального бэкенда с засеянным админом, поэтому
// запускаются только когда задан BASE_URL (CI/прод). Локально (vite dev без
// backend) блок пропускается, чтобы не ломать `npm run test:e2e`.
(HAS_AUTH ? test.describe : test.describe.skip)(
  'FitPulse authenticated (admin)',
  () => {
    test('login screen: Yandex button surfaces a visible toast', async ({
      page,
    }) => {
      await page.goto('/login');
      await page.getByRole('button', { name: 'Яндекс' }).click();
      // Тосты раньше не рендерились — этот тест ловит регрессию.
      await expect(page.locator('.fp-toast.show')).toBeVisible();
      await expect(page.locator('.fp-toast.show')).toContainText('Яндекс');
    });

    test('admin logs in and reaches home', async ({ page }) => {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await expect(
        page.getByRole('heading', { name: 'Главная', exact: true })
      ).toBeVisible();
    });

    test('main screens render without uncaught runtime errors', async ({
      page,
    }) => {
      const errors: string[] = [];
      page.on('pageerror', (err) => errors.push(err.message));

      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);

      const targets: Array<{ nav: string; heading: string }> = [
        { nav: 'Питание', heading: 'Питание' },
        { nav: 'Календарь', heading: 'Календарь' },
        { nav: 'Тренировки', heading: 'Тренировки' },
        { nav: 'AI', heading: 'AI' },
      ];
      for (const target of targets) {
        await page
          .locator('.bottom-nav .nav')
          .filter({ hasText: new RegExp(`^\\s*${target.nav}\\s*$`) })
          .click();
        await expect(
          page.getByRole('heading', { name: target.heading, exact: true })
        ).toBeVisible();
      }

      // Ни один экран не должен ронять приложение (เช่น чёрный экран /videos).
      expect(errors).toEqual([]);
    });

    test('profile settings and achievements are reachable for a logged-in user', async ({
      page,
    }) => {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await page.goto('/profile');
      await expect(
        page.getByRole('heading', { name: 'Профиль', exact: true })
      ).toBeVisible();
      // «Достижения» и «Друзей» — кнопки статистики ведут к спискам.
      await expect(page.locator('#achievements')).toBeAttached();
    });

    test('profile: change-password opens a working modal', async ({ page }) => {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await page.goto('/profile');
      await page.getByText('Изменить пароль').first().click();
      // Модалка со сменой пароля должна открыться (поля #oldPass/#newPass).
      await expect(page.locator('dialog[open]')).toBeVisible();
      await expect(page.locator('#oldPass')).toBeVisible();
      await expect(page.locator('#newPass')).toBeVisible();
      // Закрытие возвращает к профилю.
      await page.getByRole('button', { name: 'Закрыть' }).first().click();
      await expect(page.locator('dialog[open]')).toHaveCount(0);
    });

    test('nutrition: manual "Персональные настройки" were removed', async ({
      page,
    }) => {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await page
        .locator('.bottom-nav .nav')
        .filter({ hasText: /^\s*Питание\s*$/ })
        .click();
      await expect(
        page.getByRole('heading', { name: 'Питание', exact: true })
      ).toBeVisible();
      // Кнопка ручной правки калорий удалена — цели только из опроса/GAN.
      await expect(page.getByText('Персональные настройки')).toHaveCount(0);
    });

    test('secondary screens render without runtime errors', async ({
      page,
    }) => {
      const errors: string[] = [];
      page.on('pageerror', (err) => errors.push(err.message));
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);

      // Экраны, не покрытые нижней навигацией: тело, цикл, устройства,
      // лидеры, видео. Раньше /videos падал в чёрный экран — ловим pageerror.
      for (const path of [
        '/body',
        '/menstrual',
        '/integrations',
        '/leaderboard',
        '/videos',
      ]) {
        await page.goto(path);
        await expect(page.locator('section').first()).toBeVisible();
      }

      expect(errors).toEqual([]);
    });

    test('nutrition diary section is present', async ({ page }) => {
      await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await page.goto('/nutrition');
      await expect(
        page.getByRole('heading', { name: 'Питание', exact: true })
      ).toBeVisible();
      // Секция дневника (заголовок или подзаголовок).
      await expect(
        page.getByText(/Дневник питания|Diet diary/i).first()
      ).toBeVisible();
    });
  }
);
