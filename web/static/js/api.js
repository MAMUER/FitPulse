const API_BASE = '/api/v1';

let authToken = localStorage.getItem('authToken');
console.log('[API] init, token:', authToken ? 'present' : 'null');

function setAuthToken(token) {
    authToken = token;
    if (token) {
        localStorage.setItem('authToken', token);
        console.log('[API] Token saved');
    } else {
        localStorage.removeItem('authToken');
        console.log('[API] Token removed');
    }
}

async function apiRequest(endpoint, options = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers
    };

    if (authToken) {
        headers['Authorization'] = `Bearer ${authToken}`;
    }

    console.log('[API] Request:', endpoint, options.method || 'GET');

    const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        headers
    });

    console.log('[API] Response status:', response.status);

    if (response.status === 401) {
        setAuthToken(null);
        window.location.reload();
        throw new Error('Сессия истекла. Войдите заново');
    }

    if (response.status === 429) {
        const retryAfter = response.headers.get('Retry-After');
        const msg = retryAfter
            ? `Слишком много запросов. Повторите через ${retryAfter} сек.`
            : 'Слишком много запросов. Попробуйте через минуту.';
        throw new Error(msg);
    }

    let data;
    let rawBody = '';
    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
        rawBody = await response.text();
        try {
            data = JSON.parse(rawBody);
        } catch (e) {
            data = rawBody;
        }
    } else {
        rawBody = await response.text();
        data = rawBody;
    }

    console.log('[API] Response data:', data);

    if (!response.ok) {
        const msg = typeof data === 'string' ? data : (data.message || data.error || `Ошибка сервера (${response.status})`);
        throw new Error(msg);
    }

    return data;
}

// Auth
async function register(email, password, fullName, role = 'client') {
    return apiRequest('/register', {
        method: 'POST',
        body: JSON.stringify({ email, password, full_name: fullName, role })
    });
}

async function login(email, password) {
    const data = await apiRequest('/login', {
        method: 'POST',
        body: JSON.stringify({ email, password })
    });
    if (data.access_token) {
        setAuthToken(data.access_token);
    }
    return data;
}

async function getProfile() {
    return apiRequest('/profile');
}

async function updateProfile(profile) {
    return apiRequest('/profile', {
        method: 'PUT',
        body: JSON.stringify(profile)
    });
}

async function changePassword(currentPassword, newPassword) {
    return apiRequest('/auth/change-password', {
        method: 'POST',
        body: JSON.stringify({ current_password: currentPassword, new_password: newPassword })
    });
}

async function changeEmail(newEmail, password) {
    return apiRequest('/auth/change-email', {
        method: 'POST',
        body: JSON.stringify({ new_email: newEmail, password })
    });
}

async function get2FAStatus() {
    return apiRequest('/auth/2fa/status');
}

async function setup2FA() {
    return apiRequest('/auth/2fa/setup', { method: 'POST' });
}

async function confirm2FA(passcode, tempSecret, backupCodes) {
    return apiRequest('/auth/2fa/confirm', {
        method: 'POST',
        body: JSON.stringify({ passcode, temp_secret: tempSecret, backup_codes: backupCodes })
    });
}

async function verify2FA(tempToken, passcode, isBackupCode = false) {
    return apiRequest('/auth/2fa/verify', {
        method: 'POST',
        body: JSON.stringify({ temp_token: tempToken, passcode, is_backup_code: isBackupCode })
    });
}

async function disable2FA(passcode) {
    return apiRequest('/auth/2fa/disable', {
        method: 'POST',
        body: JSON.stringify({ passcode })
    });
}

async function deleteProfile(password) {
    return apiRequest('/profile', {
        method: 'DELETE',
        body: JSON.stringify({ password })
    });
}

// Biometrics
async function addBiometricRecord(metricType, value, timestamp, deviceType) {
    return apiRequest('/biometrics', {
        method: 'POST',
        body: JSON.stringify({ metric_type: metricType, value, timestamp, device_type: deviceType })
    });
}

async function getBiometricRecords(metricType, from, to, limit = 100) {
    let url = `/biometrics?metric_type=${metricType}&limit=${limit}`;
    if (from) url += `&from=${from}`;
    if (to) url += `&to=${to}`;
    return apiRequest(url);
}

// Training
async function generateTrainingPlan(durationWeeks = 4, availableDays = [1, 3, 5], classificationClass = '', confidence = 0) {
    return apiRequest('/training/generate', {
        method: 'POST',
        body: JSON.stringify({
            duration_weeks: durationWeeks,
            available_days: availableDays,
            class: classificationClass,
            confidence: confidence
        })
    });
}

async function getTrainingPlans(page = 1, pageSize = 10) {
    return apiRequest(`/training/plans?page=${page}&page_size=${pageSize}`);
}

async function getPlan(planId) {
    return apiRequest(`/training/plans/${planId}`);
}

async function completeWorkout(planId, workoutId, rating, feedback) {
    return apiRequest('/training/complete', {
        method: 'POST',
        body: JSON.stringify({ plan_id: planId, workout_id: workoutId, rating, feedback })
    });
}

async function getProgress() {
    return apiRequest('/training/progress');
}

// Achievements
async function getAchievements() {
    return apiRequest('/achievements');
}

async function logout() {
    try {
        await apiRequest('/logout', { method: 'POST' });
    } catch (error) {
        console.warn('Logout request failed, clearing token anyway:', error);
    } finally {
        setAuthToken(null);
    }
}

// Health features
async function listHealthConditions(conditionType = '') {
    let url = '/health/conditions';
    if (conditionType) url += `?condition_type=${encodeURIComponent(conditionType)}`;
    return apiRequest(url);
}

async function upsertHealthCondition(data) {
    return apiRequest('/health/conditions', {
        method: 'POST',
        body: JSON.stringify(data)
    });
}

async function deleteHealthCondition(conditionId) {
    return apiRequest(`/health/conditions/${conditionId}`, { method: 'DELETE' });
}

async function listBodyComposition(from, to, limit = 100) {
    let url = `/health/body-composition?limit=${limit}`;
    if (from) url += `&from=${encodeURIComponent(from)}`;
    if (to) url += `&to=${encodeURIComponent(to)}`;
    return apiRequest(url);
}

async function createBodyComposition(data) {
    return apiRequest('/health/body-composition', {
        method: 'POST',
        body: JSON.stringify(data)
    });
}

async function listMenstrualCycles() {
    return apiRequest('/health/menstrual-cycles');
}

async function createMenstrualCycle(data) {
    return apiRequest('/health/menstrual-cycles', {
        method: 'POST',
        body: JSON.stringify(data)
    });
}

async function updateMenstrualCycle(cycleId, data) {
    return apiRequest(`/health/menstrual-cycles/${cycleId}`, {
        method: 'PUT',
        body: JSON.stringify(data)
    });
}

async function deleteMenstrualCycle(cycleId) {
    return apiRequest(`/health/menstrual-cycles/${cycleId}`, { method: 'DELETE' });
}

async function syncFlo(accessToken, refreshToken) {
    return apiRequest('/health/sync/flo', {
        method: 'POST',
        body: JSON.stringify({ access_token: accessToken, refresh_token: refreshToken })
    });
}

async function syncOKOK(accessToken, refreshToken) {
    return apiRequest('/health/sync/okok', {
        method: 'POST',
        body: JSON.stringify({ access_token: accessToken, refresh_token: refreshToken })
    });
}

// Export shared functions for use by other modules
window.apiRequest = apiRequest;
window.setAuthToken = setAuthToken;
window.login = login;
window.get2FAStatus = get2FAStatus;
window.setup2FA = setup2FA;
window.confirm2FA = confirm2FA;
window.verify2FA = verify2FA;
window.disable2FA = disable2FA;
window.logout = logout;
window.listHealthConditions = listHealthConditions;
window.upsertHealthCondition = upsertHealthCondition;
window.deleteHealthCondition = deleteHealthCondition;
window.listBodyComposition = listBodyComposition;
window.createBodyComposition = createBodyComposition;
window.listMenstrualCycles = listMenstrualCycles;
window.createMenstrualCycle = createMenstrualCycle;
window.updateMenstrualCycle = updateMenstrualCycle;
window.deleteMenstrualCycle = deleteMenstrualCycle;
window.syncFlo = syncFlo;
window.syncOKOK = syncOKOK;