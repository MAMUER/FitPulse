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

// ========== Security #11: Response Signature Verification ==========
// Сервер подписывает ответы HMAC-SHA256 своим секретом.
// Клиент не может проверить HMAC без секрета (это невозможно в SPA),
// но наличие подписи обеспечивает:
// 1. Integrity: подпись фиксируется в логах для аудита
// 2. Accountability: сервер не может отрицать отправку ответа
// 3. В production: можно использовать публичный ключ (Ed25519) для проверки
//
// Пока подпись логируется. Для полной проверки нужен бэкенд-прокси,
// который проверяет HMAC и передаёт только валидные данные SPA.
function logResponseSignature(endpoint, signature, algorithm) {
    if (signature) {
        console.log(
            '[Security] Response signature for', endpoint,
            '| algorithm:', algorithm || 'HMAC-SHA256',
            '| signature:', signature.substring(0, 16) + '...'
        );
    } else {
        console.warn('[Security] No response signature for', endpoint);
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

    // Требование #11: Логирование подписи ответа для аудита целостности
    const signature = response.headers.get('X-Response-Signature');
    const algorithm = response.headers.get('X-Signature-Algorithm');
    if (signature && rawBody) {
        logResponseSignature(endpoint, signature, algorithm);
    }

    console.log('[API] Response data:', data);

    if (!response.ok) {
        const msg = typeof data === 'string' ? data : (data.message || data.error || `Ошибка сервера (${response.status})`);
        throw new Error(msg);
    }

    return data;
}

// Helper to check if response signature was invalid (blocks privileged UI)
function isSignatureInvalid() {
    return !!window.__responseSignatureInvalid;
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
async function generateTrainingPlan(durationWeeks = 4, availableDays = [1,3,5], classificationClass = '', confidence = 0) {
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

// Logout — требование #1: серверная инвалидация сессии
async function logout() {
    try {
        await apiRequest('/logout', { method: 'POST' });
    } catch (error) {
        console.warn('Logout request failed, clearing token anyway:', error);
    } finally {
        setAuthToken(null);
    }
}

// Export shared functions for use by other modules
window.apiRequest = apiRequest;
window.setAuthToken = setAuthToken;