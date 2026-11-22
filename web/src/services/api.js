import { backendRequest } from '../utils/backendRequest';
import { pathSegment } from '../utils/pathSegment';

export async function login(email, password) {
  return backendRequest('/api/v1/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export async function logout() {
  try {
    await backendRequest('/api/v1/logout', { method: 'POST' });
  } catch {
    // ignore logout errors
  }
}

export async function register(email, password, fullName, role = 'client') {
  if (!email || !password || !fullName) {
    throw new Error('Заполните все поля');
  }
  if (password.length < 8) {
    throw new Error('Пароль должен содержать минимум 8 символов');
  }
  // Возвращаем промис напрямую (как в login), без внутреннего await —
  // иначе SonarJS неверно выводит тип возврата и считает await у вызывающего
  // ошибочным (javascript:S4123).
  return backendRequest('/api/v1/register', {
    method: 'POST',
    body: JSON.stringify({
      email,
      password,
      full_name: fullName,
      role,
    }),
  });
}

export function forgotPassword(email) {
  if (!email?.includes('@')) {
    throw new Error('Введите корректный email');
  }
  return { email };
}

export async function submitResetEmail(email) {
  const data = await backendRequest('/api/v1/auth/forgot-password', {
    method: 'POST',
    body: JSON.stringify({ email }),
  });
  return data;
}

export function submitResetCode(code) {
  if (!code) {
    throw new Error('Введите код');
  }
  return { code };
}

export async function submitNewPassword(email, code, newPassword) {
  if (newPassword.length < 6) {
    throw new Error('Пароль должен содержать минимум 6 символов');
  }
  const data = await backendRequest('/api/v1/auth/reset', {
    method: 'POST',
    body: JSON.stringify({ email, code, new_password: newPassword }),
  });
  return data;
}

export function continueAsGuest() {
  return { guest: true };
}

export function socialLogin(provider) {
  if (provider === 'yandex') {
    return { redirect: '/api/v1/auth/yandex' };
  }
  throw new Error(`Интеграция ${provider} не поддерживается`);
}

// Critical session — step-up подтверждение для чувствительных операций (2FA).
// Раньше фронт звал /2fa/setup напрямую и получал 412 Precondition Failed.
export async function createCriticalSession() {
  const data = await backendRequest('/api/v1/auth/critical-session', {
    method: 'POST',
  });
  return data?.critical_session_token || '';
}

function criticalHeaders(token) {
  return token ? { 'X-Critical-Session-Token': token } : {};
}

export async function setup2FA(criticalToken) {
  const data = await backendRequest('/api/v1/auth/2fa/setup', {
    method: 'POST',
    headers: criticalHeaders(criticalToken),
  });
  return data;
}

export async function confirm2FA(
  passcode,
  tempSecret,
  backupCodes,
  criticalToken
) {
  const data = await backendRequest('/api/v1/auth/2fa/confirm', {
    method: 'POST',
    headers: criticalHeaders(criticalToken),
    body: JSON.stringify({
      passcode,
      temp_secret: tempSecret,
      backup_codes: backupCodes,
    }),
  });
  return data;
}

export async function verify2FA(tempToken, passcode, isBackupCode = false) {
  const data = await backendRequest('/api/v1/auth/2fa/verify', {
    method: 'POST',
    body: JSON.stringify({
      temp_token: tempToken,
      passcode,
      is_backup_code: isBackupCode,
    }),
  });
  return data;
}

export async function disable2FA(passcode, criticalToken) {
  const data = await backendRequest('/api/v1/auth/2fa/disable', {
    method: 'POST',
    headers: criticalHeaders(criticalToken),
    body: JSON.stringify({ passcode }),
  });
  return data;
}

export async function load2FAStatus() {
  const data = await backendRequest('/api/v1/auth/2fa/status');
  return data;
}

export async function loadProfile() {
  const data = await backendRequest('/api/v1/profile');
  return data;
}

export async function saveProfile(profileData) {
  const data = await backendRequest('/api/v1/profile', {
    method: 'PUT',
    body: JSON.stringify(profileData),
  });
  return data;
}

export async function loadBiometrics() {
  const data = await backendRequest('/api/v1/biometrics');
  return data;
}

export async function addBiometric(metricType, value, deviceType = 'manual') {
  const data = await backendRequest('/api/v1/biometrics', {
    method: 'POST',
    body: JSON.stringify({
      metric_type: metricType,
      value,
      timestamp: new Date().toISOString(),
      device_type: deviceType,
    }),
  });
  return data;
}

export async function loadBodyComposition() {
  const data = await backendRequest('/api/v1/health/body-composition');
  return data;
}

export async function saveBodyComposition(record) {
  const data = await backendRequest('/api/v1/health/body-composition', {
    method: 'POST',
    body: JSON.stringify(record),
  });
  return data;
}

export async function loadTrainingPlans() {
  const data = await backendRequest('/api/v1/training/plans');
  return data;
}

export async function generatePlan(params = {}) {
  const body = {
    duration_weeks: params.durationWeeks || 4,
    available_days: params.availableDays || [1, 3, 5],
  };
  // Класс состояния определяет бэкенд по биометрии. Если клиент передаёт
  // class явно — используем его (ручной выбор), иначе поле не отправляем.
  if (params.class) {
    body.class = params.class;
  }
  body.confidence = params.confidence || 0.8;
  const data = await backendRequest('/api/v1/training/generate', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return data;
}

export async function getPlanDetails(planId) {
  const data = await backendRequest(
    `/api/v1/training/plans/${pathSegment(planId)}`
  );
  return data;
}

export async function loadProgress() {
  const data = await backendRequest('/api/v1/training/progress');
  return data;
}

export async function completeWorkout(
  planId,
  workoutId,
  rating = 5,
  feedback = ''
) {
  const data = await backendRequest('/api/v1/training/complete', {
    method: 'POST',
    body: JSON.stringify({
      plan_id: planId,
      workout_id: workoutId,
      rating,
      feedback,
    }),
  });
  return data;
}

export async function loadAchievements() {
  const data = await backendRequest('/api/v1/achievements');
  return data;
}

export async function loadFriends() {
  const data = await backendRequest('/api/v1/friends');
  return data;
}

export async function loadDishes(slot = '', q = '') {
  const params = [];
  if (slot) params.push(`slot=${encodeURIComponent(slot)}`);
  if (q) params.push(`q=${encodeURIComponent(q)}`);
  const qs = params.length ? `?${params.join('&')}` : '';
  const data = await backendRequest(`/api/v1/dishes${qs}`);
  return data;
}

export async function addDish(dish) {
  return backendRequest('/api/v1/dishes', {
    method: 'POST',
    body: JSON.stringify({
      name: dish.name,
      meal_slot: dish.meal_slot || 'any',
      kcal: dish.kcal || 0,
      protein_g: dish.protein_g || 0,
      fat_g: dish.fat_g || 0,
      carbs_g: dish.carbs_g || 0,
      image_url: dish.image_url || '',
    }),
  });
}

export async function addFriend(
  friendName,
  friendUserId = '',
  status = 'offline'
) {
  return backendRequest('/api/v1/friends', {
    method: 'POST',
    body: JSON.stringify({
      friend_name: friendName,
      friend_user_id: friendUserId,
      status,
    }),
  });
}

export async function removeFriend(id) {
  return backendRequest(`/api/v1/friends/${id}`, { method: 'DELETE' });
}

// ---- Leaderboard & friend requests ----

export async function loadLeaderboard() {
  return backendRequest('/api/v1/leaderboard');
}

export async function syncLeaderboard({ display_name, points, xp, workouts }) {
  return backendRequest('/api/v1/leaderboard/me', {
    method: 'PUT',
    body: JSON.stringify({ display_name, points, xp, workouts }),
  });
}

export async function searchUsers(q) {
  return backendRequest(
    `/api/v1/users/search?q=${encodeURIComponent(q || '')}`
  );
}

export async function sendFriendRequest(toUserId) {
  return backendRequest('/api/v1/friends/request', {
    method: 'POST',
    body: JSON.stringify({ to_user_id: toUserId }),
  });
}

export async function loadFriendRequests() {
  return backendRequest('/api/v1/friends/requests');
}

export async function respondFriendRequest(id, action) {
  return backendRequest(`/api/v1/friends/requests/${id}/${action}`, {
    method: 'POST',
  });
}

// ---- Reports (жалобы, напр. на фото) ----

export async function reportUser(reportedUserId, reason) {
  return backendRequest('/api/v1/report', {
    method: 'POST',
    body: JSON.stringify({
      reported_user_id: reportedUserId,
      reason: reason || 'Жалоба на фото',
    }),
  });
}

export async function loadMeals() {
  const data = await backendRequest('/api/v1/nutrition/meals');
  return data;
}

export async function createMeal(name, calories, time = '') {
  const data = await backendRequest('/api/v1/nutrition/meals', {
    method: 'POST',
    body: JSON.stringify({
      name,
      calories,
      time:
        time ||
        new Date().toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit',
        }),
    }),
  });
  return data;
}

export async function removeMeal(mealId) {
  const data = await backendRequest(
    `/api/v1/nutrition/meals/${pathSegment(mealId)}`,
    {
      method: 'DELETE',
    }
  );
  return data;
}

export async function updateMeal(mealId, meal) {
  const data = await backendRequest(
    `/api/v1/nutrition/meals/${pathSegment(mealId)}`,
    {
      method: 'PUT',
      body: JSON.stringify({
        name: meal.name,
        calories: meal.calories || 0,
        time: meal.time || '',
      }),
    }
  );
  return data;
}

export async function loadCalendarEvents() {
  const data = await backendRequest('/api/v1/calendar/events');
  return data;
}

export async function createEvent(eventData) {
  const data = await backendRequest('/api/v1/calendar/events', {
    method: 'POST',
    body: JSON.stringify(eventData),
  });
  return data;
}

export async function updateEvent(eventId, title, date, type, description) {
  const data = await backendRequest(
    `/api/v1/calendar/events/${pathSegment(eventId)}`,
    {
      method: 'PUT',
      body: JSON.stringify({ title, date, type, description }),
    }
  );
  return data;
}

export async function deleteCalendarEvent(eventId) {
  const data = await backendRequest(
    `/api/v1/calendar/events/${pathSegment(eventId)}`,
    {
      method: 'DELETE',
    }
  );
  return data;
}

export async function sendAIMessage(text) {
  const data = await backendRequest('/api/v1/chat', {
    method: 'POST',
    body: JSON.stringify({ message: text }),
    // /api/v1/chat is behind AuthMiddleware, which returns 404 (not 401) for
    // a missing/expired access token. Let backendRequest refresh and retry.
    retryOnNotFound: true,
  });
  return data;
}

export const quickAI = sendAIMessage;

export async function loadVideos() {
  const data = await backendRequest('/api/v1/videos');
  return data;
}

export async function loadConditions() {
  const data = await backendRequest('/api/v1/health/conditions');
  return data;
}

export async function createCondition(conditionData) {
  const data = await backendRequest('/api/v1/health/conditions', {
    method: 'POST',
    body: JSON.stringify(conditionData),
  });
  return data;
}

export async function deleteCondition(conditionId) {
  const data = await backendRequest(
    `/api/v1/health/conditions/${pathSegment(conditionId)}`,
    {
      method: 'DELETE',
    }
  );
  return data;
}

export async function loadMenstrualCycles() {
  const data = await backendRequest('/api/v1/health/menstrual-cycles');
  return data;
}

export async function createMenstrualCycle(cycleData) {
  const data = await backendRequest('/api/v1/health/menstrual-cycles', {
    method: 'POST',
    body: JSON.stringify(cycleData),
  });
  return data;
}

export async function updateMenstrualCycle(cycleId, cycleData) {
  const data = await backendRequest(
    `/api/v1/health/menstrual-cycles/${pathSegment(cycleId)}`,
    {
      method: 'PUT',
      body: JSON.stringify(cycleData),
    }
  );
  return data;
}

export async function deleteMenstrualCycle(cycleId) {
  const data = await backendRequest(
    `/api/v1/health/menstrual-cycles/${pathSegment(cycleId)}`,
    {
      method: 'DELETE',
    }
  );
  return data;
}

export async function changePassword(oldPass, newPass, confirmPass) {
  if (!oldPass) {
    throw new Error('Введите старый пароль');
  }
  // Минимум 8 — синхронно с validator.ValidateRegisterRequest на бэкенде.
  if (newPass.length < 8) {
    throw new Error('Пароль должен содержать минимум 8 символов');
  }
  if (newPass !== confirmPass) {
    throw new Error('Пароли не совпадают');
  }
  // Раньше функция ничего не отправляла — возвращала {oldPass,newPass} как заглушку,
  // поэтому кнопка «Изменить пароль» не делала ничего. Реальный эндпоинт:
  // POST /api/v1/auth/change-password (проверяет текущий пароль и отзывает refresh-токены).
  return backendRequest('/api/v1/auth/change-password', {
    method: 'POST',
    body: JSON.stringify({ old_password: oldPass, new_password: newPass }),
  });
}

export async function changeEmail(newEmail, password) {
  if (!newEmail) {
    throw new Error('Введите новый email');
  }
  if (!password) {
    throw new Error('Введите текущий пароль');
  }
  return backendRequest('/api/v1/auth/email', {
    method: 'PUT',
    body: JSON.stringify({ new_email: newEmail, password }),
  });
}

export async function deleteAccount() {
  // Серверное удаление аккаунта (каскадит все данные пользователя).
  try {
    await backendRequest('/api/v1/account', { method: 'DELETE' });
  } catch {
    // Даже если сервер недоступен — локальные данные всё равно чистим.
  }
  // Локальная очистка (на случай офлайн/сбоя).
  try {
    localStorage.removeItem('fitpulse-merged-v1');
    localStorage.removeItem('fitpulse-merged-v9');
  } catch {
    // ignore
  }
  return true;
}

export async function saveSurvey(surveyData) {
  const data = await backendRequest('/api/v1/survey', {
    method: 'POST',
    body: JSON.stringify(surveyData),
  });
  return data;
}

export async function loadSurvey() {
  const data = await backendRequest('/api/v1/survey');
  return data;
}

export async function confirmEmail(token) {
  const data = await backendRequest('/api/v1/auth/confirm', {
    method: 'POST',
    body: JSON.stringify({ token }),
  });
  return data;
}

export async function loadIntegrationProviders() {
  const data = await backendRequest('/api/v1/integrations/providers');
  return data;
}

export async function disconnectIntegration(source) {
  const data = await backendRequest(
    `/api/v1/integrations/${pathSegment(source)}/disconnect`,
    { method: 'POST' }
  );
  return data;
}
