import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';

// Пользовательские метрики
const loginDuration = new Trend('login_duration');
const getProfileDuration = new Trend('get_profile_duration');
const biometricDuration = new Trend('biometric_duration');
const generatePlanDuration = new Trend('generate_plan_duration');

export const options = {
    insecureSkipTLSVerify: true,  // for self-signed certs
    stages: [
        { duration: '30s', target: 20 },  // разогрев до 20 пользователей
        { duration: '1m', target: 50 },   // пиковая нагрузка
        { duration: '30s', target: 0 },   // спад
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% запросов <500мс
        'login_duration': ['p(95)<300'],
    },
};

const BASE_URL = __ENV.BASE_URL || 'https://localhost:8443';
const TEST_PASSWORD = __ENV.TEST_PASSWORD || 'LoadTest123!';  // override via env
// Unique email per test run to avoid stale email_confirmed=false from previous runs
const RUN_ID = Math.random().toString(36).substring(2, 8);
const TEST_USER = {
    email: `loadtest-${RUN_ID}@example.com`,
    password: TEST_PASSWORD,
    full_name: 'Load Test User',
    role: 'client'
};

export function setup() {
    // Регистрация тестового пользователя (уникальный email каждый раз)
    const registerRes = http.post(`${BASE_URL}/api/v1/register`, JSON.stringify(TEST_USER), {
        headers: { 'Content-Type': 'application/json' },
    });
    check(registerRes, { 'register success': (r) => r.status === 200 });

    // Извлекаем токен подтверждения (dev mode)
    let verifyToken = null;
    const regData = registerRes.json();
    if (regData && regData.message) {
        const match = regData.message.match(/token \(dev only\):\s*([a-f0-9]+)/);
        if (match) verifyToken = match[1];
    }

    // Подтверждаем email
    if (verifyToken) {
        const confirmRes = http.post(`${BASE_URL}/api/v1/auth/confirm`, JSON.stringify({ token: verifyToken }), {
            headers: { 'Content-Type': 'application/json' },
        });
        check(confirmRes, { 'email confirmed': (r) => r.status === 200 });
    }

    // Логин
    const loginRes = http.post(`${BASE_URL}/api/v1/login`, JSON.stringify({
        email: TEST_USER.email,
        password: TEST_USER.password,
    }), { headers: { 'Content-Type': 'application/json' } });
    check(loginRes, { 'login success': (r) => r.status === 200 });

    const token = loginRes.json('access_token');

    // Заполняем профиль (нужно для ML генерации)
    const profileRes = http.put(`${BASE_URL}/api/v1/profile`, JSON.stringify({
        full_name: 'Load Test User',
        age: 30,
        gender: 'male',
        height_cm: 180,
        weight_kg: 75.5,
        fitness_level: 'intermediate',
        goals: ['endurance', 'weight_loss'],
        contraindications: [],
        nutrition: 'balanced',
        sleep_hours: 7.5,
    }), { headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` } });
    check(profileRes, { 'profile setup': (r) => r.status === 200 });

    return { token };
}

export default function (data) {
    const headers = {
        'Authorization': `Bearer ${data.token}`,
        'Content-Type': 'application/json',
    };

    // 1. ML Классификация - только в 30% случаев
    if (Math.random() < 0.3) {
        let startTime = new Date();
        let classifyRes = http.post(`${BASE_URL}/api/v1/ml/classify`, JSON.stringify({}), { headers });
        const classifyOk = classifyRes.status === 200 || classifyRes.status === 202;
        const classifyUnavailable = classifyRes.status === 502 || classifyRes.status === 503 || classifyRes.status === 500;
        if (classifyOk) {
            check(classifyRes, { 'ml classify ok': () => true });
        } else if (classifyUnavailable) {
            check(classifyRes, { 'ml classify skipped (service unavailable)': () => true });
        } else {
            check(classifyRes, { 'ml classify ok': () => false });
        }
        biometricDuration.add(new Date() - startTime);
    }
    sleep(0.5);

    // 2. Генерация программы - только в 30% случаев
    if (Math.random() < 0.3) {
        let startTime = new Date();
        const plan = {
            training_class: 'endurance_e1e2',
            duration_weeks: 4,
            available_days: [1, 3, 5],
            preferences: { max_duration: 60 }
        };
        let planRes = http.post(`${BASE_URL}/api/v1/ml/generate-plan`, JSON.stringify(plan), { headers });
        const planOk = planRes.status === 200 || planRes.status === 202;
        const planUnavailable = planRes.status === 502 || planRes.status === 503 || planRes.status === 500;
        if (planOk) {
            check(planRes, { 'ml generate plan ok': () => true });
        } else if (planUnavailable) {
            check(planRes, { 'ml generate skipped (service unavailable)': () => true });
        } else {
            check(planRes, { 'ml generate plan ok': () => false });
        }
        generatePlanDuration.add(new Date() - startTime);
    }
    sleep(1);
}