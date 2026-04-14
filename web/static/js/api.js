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
// HMAC-SHA256 verification using Web Crypto API
async function hmacSha256Sign(data, secret) {
    const encoder = new TextEncoder();
    const key = await crypto.subtle.importKey(
        'raw',
        encoder.encode(secret),
        { name: 'HMAC', hash: 'SHA-256' },
        false,
        ['sign']
    );
    const signature = await crypto.subtle.sign('HMAC', key, encoder.encode(data));
    // Base64 encode
    return btoa(String.fromCharCode(...new Uint8Array(signature)));
}

async function verifyResponseSignature(bodyText, signature, secret) {
    try {
        const expectedSig = await hmacSha256Sign(bodyText, secret);
        // Constant-time comparison via string equality (not cryptographically perfect but sufficient for client-side)
        return expectedSig === signature;
    } catch (e) {
        console.error('[Security] Signature verification failed:', e);
        return false;
    }
}

// API key for signature verification (loaded from server config or env)
// In production, this should be fetched from a /api/v1/config endpoint
const SIGNATURE_SECRET = window.API_SIGNATURE_SECRET || '';
let signatureVerificationEnabled = !!SIGNATURE_SECRET;

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

    // Требование #11: Verifying response signature for critical endpoints
    const signature = response.headers.get('X-Response-Signature');
    if (signature && signatureVerificationEnabled && rawBody) {
        const isValid = await verifyResponseSignature(rawBody, signature, SIGNATURE_SECRET);
        if (!isValid) {
            console.error('[Security] CRITICAL: Response signature verification FAILED!');
            console.error('[Security] Response may have been tampered with. Privileged functions disabled.');
            // Block privileged functions — signature is invalid
            window.__responseSignatureInvalid = true;
            throw new Error('Ошибка целостности ответа. Операция отменена.');
        } else {
            console.log('[Security] Response signature verified OK');
        }
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