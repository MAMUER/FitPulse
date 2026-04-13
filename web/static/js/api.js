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

    let data;
    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
        data = await response.json();
    } else {
        data = await response.text();
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