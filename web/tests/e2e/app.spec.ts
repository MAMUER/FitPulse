import { test, expect } from '@playwright/test';

test.describe('FitPulse E2E smoke tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('renders auth screen when not authenticated', async ({ page }) => {
    await expect(page.locator('text=Регистрация')).toBeVisible();
    await expect(page.locator('text=Вход')).toBeVisible();
  });

  test('navigates to privacy policy', async ({ page }) => {
    await page.click('text=Политика конфиденциальности');
    await expect(page.locator('text=Политика конфиденциальности')).toBeVisible();
  });

  test('navigates to terms of service', async ({ page }) => {
    await page.click('text=Условия использования');
    await expect(page.locator('text=Условия использования')).toBeVisible();
  });

  test('successful login shows dashboard', async ({ page }) => {
    await page.route('**/api/v1/login', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          access_token: 'test-token',
          refresh_token: 'test-refresh',
          role: 'client',
        }),
      });
    });

    await page.route('**/api/v1/profile', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'user-1',
          email: 'test@example.com',
          full_name: 'Test User',
          role: 'client',
        }),
      });
    });

    await page.fill('#login-email', 'test@example.com');
    await page.fill('#login-password', 'password123');
    await page.click('button[type="submit"]');

    await expect(page.locator('text=Дашборд')).toBeVisible();
  });

  test('failed login shows error message', async ({ page }) => {
    await page.route('**/api/v1/login', (route) => {
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ detail: 'Invalid credentials' }),
      });
    });

    await page.fill('#login-email', 'wrong@example.com');
    await page.fill('#login-password', 'wrongpassword');
    await page.click('button[type="submit"]');

    await expect(page.locator('.auth-error')).toContainText('Invalid credentials');
  });

  test('successful registration switches to verify mode', async ({ page }) => {
    await page.route('**/api/v1/register', (route) => {
      route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          message: 'Registration successful. Please verify your email.',
        }),
      });
    });

    await page.click('text=Создать');
    await page.fill('#register-name', 'Test User');
    await page.fill('#register-email', 'new@example.com');
    await page.fill('#register-password', 'SecurePass1');
    await page.click('button[type="submit"]');

    await expect(page.locator('text=Подтвердите email')).toBeVisible();
  });

  test('switches between login and register modes', async ({ page }) => {
    await page.click('text=Создать');
    await expect(page.locator('text=Создать аккаунт')).toBeVisible();

    await page.click('text=Войти');
    await expect(page.locator('text=Войти')).toBeVisible();
  });
});
