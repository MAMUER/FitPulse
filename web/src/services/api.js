import { backendRequest } from '../utils/backendRequest';

export async function login(email, password) {
  return backendRequest('/api/v1/auth/login', {
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

export function register(email, password, confirm) {
  if (!email || !password || !confirm) {
    throw new Error('Заполните все поля');
  }
  if (password !== confirm) {
    throw new Error('Пароли не совпадают');
  }
  if (password.length < 6) {
    throw new Error('Пароль должен содержать минимум 6 символов');
  }
  return { email, password, confirm };
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
  if (provider === 'google') {
    return { redirect: '/api/v1/auth/google' };
  }
  throw new Error(`Интеграция ${provider} не поддерживается`);
}

export async function setup2FA() {
  const data = await backendRequest('/api/v1/auth/2fa/setup', {
    method: 'POST',
  });
  return data;
}

export async function confirm2FA(passcode, tempSecret, backupCodes) {
  const data = await backendRequest('/api/v1/auth/2fa/confirm', {
    method: 'POST',
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

export async function disable2FA(passcode) {
  const data = await backendRequest('/api/v1/auth/2fa/disable', {
    method: 'POST',
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
  const data = await backendRequest('/api/v1/training/generate', {
    method: 'POST',
    body: JSON.stringify({
      duration_weeks: params.durationWeeks || 4,
      available_days: params.availableDays || [1, 3, 5],
      class: params.class || 'endurance_basic',
      confidence: params.confidence || 0.8,
    }),
  });
  return data;
}

export async function getPlanDetails(planId) {
  const data = await backendRequest(`/api/v1/training/plans/${planId}`);
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
  const data = await backendRequest(`/api/v1/nutrition/meals/${mealId}`, {
    method: 'DELETE',
  });
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
  const data = await backendRequest(`/api/v1/calendar/events/${eventId}`, {
    method: 'PUT',
    body: JSON.stringify({ title, date, type, description }),
  });
  return data;
}

export async function deleteCalendarEvent(eventId) {
  const data = await backendRequest(`/api/v1/calendar/events/${eventId}`, {
    method: 'DELETE',
  });
  return data;
}

export async function sendAIMessage(text) {
  const data = await backendRequest('/api/v1/chat', {
    method: 'POST',
    body: JSON.stringify({ message: text }),
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
    `/api/v1/health/conditions/${conditionId}`,
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
    `/api/v1/health/menstrual-cycles/${cycleId}`,
    {
      method: 'PUT',
      body: JSON.stringify(cycleData),
    }
  );
  return data;
}

export async function deleteMenstrualCycle(cycleId) {
  const data = await backendRequest(
    `/api/v1/health/menstrual-cycles/${cycleId}`,
    {
      method: 'DELETE',
    }
  );
  return data;
}

export function changePassword(oldPass, newPass, confirmPass) {
  if (!oldPass) {
    throw new Error('Введите старый пароль');
  }
  if (newPass.length < 6) {
    throw new Error('Пароль должен содержать минимум 6 символов');
  }
  if (newPass !== confirmPass) {
    throw new Error('Пароли не совпадают');
  }
  return { oldPass, newPass };
}

export async function deleteAccount() {
  localStorage.removeItem('fitpulse-merged-v9');
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
    `/api/v1/integrations/${source}/disconnect`,
    { method: 'POST' }
  );
  return data;
}
