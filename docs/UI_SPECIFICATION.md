# FitPulse — UI Specification v11

> **Scope:** Mobile-first React SPA aligned with `FitPulse_FINAL_v11_RELEASE.html`.
> **Target device:** Mobile browser (`390–430 px` viewport, touch-first).
> **Runtime:** React 19 + Vite 8 + React Router v7 + Context API.

---

## 1. Architecture

### 1.1. Application type

Single Page Application (SPA). All screens are React components under `web/src/screens/`. Navigation is handled by React Router in `web/src/App.jsx`. Global state lives in `web/src/contexts/AppContext.jsx`.

### 1.2. Screens and routes

| Route | Screen component | Notes |
| --- | --- | --- |
| `/login` | `Login` | Default auth screen |
| `/register` | `Register` | Registration + consent |
| `/reset` | `Reset` | Password reset flow |
| `/` | `Home` | Default main screen |
| `/home` | `Home` | Dashboard, stories, metrics |
| `/nutrition` | `Nutrition` | Meal plan, macros, alternatives |
| `/calendar` | `Calendar` | Month view, quick events |
| `/training` | `Training` | Plans, start workout, places |
| `/videos` | `Videos` | Video workout cards |
| `/ai` | `AI` | AI chat + quick actions |
| `/body` | `Body` | Body composition, metrics |
| `/profile` | `Profile` | Profile, settings, account |
| `/chat` | `Chat` | Chat contacts and messages |
| `/legal` | `Legal` | Privacy, terms, consent |

Unauthenticated users see auth routes only. Authenticated users see main routes with a bottom tab bar.

### 1.3. Bottom tab bar

Tabs: Home, Nutrition, Calendar, Training, AI, Chat. Active tab is highlighted with the blue accent.

### 1.4. Technologies

- **React 19** with Vite 8.
- **React Router v7** for client-side routing.
- **Plain CSS** with CSS variables in `web/src/index.css`.
- **Font Awesome 6** via CDN, with local unicode fallbacks.
- **Google Fonts:** Barlow Condensed + Manrope.

---

## 2. Design tokens

### 2.1. Dark theme (default)

```css
:root {
  --bg: #050b08;
  --card: #0d1b14;
  --card-2: #15291f;
  --line: rgba(255,255,255,.1);
  --text: #f3fff7;
  --muted: #8da99a;
  --green: #6fae20;
  --green-light: #a5ed38;
  --blue: #19d8da;
  --orange: #ff9947;
  --red: #ff6375;
  --purple: #a78bfa;
  --yellow: #f5d45d;
  --gold: #f5c842;
  --radius: 20px;
  --font: 'Manrope', sans-serif;
  --font-display: 'Barlow Condensed', sans-serif;
}
```

### 2.2. Light theme

```css
[data-theme="light"] {
  --bg: #f1f8f4;
  --card: #fff;
  --card-2: #e2f0e8;
  --line: rgba(20,70,45,.13);
  --text: #13251b;
  --muted: #607669;
}
```

### 2.3. High contrast

```css
[data-highcontrast="on"] {
  --bg: #fff !important;
  --card: #fff !important;
  --card-2: #f2f2f2 !important;
  --line: #111 !important;
  --text: #000 !important;
  --muted: #333 !important;
}
```

Dark high-contrast overrides are also defined via `[data-theme="dark"][data-highcontrast="on"]`.

### 2.4. Layout

- App shell: `width: min(100%, 460px); height: min(920px, 100vh); border-radius: 32px;`
- Page padding: `24px 17px 108px`.
- Safe-area insets respected via `env(safe-area-inset-*)`.

---

## 3. Screen specifications

### 3.1. Login

**File:** `web/src/screens/Login.jsx`

- Email + password fields.
- Primary CTA: `doLogin()`.
- Secondary: `continueAsGuest()`.
- Social: `socialLogin('google')`.
- Links: register, forgot password, terms, privacy, personal data agreement.
- Controls: theme toggle, language toggle, high-contrast toggle.

### 3.2. Register

**File:** `web/src/screens/Register.jsx`

- Email, password, confirm password.
- Consent checkbox.
- CTA: `doRegister()`.
- Social login, legal links, language/high-contrast toggles.

### 3.3. Reset

**File:** `web/src/screens/Reset.jsx`

- Step 1: email input + submit.
- Step 2: code input + submit.
- Step 3: new password + confirm + submit.

### 3.4. Home

**File:** `web/src/screens/Home.jsx`

- Notification bell (`showNotification`).
- Theme toggle.
- Profile shortcut.
- Survey resume (`resumeSurvey`).
- Start workout (`go('training')`).
- Stories carousel (`scrollStories`, `showAllStories`).
- Metrics: pulse, sleep, oxygen, stress (`openMetric`).
- Mini calendar with `changeCalendar(-1|1)`.
- Water tracker (`drinkWater`).
- Places map with `editPlace`, `selectPlace`, `addPlace`.
- Quick links: body analytics, full calendar.

### 3.5. Calendar

**File:** `web/src/screens/Calendar.jsx`

- Month grid with event dots.
- Quick add events: gym, pool, recovery, work, study, note (`quickAddEvent`).
- Month navigation (`changeCalendar`).
- Event list for selected day.
- Calendar page overlay (`openCalendarPage`).

### 3.6. Nutrition

**File:** `web/src/screens/Nutrition.jsx`

- Period tabs: day / week / month (`setNutritionPeriod`).
- Goal display + nutrition settings (`nutritionSettings`).
- Water tracker (`drinkWater`).
- Meal slots: breakfast, lunch, snack, dinner.
- Meal modal with alternatives (`openMeal`, `selectMeal`).
- Custom meal save (`saveCustomMeal`).
- Ask AI shortcut (`askAI`).

### 3.7. Body

**File:** `web/src/screens/Body.jsx`

- Body composition cards: weight, fat, muscle, protein.
- Metric modals (`openMetric`).
- Add weight (`addWeight`).
- Additional metrics: pulse, sleep, oxygen, HRV, stress, BMI.

### 3.8. Training

**File:** `web/src/screens/Training.jsx`

- Period tabs: day / week / month (`setTrainingPeriod`).
- Workout cards with expandable exercise lists (`toggleWorkout`).
- Start workout (`startWorkout`).
- Places editor (`editPlace`, `selectPlace`, `addPlace`).

### 3.9. Videos

**File:** `web/src/screens/Videos.jsx`

- Video cards with play overlay.
- Equipment tags, duration, level.
- Navigation via bottom tab bar.

### 3.10. AI

**File:** `web/src/screens/AI.jsx`

- Quick action buttons: menu, dinner, calendar, body composition.
- Chat-like message list (`sendAI`, `quickAI`).
- AI can add calendar events from natural language.

### 3.11. Profile

**File:** `web/src/screens/Profile.jsx`

- Theme toggle, language toggle, high-contrast toggle.
- Edit profile (`editProfile`).
- Nutrition settings (`nutritionSettings`).
- Two-factor toggle (`toggleTwoFactor`).
- Change password (`changePassword`).
- Connected devices (`showDevices`).
- Restrictions/allergies (`showRestrictions`).
- Chat settings (`chatSettings`).
- Delete account (`deleteAccount`).
- Legal links: privacy, terms, personal data.

### 3.12. Chat

**File:** `web/src/screens/Chat.jsx`

- Chat contacts list.
- Chat messages for selected contact.
- Chat settings (`chatSettings`).
- Legal links.
- Navigation to login/register for guests.

### 3.13. Legal

**File:** `web/src/screens/Legal.jsx`

- Privacy policy, terms of use, personal data agreement.
- Back button returns to previous screen.

### 3.14. Cookie Consent

**File:** `web/src/components/CookieConsent.jsx`

- Fixed banner at bottom of screen.
- Text: "Мы используем файлы cookie для улучшения работы сервиса. Продолжая использовать FitPulse, вы соглашаетесь с нашей Политикой конфиденциальности."
- Actions: «Принять» (accept), «Отклонить» (decline).
- State: `localStorage.getItem('cookie-consent')` → `'accepted'` | `'declined'`.
- Accessibility: `role="dialog"`, `aria-label="Cookie consent"`.
- Does not block scrolling; `position: fixed` at bottom.

### 3.15. Medical Disclaimer

**File:** Inline in `web/src/screens/Body.jsx`, `web/src/screens/Register.jsx`

- Text: "Это не медицинский совет. При заболеваниях или травмах consult врача."
- Styling: yellow-ish background (`#fff3cd33`), border (`#ffc10755`), rounded corners.
- Accessibility: `role="note"`, `aria-live="polite"`.
- High contrast mode: applies high-contrast colors.
- Does not auto-dismiss; user must explicitly proceed.

### 3.16. Special Category Consent (Register)

**File:** `web/src/screens/Register.jsx`

- Checkbox: "Я даю согласие на обработку специальных категорий персональных данных (сведения о здоровье, менструальном цикле) в соответствии с Политикой конфиденциальности."
- Required for registration (cannot proceed without checking).
- Accessibility: `aria-describedby` linked to consent text.

### 3.17. Real-time Metrics & ARIA Live

**File:** `web/src/screens/Home.jsx`, `web/src/screens/Body.jsx`

- Metric updates (pulse, SpO2, etc.) announced via `aria-live="polite"` regions.
- Toasts: `role="status"`, `aria-live="polite"`, `aria-atomic="true"`.
- Loading states: `role="progressbar"`, `aria-valuenow`, `aria-valuemin`, `aria-valuemax`.
- Error messages: `role="alert"`, `aria-live="assertive"`.

---

## 4. State management

**File:** `web/src/contexts/AppContext.jsx`

Centralized state via React Context. State shape includes:

- `screen`, `theme`, `highContrast`, `language`
- `profile`, `events`, `meals`, `alternatives`
- `metricInfo`, `weightHistory`, `trainingData`, `nutritionData`
- `messages`, `chatContacts`, `chatMessages`
- `stories`, `pinnedPlaces`, `trainingVideos`
- `achievements`, `dailyQuests`, `lifeHacks`
- `survey`, `chatSettings`, `progressGoals`

State is persisted to `localStorage` under `fitpulse-merged-v9`.

---

## 5. API integration

**File:** `web/src/utils/backendRequest.js`

```js
export async function backendRequest(path, options = {})
```

Used for backend communication. Falls back to local demo data when `BACKEND_CONFIG.baseUrl` is empty.

---

## 6. Internationalization

**File:** `web/src/utils/i18n.js`

```js
export function t(ru, _en)
```

Returns the appropriate string based on `localStorage` language preference. All strings are passed as `(ru, en)` pairs from components and context.

---

## 7. Accessibility

See `docs/A11Y.md`.

---

## 8. Testing

### 8.1. Unit tests

- Vitest + React Testing Library.
- Located alongside components: `*.test.jsx`, `*.test.js`.
- Run: `npm run test`

### 8.2. E2E tests

- Playwright (`@playwright/test`).
- Config: `web/playwright.config.ts`.
- Test matrix: 8 configurations (RU/EN × Light/Dark × Normal/High Contrast).
- Run: `npm run test:e2e`

---

## 9. Actual files

| Path | Purpose |
| --- | --- |
| `web/src/main.jsx` | React entry point |
| `web/src/App.jsx` | Router + tab bar |
| `web/src/index.css` | Global styles, design tokens |
| `web/src/contexts/AppContext.jsx` | Global state and handlers |
| `web/src/utils/i18n.js` | Localization helper |
| `web/src/utils/helpers.js` | Date, calendar, text utilities |
| `web/src/utils/backendRequest.js` | Backend API helper |
| `web/src/screens/*.jsx` | Screen components |
| `web/tests/e2e/app.spec.ts` | Playwright E2E matrix |
| `web/playwright.config.ts` | Playwright config |

---

## 10. Migration notes

Previous architecture used separate component folders (`components/Dashboard/`, `components/Training/`, etc.) with a tab-based view switcher. v11 consolidates screen-specific markup into `src/screens/` while preserving shared utilities and contexts.
