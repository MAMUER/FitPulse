const API_BASE = '/api/v1';

function setAuthToken(token) {
  if (token) {
    localStorage.setItem('authToken', token);
  } else {
    localStorage.removeItem('authToken');
  }
}

function getAuthToken() {
  return localStorage.getItem('authToken');
}

async function apiRequest(endpoint, options = {}) {
  const token = getAuthToken();
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options.headers,
  };

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    setAuthToken(null);
    window.location.reload();
    throw new Error('Сессия истекла. Войдите заново');
  }

  if (response.status === 429) {
    const retryAfter = response.headers.get('Retry-After');
    throw new Error(
      retryAfter
        ? `Слишком много запросов. Повторите через ${retryAfter} сек.`
        : 'Слишком много запросов. Попробуйте через минуту.'
    );
  }

  const contentType = response.headers.get('content-type') || '';
  let data;
  if (contentType.includes('application/json')) {
    data = await response.json();
  } else {
    data = await response.text();
  }

  if (!response.ok) {
    const msg =
      typeof data === 'string'
        ? data
        : data?.message || data?.error || `Ошибка сервера (${response.status})`;
    throw new Error(msg);
  }

  return data;
}

export async function login(email, password) {
  const data = await apiRequest('/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
  if (data.access_token) {
    setAuthToken(data.access_token);
  }
  return data;
}

export async function register(email, password, fullName, role = 'client') {
  return apiRequest('/register', {
    method: 'POST',
    body: JSON.stringify({ email, password, full_name: fullName, role }),
  });
}

export async function registerWithInvite(code, name, email, password) {
  return apiRequest('/register/invite', {
    method: 'POST',
    body: JSON.stringify({
      invite_code: code,
      full_name: name,
      email,
      password,
    }),
  });
}

export async function validateInvite(code) {
  return apiRequest('/invite/validate', {
    method: 'POST',
    body: JSON.stringify({ code }),
  });
}

export async function getProfile() {
  return apiRequest('/profile');
}

export async function updateProfile(profile) {
  return apiRequest('/profile', {
    method: 'PUT',
    body: JSON.stringify(profile),
  });
}

export async function changePassword(currentPassword, newPassword) {
  return apiRequest('/auth/change-password', {
    method: 'POST',
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });
}

export async function changeEmail(newEmail, password) {
  return apiRequest('/auth/change-email', {
    method: 'POST',
    body: JSON.stringify({ new_email: newEmail, password }),
  });
}

export async function get2FAStatus() {
  return apiRequest('/auth/2fa/status');
}

export async function setup2FA() {
  return apiRequest('/auth/2fa/setup', { method: 'POST' });
}

export async function confirm2FA(passcode, tempSecret, backupCodes) {
  return apiRequest('/auth/2fa/confirm', {
    method: 'POST',
    body: JSON.stringify({
      passcode,
      temp_secret: tempSecret,
      backup_codes: backupCodes,
    }),
  });
}

export async function verify2FA(tempToken, passcode, isBackupCode = false) {
  return apiRequest('/auth/2fa/verify', {
    method: 'POST',
    body: JSON.stringify({
      temp_token: tempToken,
      passcode,
      is_backup_code: isBackupCode,
    }),
  });
}

export async function disable2FA(passcode) {
  return apiRequest('/auth/2fa/disable', {
    method: 'POST',
    body: JSON.stringify({ passcode }),
  });
}

export async function deleteProfile(password) {
  return apiRequest('/profile', {
    method: 'DELETE',
    body: JSON.stringify({ password }),
  });
}

export async function logout() {
  try {
    await apiRequest('/logout', { method: 'POST' });
  } catch (error) {
    console.warn('Logout request failed, clearing token anyway:', error);
  } finally {
    setAuthToken(null);
  }
}

export async function addBiometricRecord(
  metricType,
  value,
  timestamp,
  deviceType
) {
  return apiRequest('/biometrics', {
    method: 'POST',
    body: JSON.stringify({
      metric_type: metricType,
      value,
      timestamp,
      device_type: deviceType,
    }),
  });
}

export async function getBiometricRecords(metricType, from, to, limit = 100) {
  let url = `/biometrics?metric_type=${metricType}&limit=${limit}`;
  if (from) url += `&from=${from}`;
  if (to) url += `&to=${to}`;
  return apiRequest(url);
}

export async function generateTrainingPlan(
  durationWeeks = 4,
  availableDays = [1, 3, 5],
  classificationClass = '',
  confidence = 0
) {
  return apiRequest('/training/generate', {
    method: 'POST',
    body: JSON.stringify({
      duration_weeks: durationWeeks,
      available_days: availableDays,
      class: classificationClass,
      confidence: confidence,
    }),
  });
}

export async function getTrainingPlans(page = 1, pageSize = 10) {
  return apiRequest(`/training/plans?page=${page}&page_size=${pageSize}`);
}

export async function getPlan(planId) {
  return apiRequest(`/training/plans/${planId}`);
}

export async function completeWorkout(planId, workoutId, rating, feedback) {
  return apiRequest('/training/complete', {
    method: 'POST',
    body: JSON.stringify({
      plan_id: planId,
      workout_id: workoutId,
      rating,
      feedback,
    }),
  });
}

export async function getProgress() {
  return apiRequest('/training/progress');
}

export async function getAchievements() {
  return apiRequest('/achievements');
}

export async function listHealthConditions(conditionType = '') {
  let url = '/health/conditions';
  if (conditionType)
    url += `?condition_type=${encodeURIComponent(conditionType)}`;
  return apiRequest(url);
}

export async function upsertHealthCondition(data) {
  return apiRequest('/health/conditions', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteHealthCondition(conditionId) {
  return apiRequest(`/health/conditions/${conditionId}`, { method: 'DELETE' });
}

export async function listBodyComposition(from, to, limit = 100) {
  let url = `/health/body-composition?limit=${limit}`;
  if (from) url += `&from=${encodeURIComponent(from)}`;
  if (to) url += `&to=${encodeURIComponent(to)}`;
  return apiRequest(url);
}

export async function createBodyComposition(data) {
  return apiRequest('/health/body-composition', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function listMenstrualCycles() {
  return apiRequest('/health/menstrual-cycles');
}

export async function createMenstrualCycle(data) {
  return apiRequest('/health/menstrual-cycles', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateMenstrualCycle(cycleId, data) {
  return apiRequest(`/health/menstrual-cycles/${cycleId}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deleteMenstrualCycle(cycleId) {
  return apiRequest(`/health/menstrual-cycles/${cycleId}`, {
    method: 'DELETE',
  });
}

export async function getProviders() {
  return apiRequest('/integrations/providers');
}

export async function disconnectIntegration(source) {
  return apiRequest(`/integrations/${source}/disconnect`, {
    method: 'POST',
  });
}

export async function confirmEmail(token) {
  return apiRequest('/auth/confirm-email', {
    method: 'POST',
    body: JSON.stringify({ token }),
  });
}

export async function classifyState(biometrics) {
  return apiRequest('/ml/classify', {
    method: 'POST',
    body: JSON.stringify(biometrics),
  });
}

export async function generateMLPlan(
  trainingClass,
  userProfile,
  goal,
  constraints
) {
  return apiRequest('/ml/generate-plan', {
    method: 'POST',
    body: JSON.stringify({
      training_class: trainingClass,
      user_profile: userProfile,
      goal,
      constraints,
    }),
  });
}

export async function listInvites(page = 1, pageSize = 10, used = '') {
  const params = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
  });
  if (used !== '') params.set('used', String(used));
  return apiRequest(`/invites?${params.toString()}`);
}

export async function createInvite(role, specialty, maxUses) {
  return apiRequest('/invites', {
    method: 'POST',
    body: JSON.stringify({ role, specialty, max_uses: maxUses }),
  });
}

export async function revokeInvite(code) {
  return apiRequest(`/invites/${code}/revoke`, { method: 'POST' });
}

export async function listUsers(page = 1, pageSize = 10) {
  return apiRequest(`/admin/users?page=${page}&page_size=${pageSize}`);
}

export { apiRequest, getAuthToken, setAuthToken };
