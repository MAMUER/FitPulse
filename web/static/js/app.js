document.addEventListener('DOMContentLoaded', () => {
    console.log('[APP] Loaded');
    // ===== State =====
    const state = {
        currentView: 'dashboard',
        heartChart: null,
        isAdmin: false,
        twoFATempToken: null,
    };
    // Mapping from exercise identifiers to Russian display names
    const EXERCISE_NAME_MAP = {
        "jumping_jacks": "Прыжки на месте",
        "arm_circles": "Вращение руками",
        "high_knees": "Подъем коленей",
        "pushups": "Отжимания",
        "squats": "Приседания",
        "plank": "Планка",
        "lunges": "Выпады",
        "burpees": "Бёрпи",
        "mountain_climbers": "Альпинист",
        "stretching": "Растяжка",
        "deep_breathing": "Глубокое дыхание",
        "treadmill_walk": "Ходьба на беговой дорожке",
        "dynamic_stretch": "Динамическая растяжка",
        "bench_press": "Жим лёжа",
        "deadlift": "Становая тяга",
        "leg_press": "Жим ногами",
        "lat_pulldown": "Тяга верхнего блока",
        "shoulder_press": "Жим плечами",
        "cable_rows": "Тяга блока",
        "foam_rolling": "Фоам-роллинг",
        "static_stretching": "Статическая растяжка",
        "easy_swim": "Лёгкое плавание",
        "freestyle_intervals": "Интервалы вольным стилем",
        "breaststroke": "Брасс",
        "backstroke": "На спине",
        "kickboard_drills": "Работа с доской",
        "pool_stretching": "Растяжка в бассейне",
        "brisk_walk": "Быстрая ходьба",
        "leg_swings": "Махи ногами",
        "running": "Бег",
        "cycling": "Велосипед",
        "hill_sprints": "Спринты в гору",
        "bodyweight_circuit": "Круговая тренировка",
        "walk_recovery": "Ходьба",
        "active_recovery": "Активное восстановление",
        "light_warmup": "Лёгкая разминка",
        "breathing_exercises": "Дыхательные упражнения"
    };
    // ===== DOM Elements =====
    const authScreen = document.getElementById('authScreen');
    const mainScreen = document.getElementById('mainScreen');
    const loginForm = document.getElementById('loginForm');
    const registerForm = document.getElementById('registerForm');
    const verifyForm = document.getElementById('verifyForm');
    const pageTitle = document.getElementById('pageTitle');
    const viewTitles = {
        dashboard: 'Обзор',
        profile: 'Профиль',
        training: 'Тренировки',
        devices: 'Устройства',
        achievements: 'Достижения',
        diet: 'Диета',
        health: 'Здоровье',
        ml: 'AI Анализ',
        admin: 'Админка',
    };
    // ===== Init =====
    function init() {
        console.log('[APP] Init, authToken:', authToken ? 'present' : 'null');
        const urlParams = new URLSearchParams(window.location.search);
        const confirmToken = urlParams.get('token');
        if (confirmToken) {
            console.log('[APP] Found confirm token in URL, auto-confirming...');
            showAuthScreen();
            loginForm.classList.add('hidden');
            registerForm.classList.add('hidden');
            if (verifyForm) {
                verifyForm.classList.remove('hidden');
                document.getElementById('verifyToken').value = confirmToken;
                autoConfirmEmail(confirmToken);
            }
            return;
        }
        if (authToken) {
            showMainApp();
        } else {
            showAuthScreen();
        }
        bindEvents();
    }
    function showAuthScreen() {
        authScreen.classList.add('active');
        mainScreen.classList.remove('active');
        mainScreen.classList.add('hidden');
        loginForm.classList.remove('hidden');
        registerForm.classList.add('hidden');
        const login2FAForm = document.getElementById('login2FAForm');
        if (login2FAForm) login2FAForm.classList.add('hidden');
        if (verifyForm) verifyForm.classList.add('hidden');
        clearErrors();
        console.log('[APP] Auth screen shown');
    }
    function showLogin2FA(tempToken) {
        state.twoFATempToken = tempToken;
        authScreen.classList.add('active');
        mainScreen.classList.remove('active');
        mainScreen.classList.add('hidden');
        loginForm.classList.add('hidden');
        registerForm.classList.add('hidden');
        if (verifyForm) verifyForm.classList.add('hidden');
        const login2FAForm = document.getElementById('login2FAForm');
        if (login2FAForm) {
            login2FAForm.classList.remove('hidden');
            const error = document.getElementById('login2FAError');
            if (error) error.classList.add('hidden');
            document.getElementById('totpLoginCode')?.focus();
        }
    }
    async function connectFitbit() {
        const token = localStorage.getItem('authToken');
        if (!token) {
            showToast('Необходима авторизация', 'error');
            return;
        }
        // Проксируем через Gateway
        window.location.href = '/api/v1/devices/fitbit/auth';
    }
    async function loadConnectedProviders() {
        try {
            // FIX: Убрано дублирование /api/v1
            const response = await apiRequest('/devices/providers', {
                method: 'GET'
            });
            const container = document.getElementById('connectedDevicesList');
            if (!container) return;
            if (!response.providers || response.providers.length === 0) {
                container.innerHTML = '<p style="color: var(--text-secondary);">Нет подключённых устройств</p>';
                return;
            }
            const providerNames = {
                fitbit: 'Fitbit',
                withings: 'Withings',
                garmin: 'Garmin'
            };
            const providerIcons = {
                fitbit: '⌚',
                withings: '⚖️',
                garmin: '🏃'
            };
            container.innerHTML = response.providers.map(p => `
<div style="background: var(--bg-card); padding: 16px; border-radius: var(--radius-md); margin-bottom: 12px;">
<div style="display: flex; justify-content: space-between; align-items: center;">
<div>
<h4 style="margin: 0 0 8px 0;">${providerIcons[p.provider] || '📱'} ${providerNames[p.provider] || p.provider}</h4>
<p style="font-size: 13px; color: var(--text-secondary); margin: 0;">
ID: ${p.provider_user_id}
${p.last_sync_at ? `<br>Последняя синхронизация: ${new Date(p.last_sync_at).toLocaleString('ru-RU')}` : ''}
</p>
</div>
<div style="display: flex; gap: 8px;">
<span style="padding: 4px 8px; background: ${p.is_active ? 'var(--green)' : 'var(--text-tertiary)'}; color: white; border-radius: 12px; font-size: 12px;">
${p.is_active ? 'Активен' : 'Отключён'}
</span>
${p.is_active ? `<button data-action="disconnect-provider" data-provider="${p.provider}" class="btn-secondary" style="padding: 8px 12px; font-size: 13px;">
Отключить
</button>` : ''}
</div>
</div>
</div>
`).join('');
        } catch (err) {
            console.error('Failed to load providers:', err);
        }
    }
    // Универсальная функция отключения провайдера
    async function disconnectProvider(provider) {
        if (!confirm(`Отключить ${provider}?`)) return;
        try {
            await apiRequest(`/devices/${provider}/disconnect`, {
                method: 'POST'
            });
            showToast(`${provider} отключён`, 'success');
            loadConnectedProviders();
        } catch (err) {
            showToast(`Ошибка отключения: ${err.message}`, 'error');
        }
    }
    async function disconnectFitbit() {
        if (!confirm('Отключить Fitbit?')) return;
        try {
            await apiRequest('/devices/fitbit/disconnect', {
                method: 'POST'
            });
            showToast('Fitbit отключён', 'success');
            loadConnectedProviders();
        } catch (err) {
            showToast('Ошибка отключения: ' + err.message, 'error');
        }
    }
    window.connectFitbit = connectFitbit;
    window.disconnectFitbit = disconnectFitbit;
    async function connectWithings() {
        const token = localStorage.getItem('authToken');
        if (!token) {
            showToast('Необходима авторизация', 'error');
            return;
        }
        window.location.href = '/api/v1/devices/withings/auth';
    }
    async function disconnectWithings() {
        if (!confirm('Отключить Withings?')) return;
        try {
            await apiRequest('/devices/withings/disconnect', { method: 'POST' });
            showToast('Withings отключён', 'success');
            loadConnectedProviders();
        } catch (err) {
            showToast('Ошибка: ' + err.message, 'error');
        }
    }
    window.connectWithings = connectWithings;
    window.disconnectWithings = disconnectWithings;
    function showVerification(email, message, userId) {
        console.log('[APP] Show verification for:', email);
        loginForm.classList.add('hidden');
        registerForm.classList.add('hidden');
        if (verifyForm) verifyForm.classList.remove('hidden');
        document.getElementById('verifyEmail').textContent = email;
        const tokenMatch = message.match(/token \(dev only\):\s*([a-f0-9]+)/i);
        const devSection = document.getElementById('devTokenSection');
        if (tokenMatch && devSection) {
            devSection.classList.remove('hidden');
            document.getElementById('devToken').textContent = tokenMatch[1];
        } else if (devSection) {
            devSection.classList.add('hidden');
        }
        const confirmErr = document.getElementById('confirmError');
        const confirmOk = document.getElementById('confirmSuccess');
        if (confirmErr) { confirmErr.textContent = ''; confirmErr.classList.add('hidden'); }
        if (confirmOk) confirmOk.classList.add('hidden');
        const tokenInput = document.getElementById('verifyToken');
        if (tokenInput) tokenInput.value = '';
    }
    function copyToken() {
        const token = document.getElementById('devToken')?.textContent;
        if (token) {
            navigator.clipboard.writeText(token).then(() => {
                showToast('Токен скопирован!', 'success');
            });
        }
    }
    window.copyToken = copyToken;
    async function confirmEmail(token) {
        console.log('[AUTH] Confirming email with token:', token);
        const response = await fetch('/api/v1/auth/confirm', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token })
        });
        const data = await response.json();
        if (!response.ok) {
            throw new Error(data.message || data.error || 'Ошибка подтверждения');
        }
        return data;
    }
    async function autoConfirmEmail(token) {
        const confirmErr = document.getElementById('confirmError');
        const confirmOk = document.getElementById('confirmSuccess');
        const btn = document.getElementById('confirmBtn');
        if (btn) { btn.disabled = true; btn.textContent = 'Подтверждение...'; }
        if (confirmErr) confirmErr.classList.add('hidden');
        if (confirmOk) confirmOk.classList.add('hidden');
        try {
            await confirmEmail(token);
            if (confirmOk) confirmOk.classList.remove('hidden');
            showToast('Email подтверждён! Переход ко входу...', 'success');
            setTimeout(() => {
                loginForm.classList.remove('hidden');
                if (verifyForm) verifyForm.classList.add('hidden');
            }, 2000);
        } catch (err) {
            console.error('[AUTH] Auto-confirm failed:', err);
            if (confirmErr) {
                confirmErr.textContent = 'Ошибка: ' + err.message;
                confirmErr.classList.remove('hidden');
            }
        } finally {
            if (btn) { btn.disabled = false; btn.textContent = 'Подтвердить email'; }
        }
    }
    // ===== Admin Functions =====
    // После логина показываем вкладку админа для админов
    async function checkAdminRole() {
        try {
            // FIX: Убрано дублирование /api/v1
            const profile = await apiRequest('/profile', {
                method: 'GET'
            });
            if (profile && (profile.role === 'admin' || (profile.profile && profile.profile.role === 'admin'))) {
                state.isAdmin = true;
                const adminTab = document.getElementById('adminTab');
                if (adminTab) adminTab.style.display = 'flex';
                console.log('[APP] Admin role detected, admin tab shown');
            }
        } catch (e) {
            console.error('Failed to check admin role:', e);
        }
    }
    // Функции админки — экспортируем в window для onclick
    async function loadAdminPanel() {
        try {
            const [invitesResponse, usersResponse] = await Promise.allSettled([
                apiRequest('/admin/invites', { method: 'GET' }),
                listUsers(1, 20)
            ]);
            if (invitesResponse.status === 'fulfilled') {
                renderInvitesList(invitesResponse.value.invites || []);
            } else {
                console.error('Failed to load invites:', invitesResponse.reason);
                const invContainer = document.getElementById('invitesList');
                if (invContainer) {
                    invContainer.innerHTML = '<p style="color: var(--accent); text-align: center;">Ошибка загрузки инвайтов</p>';
                }
            }
            if (usersResponse.status === 'fulfilled') {
                renderUsersList(usersResponse.value.users || []);
            } else {
                console.error('Failed to load users:', usersResponse.reason);
                const usersContainer = document.getElementById('usersList');
                if (usersContainer) {
                    usersContainer.innerHTML = '<p style="color: var(--accent); text-align: center;">Ошибка загрузки пользователей</p>';
                }
            }
        } catch (e) {
            console.error('Failed to load admin panel:', e);
        }
    }
    function renderInvitesList(invites) {
        const container = document.getElementById('invitesList');
        if (!container) return;
        if (!invites || !invites.length) {
            container.innerHTML = '<p style="color: var(--text-secondary); text-align: center;">Нет инвайтов</p>';
            return;
        }
        container.innerHTML = invites.map(inv => `
<div style="background: var(--bg-card); padding: 16px; border-radius: var(--radius-md);">
<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
<code style="font-family: monospace; color: var(--accent); font-size: 14px;">${inv.code}</code>
<span style="padding: 4px 8px; background: ${inv.is_active ? 'var(--green)' : 'var(--text-tertiary)'}; color: white; border-radius: 12px; font-size: 12px;">
${inv.is_active ? 'Активен' : 'Отозван'}
</span>
</div>
<div style="font-size: 13px; color: var(--text-secondary); margin-bottom: 8px;">
Роль: <strong>${inv.role}</strong> · Использований: ${inv.used_count}/${inv.max_uses}
</div>
<div style="display: flex; gap: 8px;">
<button data-action="copy-invite" data-url="${inv.invite_url}" class="btn-secondary" style="flex: 1; font-size: 13px;">
📋 Копировать ссылку
</button>
${inv.is_active ? `<button data-action="revoke-invite" data-code="${inv.code}" class="btn-secondary" style="flex: 1; color: var(--accent); font-size: 13px;">
❌ Отозвать
</button>` : ''}
</div>
</div>
`).join('');
    }
    function renderUsersList(users) {
        const container = document.getElementById('usersList');
        if (!container) return;
        if (!users || !users.length) {
            container.innerHTML = '<p style="color: var(--text-secondary); text-align: center;">Нет пользователей</p>';
            return;
        }
        container.innerHTML = `
<div style="display: grid; grid-template-columns: 1fr auto auto auto; gap: 8px; padding: 12px; background: var(--bg-surface); border-radius: var(--radius-sm); font-weight: 600; font-size: 13px; color: var(--text-secondary);">
<div>Пользователь</div><div>Роль</div><div>Создан</div><div>Обновлён</div>
</div>
` + users.map(u => `
<div style="display: grid; grid-template-columns: 1fr auto auto auto; gap: 8px; padding: 12px; background: var(--bg-card); border-radius: var(--radius-md); font-size: 14px; align-items: center;">
<div>
<strong>${u.full_name || u.email || 'Без имени'}</strong>
<div style="font-size: 12px; color: var(--text-secondary);">${u.email || ''}</div>
</div>
<div><span style="padding: 4px 8px; background: ${u.role === 'admin' ? 'var(--purple)' : 'var(--blue)'}; color: white; border-radius: 12px; font-size: 12px;">${u.role}</span></div>
<div style="color: var(--text-secondary); font-size: 12px;">${u.created_at ? new Date(u.created_at).toLocaleDateString('ru-RU') : ''}</div>
<div style="color: var(--text-secondary); font-size: 12px;">${u.updated_at ? new Date(u.updated_at).toLocaleDateString('ru-RU') : ''}</div>
</div>
`).join('');
    }
    async function _createNewInvite() {
        const role = document.getElementById('newInviteRole')?.value || 'client';
        const maxUses = parseInt(document.getElementById('newInviteMaxUses')?.value || '1', 10);
        try {
            const result = await apiRequest('/admin/invites', {
                method: 'POST',
                body: JSON.stringify({ role: role, max_uses: maxUses })
            });
            showToast(`Инвайт создан: ${result.code}`, 'success');
            _copyToClipboard(result.invite_url);
            loadAdminPanel();
        } catch (e) {
            console.error('Failed to create invite:', e);
            showToast('Ошибка создания инвайта: ' + (e.message || 'неизвестная ошибка'), 'error');
        }
    }
    async function _revokeInvite(code) {
        if (!confirm('Отозвать этот инвайт?')) return;
        try {
            await apiRequest(`/admin/invites/${code}/revoke`, {
                method: 'POST'
            });
            showToast('Инвайт отозван', 'success');
            loadAdminPanel();
        } catch (e) {
            console.error('Failed to revoke invite:', e);
            showToast('Ошибка отзыва: ' + (e.message || 'неизвестная ошибка'), 'error');
        }
    }
    function _copyToClipboard(text) {
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(() => {
                showToast('Ссылка скопирована', 'success');
            }).catch(() => {
                // Fallback для старых браузеров
                _fallbackCopy(text);
            });
        } else {
            _fallbackCopy(text);
        }
    }
    function _fallbackCopy(text) {
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        try {
            document.execCommand('copy');
            showToast('Ссылка скопирована', 'success');
        } catch (err) {
            showToast('Не удалось скопировать', 'error');
        }
        document.body.removeChild(textarea);
    }
    // Экспортируем функции в window для использования в onclick
    window.createNewInvite = _createNewInvite;
    window.revokeInvite = _revokeInvite;
    window.copyToClipboard = _copyToClipboard;
    // Обработчик переключения на админку + делегирование data-action
    // (inline-обработчики заменены на data-action ради strict nonce-based CSP)
    document.addEventListener('click', (e) => {
        if (e.target.closest('[data-view="admin"]')) {
            loadAdminPanel();
        }
        const actionEl = e.target.closest('[data-action]');
        if (!actionEl) return;
        const action = actionEl.dataset.action;
        switch (action) {
            case 'copy-token':
                copyToken();
                break;
            case 'create-invite':
                _createNewInvite();
                break;
            case 'disconnect-provider':
                disconnectProvider(actionEl.dataset.provider);
                break;
            case 'copy-invite':
                _copyToClipboard(actionEl.dataset.url);
                break;
            case 'revoke-invite':
                _revokeInvite(actionEl.dataset.code);
                break;
        }
    });
    function showMainApp() {
        authScreen.classList.remove('active');
        mainScreen.classList.add('active');
        mainScreen.classList.remove('hidden');
        switchView('dashboard');
        // Проверяем роль админа после входа
        checkAdminRole();
        if (window.AppModules && window.AppModules.TrainingModule) {
            window.AppModules.TrainingModule.loadPlans();
        }
        if (window.AppModules && window.AppModules.HealthModule) {
            window.AppModules.HealthModule.bindEvents();
        }
        console.log('[APP] Main app shown');
    }
    // ===== Validation =====
    const validators = {
        email: (v) => {
            if (!v) return 'Введите email';
            if (v.length > 254) return 'Email слишком длинный';
            if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)) return 'Некорректный формат email';
            return '';
        },
        loginPassword: (v) => {
            if (!v) return 'Введите пароль';
            return '';
        },
        password: (v) => {
            const checks = {
                length: v.length >= 8,
                upper: /[A-ZА-ЯЁ]/.test(v),
                lower: /[a-zа-яё]/.test(v),
                digit: /\d/.test(v),
            };
            if (!v) return { error: 'Введите пароль', checks };
            if (!checks.length) return { error: 'Минимум 8 символов', checks };
            return { error: '', checks };
        },
        name: (v) => {
            if (!v) return 'Введите имя';
            if (v.length < 2) return 'Минимум 2 символа';
            if (v.length > 100) return 'Максимум 100 символов';
            if (!/^[A-Za-zА-Яа-яЁё\s\-]+$/.test(v)) return 'Только буквы';
            return '';
        },
        // Никнейм: обязательно, уникальность проверяется на сервере
        nickname: (v) => {
            if (!v || !v.trim()) return 'Никнейм обязателен';
            if (v.trim().length < 2) return 'Минимум 2 символа';
            if (v.trim().length > 30) return 'Максимум 30 символов';
            if (!/^[A-Za-zА-Яа-яЁё0-9_\s\-]+$/.test(v.trim())) return 'Только буквы, цифры, _ и -';
            return '';
        },
        // Возраст — только целые цифры
        age: (v) => {
            if (!v) return '';
            if (!/^\d+$/.test(v)) return 'Только целые цифры';
            const n = parseInt(v, 10);
            if (n < 18 || n > 100) return 'От 18 до 100';
            return '';
        },
        // Рост — только целые цифры
        height: (v) => {
            if (!v) return '';
            if (!/^\d+$/.test(v)) return 'Только целые цифры';
            const n = parseInt(v, 10);
            if (n < 50 || n > 300) return 'От 50 до 300 см';
            return '';
        },
        // Вес — цифры с десятичной точкой
        weight: (v) => {
            if (!v) return '';
            if (!/^\d+(\.\d{1,2})?$/.test(v)) return 'Число (например, 70.5)';
            const n = parseFloat(v);
            if (n < 20 || n > 500) return 'От 20 до 500 кг';
            return '';
        },
    };
    function setFieldError(input, errorEl, msg) {
        if (!input) return;
        input.classList.toggle('invalid', !!msg);
        input.classList.toggle('valid', !msg && input.value.length > 0);
        if (errorEl) errorEl.textContent = msg || '';
    }
    function updatePasswordHint(result) {
        const hint = document.getElementById('passwordHint');
        if (!hint) return;
        hint.classList.toggle('hidden', !result || !result.checks);
        if (!result) return;
        const items = {
            hintLength: result.checks.length,
            hintUpper: result.checks.upper,
            hintLower: result.checks.lower,
            hintDigit: result.checks.digit,
        };
        for (const [id, pass] of Object.entries(items)) {
            const el = document.getElementById(id);
            if (el) {
                el.classList.toggle('pass', pass);
                el.textContent = (pass ? '✓ ' : '✗ ') + el.textContent.slice(2);
            }
        }
        const btn = document.getElementById('registerBtn');
        if (btn) btn.disabled = !!result.error;
    }
    // ===== BMI Calculation =====
    function calculateAndShowBMI() {
        const heightCm = parseFloat(document.getElementById('profHeight')?.value) || 0;
        const weightKg = parseFloat(document.getElementById('profWeight')?.value) || 0;
        const bmiHint = document.getElementById('bmiHint');
        const bmiValue = document.getElementById('bmiValue');
        const bmiCategory = document.getElementById('bmiCategory');
        const bmiRecommendation = document.getElementById('bmiRecommendation');

        if (heightCm > 0 && weightKg > 0) {
            const heightM = heightCm / 100;
            const bmi = weightKg / (heightM * heightM);
            let category = '';
            let recommendation = '';
            let recommendedGoal = '';

            if (bmi < 18.5) {
                category = 'Недостаточный вес';
                recommendation = 'Рекомендуется набор мышечной массы.';
                recommendedGoal = 'muscle_gain';
            } else if (bmi < 25) {
                category = 'Нормальный вес';
                recommendation = 'Ваш вес в норме. Можно выбрать любую цель.';
                recommendedGoal = '';
            } else if (bmi < 30) {
                category = 'Избыточный вес';
                recommendation = 'Рекомендуется снижение веса.';
                recommendedGoal = 'weight_loss';
            } else {
                category = 'Ожирение';
                recommendation = 'Настоятельно рекомендуется снижение веса.';
                recommendedGoal = 'weight_loss';
            }

            if (bmiValue) bmiValue.textContent = bmi.toFixed(1);
            if (bmiCategory) bmiCategory.textContent = category;
            if (bmiRecommendation) bmiRecommendation.textContent = recommendation;
            if (bmiHint) bmiHint.style.display = 'block';

            // Автоматически предлагаем цель, если она еще не выбрана
            if (recommendedGoal) {
                const goalRadio = document.querySelector(`input[name="goal"][value="${recommendedGoal}"]`);
                if (goalRadio && !document.querySelector('input[name="goal"]:checked')) {
                    goalRadio.checked = true;
                }
            }
        } else {
            if (bmiHint) bmiHint.style.display = 'none';
        }
    }

    async function submitLogin2FA(isBackupCode) {
        const errorEl = document.getElementById('login2FAError');
        const btn = document.getElementById('login2FABtn');
        const backupBtn = document.getElementById('useBackupLoginBtn');
        const codeInput = document.getElementById(isBackupCode ? 'backupLoginCode' : 'totpLoginCode');
        const code = (codeInput?.value || '').trim();
        if (errorEl) errorEl.classList.add('hidden');
        if (!state.twoFATempToken) {
            if (errorEl) {
                errorEl.textContent = 'Сессия 2FA истекла. Войдите заново.';
                errorEl.classList.remove('hidden');
            }
            return;
        }
        if (!code) {
            if (errorEl) {
                errorEl.textContent = isBackupCode ? 'Введите резервный код' : 'Введите 6-значный код';
                errorEl.classList.remove('hidden');
            }
            return;
        }
        if (btn) btn.disabled = true;
        if (backupBtn) backupBtn.disabled = true;
        try {
            const data = await verify2FA(state.twoFATempToken, code, isBackupCode);
            setAuthToken(data.access_token);
            showToast('Вход выполнен', 'success');
            showMainApp();
        } catch (err) {
            if (errorEl) {
                errorEl.textContent = err.message || 'Неверный код 2FA';
                errorEl.classList.remove('hidden');
            }
        } finally {
            if (btn) btn.disabled = false;
            if (backupBtn) backupBtn.disabled = false;
        }
    }

    // ===== Events =====
    function bindEvents() {
        // --- Fields ---
        const loginEmail = document.getElementById('loginEmail');
        const loginPassword = document.getElementById('loginPassword');
        const loginEmailErr = document.getElementById('loginEmailError');
        const loginPassErr = document.getElementById('loginPasswordError');
        const loginErrEl = document.getElementById('loginError');
        const regName = document.getElementById('regName');
        const regEmail = document.getElementById('regEmail');
        const regPassword = document.getElementById('regPassword');
        const regNameErr = document.getElementById('regNameError');
        const regEmailErr = document.getElementById('regEmailError');
        const regPassErr = document.getElementById('regPasswordError');
        const regErrEl = document.getElementById('registerError');
        console.log('[APP] Elements:', { loginForm: !!loginForm, registerForm: !!registerForm, loginEmail: !!loginEmail });
        // Login field validation
        if (loginEmail) loginEmail.addEventListener('input', () => {
            setFieldError(loginEmail, loginEmailErr, validators.email(loginEmail.value));
            if (loginErrEl) loginErrEl.classList.add('hidden');
        });
        if (loginPassword) loginPassword.addEventListener('input', () => {
            setFieldError(loginPassword, loginPassErr, validators.loginPassword(loginPassword.value));
            if (loginErrEl) loginErrEl.classList.add('hidden');
        });
        // Register field validation
        if (regName) regName.addEventListener('input', () => {
            setFieldError(regName, regNameErr, validators.name(regName.value));
            if (regErrEl) regErrEl.classList.add('hidden');
        });
        if (regEmail) regEmail.addEventListener('input', () => {
            setFieldError(regEmail, regEmailErr, validators.email(regEmail.value));
            if (regErrEl) regErrEl.classList.add('hidden');
        });
        if (regPassword) regPassword.addEventListener('input', () => {
            const result = validators.password(regPassword.value);
            const err = typeof result === 'object' ? result.error : result;
            setFieldError(regPassword, regPassErr, err);
            updatePasswordHint(typeof result === 'object' ? result : null);
            if (regErrEl) regErrEl.classList.add('hidden');
        });
        // Auth toggle
        document.getElementById('toRegister')?.addEventListener('click', e => {
            e.preventDefault();
            console.log('[APP] Switch to register');
            loginForm.classList.add('hidden');
            registerForm.classList.remove('hidden');
            clearErrors();
        });
        document.getElementById('toLogin')?.addEventListener('click', e => {
            e.preventDefault();
            console.log('[APP] Switch to login');
            registerForm.classList.add('hidden');
            loginForm.classList.remove('hidden');
            if (verifyForm) verifyForm.classList.add('hidden');
            clearErrors();
        });
        document.getElementById('backToLogin')?.addEventListener('click', e => {
            e.preventDefault();
            console.log('[APP] Back to login from verify');
            loginForm.classList.remove('hidden');
            if (verifyForm) verifyForm.classList.add('hidden');
            clearErrors();
        });
        document.getElementById('backToLoginFrom2FA')?.addEventListener('click', e => {
            e.preventDefault();
            state.twoFATempToken = null;
            showAuthScreen();
        });
        document.getElementById('login2FABtn')?.addEventListener('click', async e => {
            e.preventDefault();
            await submitLogin2FA(false);
        });
        document.getElementById('useBackupLoginBtn')?.addEventListener('click', async e => {
            e.preventDefault();
            await submitLogin2FA(true);
        });
        document.getElementById('confirmBtn')?.addEventListener('click', async (e) => {
            e.preventDefault();
            const token = document.getElementById('verifyToken').value.trim();
            const confirmErr = document.getElementById('confirmError');
            const confirmOk = document.getElementById('confirmSuccess');
            const btn = document.getElementById('confirmBtn');
            if (!token) {
                if (confirmErr) { confirmErr.textContent = 'Вставьте токен'; confirmErr.classList.remove('hidden'); }
                return;
            }
            if (btn) { btn.disabled = true; btn.textContent = 'Подтверждение...'; }
            if (confirmErr) confirmErr.classList.add('hidden');
            if (confirmOk) confirmOk.classList.add('hidden');
            try {
                await confirmEmail(token);
                if (confirmOk) confirmOk.classList.remove('hidden');
                showToast('Email подтверждён! Теперь войдите.', 'success');
                setTimeout(() => {
                    loginForm.classList.remove('hidden');
                    if (verifyForm) verifyForm.classList.add('hidden');
                }, 2000);
            } catch (err) {
                if (confirmErr) { confirmErr.textContent = err.message; confirmErr.classList.remove('hidden'); }
            } finally {
                if (btn) { btn.disabled = false; btn.textContent = 'Подтвердить email'; }
            }
        });
        // ===== LOGIN SUBMIT =====
        if (loginForm) {
            loginForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                console.log('[LOGIN] Submit!');
                const email = loginEmail.value.trim();
                const password = loginPassword.value;
                const emailErr = validators.email(email);
                const passErr = validators.loginPassword(password);
                setFieldError(loginEmail, loginEmailErr, emailErr);
                setFieldError(loginPassword, loginPassErr, passErr);
                if (emailErr || passErr) {
                    console.log('[LOGIN] Validation failed');
                    if (loginErrEl) { loginErrEl.textContent = 'Проверьте введённые данные'; loginErrEl.classList.remove('hidden'); }
                    return;
                }
                const btn = document.getElementById('loginBtn');
                if (btn) { btn.disabled = true; btn.textContent = 'Вход...'; }
                try {
                    console.log('[LOGIN] Calling API for:', email);
                    const data = await login(email, password);
                    console.log('[LOGIN] Got response type:', typeof data, 'keys:', data ? Object.keys(data) : 'null');
                    if (data && typeof data === 'object' && data.access_token) {
                        setAuthToken(data.access_token);
                        state.isAdmin = data.role === 'admin';
                        showMainApp();
                    } else if (data && data.requires_2fa && data.temp_token) {
                        showLogin2FA(data.temp_token);
                    } else {
                        console.error('[LOGIN] Unexpected response:', data);
                        const msg = data && data.message ? data.message : 'Сервер вернул неожиданный ответ. Попробуйте войти позже.';
                        throw new Error(msg);
                    }
                } catch (err) {
                    console.error('[LOGIN] Error:', err);
                    if (loginErrEl) { loginErrEl.textContent = err.message; loginErrEl.classList.remove('hidden'); }
                } finally {
                    if (btn) { btn.disabled = false; btn.textContent = 'Войти'; }
                }
            });
        }
        // ===== REGISTER SUBMIT =====
        if (registerForm) {
            registerForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                console.log('[REGISTER] Submit!');
                const name = regName.value.trim();
                const email = regEmail.value.trim();
                const password = regPassword.value;
                const nameErr = validators.name(name);
                const emailErr = validators.email(email);
                const passResult = validators.password(password);
                const passErr = typeof passResult === 'object' ? passResult.error : passResult;
                setFieldError(regName, regNameErr, nameErr);
                setFieldError(regEmail, regEmailErr, emailErr);
                setFieldError(regPassword, regPassErr, passErr);
                if (nameErr || emailErr || passErr) {
                    console.log('[REGISTER] Validation failed');
                    if (regErrEl) { regErrEl.textContent = 'Проверьте введённые данные'; regErrEl.classList.remove('hidden'); }
                    return;
                }
                const btn = document.getElementById('registerBtn');
                if (btn) { btn.disabled = true; btn.textContent = 'Создание...'; }
                try {
                    console.log('[REGISTER] Calling API for:', email, name);
                    const data = await register(email, password, name);
                    console.log('[REGISTER] Got response:', data);
                    showVerification(email, data.message || '', data.user_id);
                } catch (err) {
                    console.error('[REGISTER] Error:', err);
                    if (regErrEl) { regErrEl.textContent = err.message; regErrEl.classList.remove('hidden'); regErrEl.style.color = ''; }
                } finally {
                    if (btn) { btn.disabled = false; btn.textContent = 'Создать аккаунт'; }
                }
            });
        }
        // Logout
        document.getElementById('logoutBtn')?.addEventListener('click', async () => {
            await logout();
            setAuthToken(null);
            showAuthScreen();
        });
        // Tab bar
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', () => switchView(tab.dataset.view));
        });
        // Generate plan (training page button only)
        document.getElementById('generatePlanBtn')?.addEventListener('click', generatePlan);
        // ML classify
        document.getElementById('mlClassifyBtn')?.addEventListener('click', mlClassify);
        // Profile save
        document.getElementById('profileForm')?.addEventListener('submit', saveProfile);
        // Change password button
        document.getElementById('changePasswordBtn')?.addEventListener('click', () => {
            document.getElementById('changePasswordModal').classList.remove('hidden');
        });
        document.getElementById('enable2FABtn')?.addEventListener('click', start2FASetup);
        document.getElementById('confirm2FABtn')?.addEventListener('click', confirm2FASetup);
        document.getElementById('disable2FABtn')?.addEventListener('click', disable2FAFlow);
        // Delete profile button
        document.getElementById('deleteProfileBtn')?.addEventListener('click', () => {
            document.getElementById('deleteProfileModal').classList.remove('hidden');
        });
        // Cancel change password
        document.getElementById('cancelChangePassword')?.addEventListener('click', () => {
            document.getElementById('changePasswordModal').classList.add('hidden');
            document.getElementById('changePasswordForm').reset();
        });
        // Cancel delete profile
        document.getElementById('cancelDeleteProfile')?.addEventListener('click', () => {
            document.getElementById('deleteProfileModal').classList.add('hidden');
            document.getElementById('deleteConfirmPassword').value = '';
        });
        // Change password form submit
        document.getElementById('changePasswordForm')?.addEventListener('submit', handleChangePassword);
        // Change email button
        document.getElementById('changeEmailBtn')?.addEventListener('click', () => {
            document.getElementById('changeEmailModal').classList.remove('hidden');
        });
        // Cancel change email
        document.getElementById('cancelChangeEmail')?.addEventListener('click', () => {
            document.getElementById('changeEmailModal').classList.add('hidden');
            document.getElementById('changeEmailForm').reset();
        });
        // Change email form submit
        document.getElementById('changeEmailForm')?.addEventListener('submit', handleChangeEmail);
        // Confirm delete profile
        document.getElementById('confirmDeleteProfile')?.addEventListener('click', handleDeleteProfile);
        // Close modal on overlay click
        document.querySelectorAll('.modal-overlay').forEach(overlay => {
            overlay.addEventListener('click', (e) => {
                e.target.closest('.modal').classList.add('hidden');
            });
        });
        // Profile field validation
        const profNickname = document.getElementById('profNickname');
        const profAge = document.getElementById('profAge');
        const profHeight = document.getElementById('profHeight');
        const profWeight = document.getElementById('profWeight');
        if (profNickname) profNickname.addEventListener('input', () => {
            setFieldError(profNickname, document.getElementById('profNicknameError'), validators.nickname(profNickname.value));
        });
        if (profAge) profAge.addEventListener('input', () => {
            setFieldError(profAge, document.getElementById('profAgeError'), validators.age(profAge.value));
        });
        if (profHeight) profHeight.addEventListener('input', () => {
            setFieldError(profHeight, document.getElementById('profHeightError'), validators.height(profHeight.value));
            calculateAndShowBMI(); // Пересчитываем ИМТ при изменении роста
        });
        if (profWeight) profWeight.addEventListener('input', () => {
            setFieldError(profWeight, document.getElementById('profWeightError'), validators.weight(profWeight.value));
            calculateAndShowBMI(); // Пересчитываем ИМТ при изменении веса
        });
    }

    async function load2FAStatus() {
        const statusEl = document.getElementById('twoFAStatus');
            const enableBtn = document.getElementById('enable2FABtn');
            const disableBtn = document.getElementById('disable2FABtn');
            const disablePanel = document.getElementById('disable2FAPanel');
            const setupPanel = document.getElementById('totpSetupPanel');
        try {
            const status = await get2FAStatus();
            const enabled = status.enabled === true;
            if (statusEl) statusEl.textContent = enabled ? `Включена. Осталось резервных кодов: ${status.backup_codes_remaining}` : 'Не включена';
            if (enableBtn) enableBtn.classList.toggle('hidden', enabled);
            if (disableBtn) disableBtn.classList.toggle('hidden', !enabled);
            if (disablePanel) disablePanel.classList.toggle('hidden', !enabled);
            if (setupPanel && enabled) setupPanel.classList.add('hidden');
        } catch (err) {
            if (statusEl) statusEl.textContent = 'Не удалось загрузить статус 2FA';
            if (enableBtn) enableBtn.classList.remove('hidden');
            if (disableBtn) disableBtn.classList.add('hidden');
            if (disablePanel) disablePanel.classList.add('hidden');
        }
    }

    async function start2FASetup() {
        const panel = document.getElementById('totpSetupPanel');
        const errorEl = document.getElementById('totpSetupError');
        const successEl = document.getElementById('totpSetupSuccess');
        if (panel) panel.classList.add('hidden');
        if (errorEl) errorEl.classList.add('hidden');
        if (successEl) successEl.classList.add('hidden');
        try {
            const data = await setup2FA();
            const secret = (data.secret || '').replace(/(.{4})/g, '$1 ').trim();
            document.getElementById('totpQRCode').src = data.qr_code_base64 || '';
            document.getElementById('totpManualSecret').textContent = secret;
            const backupList = document.getElementById('totpBackupCodes');
            if (backupList) {
                backupList.innerHTML = (data.backup_codes || []).map(code => `<li><code>${code}</code></li>`).join('');
            }
            if (panel) panel.classList.remove('hidden');
        } catch (err) {
            if (errorEl) {
                errorEl.textContent = err.message || 'Не удалось начать настройку 2FA';
                errorEl.classList.remove('hidden');
            }
        }
    }

    async function confirm2FASetup() {
        const errorEl = document.getElementById('totpSetupError');
        const successEl = document.getElementById('totpSetupSuccess');
        const btn = document.getElementById('confirm2FABtn');
        const secret = document.getElementById('totpManualSecret')?.textContent.replace(/\s+/g, '') || '';
        const backupCodes = Array.from(document.querySelectorAll('#totpBackupCodes code')).map(el => el.textContent.trim());
        const passcode = (document.getElementById('totpSetupCode')?.value || '').trim();
        if (errorEl) errorEl.classList.add('hidden');
        if (successEl) successEl.classList.add('hidden');
        if (!/^\d{6}$/.test(passcode)) {
            if (errorEl) {
                errorEl.textContent = 'Введите 6-значный код из приложения';
                errorEl.classList.remove('hidden');
            }
            return;
        }
        if (btn) btn.disabled = true;
        try {
            await confirm2FA(passcode, secret, backupCodes);
            if (successEl) successEl.textContent = '2FA включена. Сохраните резервные коды в надёжном месте.';
            if (successEl) successEl.classList.remove('hidden');
            const panel = document.getElementById('totpSetupPanel');
            if (panel) panel.classList.add('hidden');
            await load2FAStatus();
        } catch (err) {
            if (errorEl) {
                errorEl.textContent = err.message || 'Неверный код подтверждения';
                errorEl.classList.remove('hidden');
            }
        } finally {
            if (btn) btn.disabled = false;
        }
    }

    async function disable2FAFlow() {
        const passcode = document.getElementById('disable2FACode')?.value.trim();
        const errorEl = document.getElementById('disable2FAError');
        if (errorEl) errorEl.classList.add('hidden');
        if (!passcode) {
            if (errorEl) {
                errorEl.textContent = 'Введите текущий код 2FA для отключения';
                errorEl.classList.remove('hidden');
            }
            return;
        }
        try {
            await disable2FA(passcode);
            showToast('2FA отключена', 'success');
            document.getElementById('disable2FACode').value = '';
            await load2FAStatus();
        } catch (err) {
            if (errorEl) {
                errorEl.textContent = err.message || 'Не удалось отключить 2FA';
                errorEl.classList.remove('hidden');
            }
        }
    }

    // ===== Profile =====
    async function loadProfile() {
        // Change Password
        document.getElementById('changePasswordBtn')?.addEventListener('click', () => {
            const form = document.getElementById('changePasswordForm');
            if (form) form.classList.remove('hidden');
        });
        document.getElementById('cancelChangePassword')?.addEventListener('click', () => {
            const form = document.getElementById('changePasswordForm');
            if (form) {
                form.classList.add('hidden');
                form.reset();
                ['currentPasswordError', 'newPasswordError', 'confirmPasswordError'].forEach(id => {
                    const el = document.getElementById(id);
                    if (el) el.textContent = '';
                });
            }
        });
        // New password validation hints
        const newPassInput = document.getElementById('newPassword');
        if (newPassInput) newPassInput.addEventListener('input', () => {
            const v = newPassInput.value;
            const hint = document.getElementById('passwordHint');
            if (hint) hint.classList.toggle('hidden', v.length === 0);
            const checks = {
                length: v.length >= 8,
                upper: /[A-ZА-ЯЁ]/.test(v),
                lower: /[a-zа-яё]/.test(v),
                digit: /\d/.test(v),
            };
            const items = {
                hintLength: checks.length,
                hintUpper: checks.upper,
                hintLower: checks.lower,
                hintDigit: checks.digit,
            };
            for (const [id, pass] of Object.entries(items)) {
                const el = document.getElementById(id);
                if (el) {
                    el.classList.toggle('pass', pass);
                    el.textContent = (pass ? '✓ ' : '✗ ') + el.textContent.slice(2);
                }
            }
            const btn = document.querySelector('#changePasswordForm .btn-primary');
            if (btn) btn.disabled = !Object.values(checks).every(Boolean);
        });
        // Confirm password validation
        const confirmPass = document.getElementById('confirmPassword');
        if (confirmPass) confirmPass.addEventListener('input', () => {
            const newP = document.getElementById('newPassword')?.value || '';
            const err = confirmPass.value !== newP ? 'Пароли не совпадают' : '';
            setFieldError(confirmPass, document.getElementById('confirmPasswordError'), err);
        });
        document.getElementById('changePasswordForm')?.addEventListener('submit', async (e) => {
            e.preventDefault();
            const currentP = document.getElementById('currentPassword').value;
            const newP = document.getElementById('newPassword').value;
            const confirmP = document.getElementById('confirmPassword').value;
            if (!currentP) {
                setFieldError(document.getElementById('currentPassword'), document.getElementById('currentPasswordError'), 'Введите текущий пароль');
                return;
            }
            if (!newP || newP.length < 8) {
                setFieldError(document.getElementById('newPassword'), document.getElementById('newPasswordError'), 'Минимум 8 символов');
                return;
            }
            if (newP !== confirmP) {
                setFieldError(document.getElementById('confirmPassword'), document.getElementById('confirmPasswordError'), 'Пароли не совпадают');
                return;
            }
            const btn = document.querySelector('#changePasswordForm .btn-primary');
            if (btn) { btn.disabled = true; btn.textContent = 'Сохранение...'; }
            try {
                await changePassword(currentP, newP);
                showToast('Пароль успешно изменён', 'success');
                document.getElementById('cancelChangePassword')?.click();
            } catch (err) {
                if (err.message.includes('incorrect')) {
                    setFieldError(document.getElementById('currentPassword'), document.getElementById('currentPasswordError'), 'Неверный текущий пароль');
                } else if (err.message.includes('8 characters') || err.message.includes('uppercase')) {
                    setFieldError(document.getElementById('newPassword'), document.getElementById('newPasswordError'), err.message);
                } else {
                    showToast('Ошибка: ' + err.message, 'error');
                }
            } finally {
                if (btn) { btn.disabled = false; btn.textContent = 'Сохранить новый пароль'; }
            }
        });
        // ===== Delete Profile =====
        document.getElementById('deleteProfileBtn')?.addEventListener('click', async () => {
            if (!confirm('Вы уверены? Это действие необратимо. Все данные будут удалены.')) return;
            if (!confirm('Точно удалить аккаунт?')) return;
            const btn = document.getElementById('deleteProfileBtn');
            if (btn) { btn.disabled = true; btn.textContent = 'Удаление...'; }
            try {
                await deleteProfile();
                showToast('Аккаунт удалён', 'success');
                setTimeout(() => {
                    setAuthToken(null);
                    window.location.reload();
                }, 1500);
            } catch (err) {
                showToast('Ошибка: ' + err.message, 'error');
                if (btn) { btn.disabled = false; btn.textContent = 'Удалить аккаунт'; }
            }
        });

        // ===== FIX: CSP - Убраны inline event handlers =====
        document.getElementById('connectFitbitBtn')?.addEventListener('click', connectFitbit);
        document.getElementById('connectWithingsBtn')?.addEventListener('click', connectWithings);

        // ===== Diet Settings =====
        document.getElementById('applyDietSettingsBtn')?.addEventListener('click', () => {
            if (window.AppModules && window.AppModules.DietModule) {
                window.AppModules.DietModule.loadDietPlan();
                showToast('Настройки диеты применены', 'success');
            }
        });
    }
    function clearErrors() {
        ['loginError', 'registerError', 'authError', 'login2FAError'].forEach(id => {
            const el = document.getElementById(id);
            if (el) { el.textContent = ''; el.classList.add('hidden'); el.style.color = ''; }
        });
        ['loginEmailError', 'loginPasswordError', 'regNameError', 'regEmailError', 'regPasswordError'].forEach(id => {
            const el = document.getElementById(id);
            if (el) el.textContent = '';
        });
    }
    // ===== Navigation =====
    function switchView(viewName) {
        state.currentView = viewName;
        document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
        const targetView = document.getElementById(`${viewName}View`);
        if (targetView) targetView.classList.add('active');
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        const activeTab = document.querySelector(`.tab[data-view="${viewName}"]`);
        if (activeTab) activeTab.classList.add('active');
        pageTitle.textContent = viewTitles[viewName] || 'FitPulse';
        if (viewName === 'dashboard') loadDashboard();
        if (viewName === 'profile') loadProfile();
        if (viewName === 'training') loadTrainingPlans();
        if (viewName === 'ml') loadMLView();
        if (viewName === 'devices') initDevicesView();
        if (viewName === 'achievements') loadAchievements();
        if (viewName === 'diet') initDietView();
        if (viewName === 'health' && window.AppModules && window.AppModules.HealthModule) {
            window.AppModules.HealthModule.loadAll();
            window.AppModules.HealthModule.bindEvents();
        }
    }
    // ===== Dashboard =====
    async function loadDashboard() {
        try {
            const [hrData, spo2Data] = await Promise.allSettled([
                getBiometricRecords('heart_rate', null, null, 10),
                getBiometricRecords('spo2', null, null, 5),
            ]);
            if (hrData.status === 'fulfilled' && hrData.value.records?.length > 0) {
                document.getElementById('hrValue').textContent = Math.round(hrData.value.records[0].value);
            }
            if (spo2Data.status === 'fulfilled' && spo2Data.value.records?.length > 0) {
                document.getElementById('spo2Value').textContent = Math.round(spo2Data.value.records[0].value);
            }
            // Chart
            if (hrData.status === 'fulfilled' && hrData.value.records?.length > 1) {
                const records = hrData.value.records.slice(0, 20).reverse();
                const labels = records.map(r => new Date(r.timestamp).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' }));
                const values = records.map(r => r.value);
                if (state.heartChart) state.heartChart.destroy();
                const ctx = document.getElementById('heartChart')?.getContext('2d');
                if (ctx) {
                    state.heartChart = new Chart(ctx, {
                        type: 'line',
                        data: {
                            labels,
                            datasets: [{
                                data: values, borderColor: '#ff375f', backgroundColor: 'rgba(255,55,95,0.1)',
                                fill: true, tension: 0.4, pointRadius: 0, borderWidth: 2.5,
                            }]
                        },
                        options: {
                            responsive: true, maintainAspectRatio: false,
                            plugins: { legend: { display: false } },
                            scales: {
                                x: { display: true, grid: { display: false }, ticks: { color: '#636366', maxTicksLimit: 6, font: { size: 11 } } },
                                y: { display: true, grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#636366', font: { size: 11 } } }
                            }
                        }
                    });
                }
            }
            // AI recommendation
            try {
                const classifyRes = await apiRequest('/ml/classify', { method: 'POST', body: '{}' });
                console.log('[Dashboard] ML classify result:', classifyRes);
                if (classifyRes && classifyRes.predicted_class_ru) {
                    document.getElementById('aiRecommendation').textContent = classifyRes.predicted_class_ru;
                    document.getElementById('aiDescription').textContent =
                        `${classifyRes.description || ''}`;
                } else if (classifyRes && classifyRes.predicted_class) {
                    document.getElementById('aiRecommendation').textContent = classifyRes.predicted_class;
                    document.getElementById('aiDescription').textContent = 'AI анализ требует больше данных';
                }
            } catch (err) {
                console.error('[Dashboard] ML classify error:', err);
                document.getElementById('aiRecommendation').textContent = 'Ошибка анализа';
                document.getElementById('aiDescription').textContent = 'Сервис AI временно недоступен';
            }
            // Today's workout - load active training plan
            try {
                let plansData = await getTrainingPlans(1, 1);
                if (typeof plansData === 'string') {
                    plansData = JSON.parse(plansData);
                }
                const plans = plansData?.plans || [];
                if (plans.length > 0) {
                    const plan = plans[0];
                    let todayWorkoutHtml = '';
                    // Try to load full plan details
                    try {
                        const fullPlan = await fetch(`/api/v1/training/plans/${plan.plan_id}`, {
                            headers: { 'Authorization': `Bearer ${localStorage.getItem('authToken')}` }
                        }).then(r => r.json());
                        const planData = fullPlan?.plan_data;
                        if (planData?.weeks && planData.weeks.length > 0) {
                            // Get today's day of week (0 = Sunday, 1 = Monday, etc)
                            const today = new Date().getDay();
                            // Search for today's workout in the first week
                            let todayWorkout = null;
                            for (const week of planData.weeks) {
                                for (const day of week.days || []) {
                                    if (day.day_of_week === today) {
                                        todayWorkout = day;
                                        break;
                                    }
                                }
                                if (todayWorkout) break;
                            }
                            if (todayWorkout) {
                                const trainingTypes = {
                                    'cardio': '🏃 Кардио',
                                    'strength': '💪 Силовая',
                                    'recovery': '🧘 Восстановление',
                                    'endurance': '🏃 Выносливость',
                                    'hiit': 'HIIT'
                                };
                                const exercises = todayWorkout.exercises || [];
                                const typeLabel = trainingTypes[todayWorkout.training_type] || '';
                                let exercisesHtml = '';
                                if (exercises.length > 0) {
                                    exercisesHtml = '<ul style="margin: 10px 0; padding-left: 20px;">' +
                                        exercises.map(ex => {
                                            const details = [];
                                            if (ex.sets) details.push(`${ex.sets}x${ex.reps}`);
                                            if (ex.duration) details.push(`${ex.duration}мин`);
                                            return `<li>${EXERCISE_NAME_MAP[ex.exercise_name] || ex.exercise_name || ''} ${details.length > 0 ? '(' + details.join(', ') + ')' : ''}</li>`;
                                        }).join('') +
                                        '</ul>';
                                }
                                todayWorkoutHtml = `
<div class="workout-content">
<h4>${typeLabel}</h4>
${exercisesHtml}
${todayWorkout.duration ? `<p> Длительность: ${todayWorkout.duration} мин</p>` : ''}
${todayWorkout.notes ? `<p>${todayWorkout.notes}</p>` : ''}
</div>
`;
                            }
                        }
                    } catch (e) {
                        console.warn('Could not load full plan details:', e);
                    }
                    // Fallback if no today's workout found
                    if (!todayWorkoutHtml) {
                        todayWorkoutHtml = `
<div class="workout-content">
<h4>😴 Отдых</h4>
<p>Сегодня нет тренировки. Вашему организму нужен отдых для восстановления.</p>
</div>
`;
                    }
                    document.getElementById('todayWorkout').innerHTML = todayWorkoutHtml;
                }
            } catch (err) {
                console.error('Failed to load today workout:', err);
            }
        } catch (err) {
            console.error('Dashboard load failed:', err);
        }
    }
    async function saveProfile(e) {
        e.preventDefault();
        // Валидация никнейма (обязательно)
        const nickname = document.getElementById('profNickname').value.trim();
        const nickErr = validators.nickname(nickname);
        setFieldError(document.getElementById('profNickname'), document.getElementById('profNicknameError'), nickErr);
        if (nickErr) {
            showToast('Ошибка: ' + nickErr, 'error');
            return;
        }
        // Валидация числовых полей
        const ageVal = document.getElementById('profAge').value;
        const heightVal = document.getElementById('profHeight').value;
        const weightVal = document.getElementById('profWeight').value;
        const ageErr = validators.age(ageVal);
        const heightErr = validators.height(heightVal);
        const weightErr = validators.weight(weightVal);
        setFieldError(document.getElementById('profAge'), document.getElementById('profAgeError'), ageErr);
        setFieldError(document.getElementById('profHeight'), document.getElementById('profHeightError'), heightErr);
        setFieldError(document.getElementById('profWeight'), document.getElementById('profWeightError'), weightErr);
        if (ageErr || heightErr || weightErr) {
            showToast('Исправьте ошибки в числовых полях', 'error');
            return;
        }
        // Цель — одна
        const selectedGoal = document.querySelector('.goals-grid input[type="radio"]:checked');
        const goals = selectedGoal ? [selectedGoal.value] : [];

        // Аллергии и противопоказания
        const allergies = document.getElementById('profAllergies')?.value.split(',').map(s => s.trim()).filter(Boolean) || [];
        const contraindications = document.getElementById('profContraindications')?.value.split(',').map(s => s.trim()).filter(Boolean) || [];

        const data = {
            full_name: nickname,
            age: ageVal ? parseInt(ageVal, 10) : null,
            gender: document.getElementById('profGender').value || null,
            height_cm: heightVal ? parseInt(heightVal, 10) : null,
            weight_kg: weightVal ? parseFloat(weightVal) : null,
            fitness_level: document.getElementById('profFitness').value || null,
            nutrition: document.getElementById('profNutrition').value || null,
            goals,
            allergies,
            contraindications
        };
        try {
            await updateProfile(data);
            showToast('Профиль сохранён', 'success');
        } catch (err) {
            console.error('Profile save failed:', err);
            // Если ошибка 503 или foreign key violation, предлагаем перезайти
            if (err.message && (err.message.includes('Недоступен') || err.message.includes('violates foreign key'))) {
                showToast('Ошибка сохранения. Попробуйте перезайти и повторить.', 'error');
            } else {
                showToast('Ошибка: ' + err.message, 'error');
            }
        }
    }
    // ===== Change Password =====
    async function handleChangePassword(e) {
        e.preventDefault();
        const currentPassword = document.getElementById('currentPassword').value;
        const newPassword = document.getElementById('newPassword').value;
        const confirmNewPassword = document.getElementById('confirmNewPassword').value;
        // Validation
        if (!currentPassword) {
            showToast('Введите текущий пароль', 'error');
            return;
        }
        if (!newPassword) {
            showToast('Введите новый пароль', 'error');
            return;
        }
        // Password strength validation
        const passwordChecks = {
            length: newPassword.length >= 8,
            upper: /[A-ZА-ЯЁ]/.test(newPassword),
            lower: /[a-zа-яё]/.test(newPassword),
            digit: /\d/.test(newPassword),
        };
        if (!passwordChecks.length) {
            showToast('Пароль должен содержать минимум 8 символов', 'error');
            return;
        }
        if (!passwordChecks.upper) {
            showToast('Пароль должен содержать заглавную букву', 'error');
            return;
        }
        if (!passwordChecks.lower) {
            showToast('Пароль должен содержать строчную букву', 'error');
            return;
        }
        if (!passwordChecks.digit) {
            showToast('Пароль должен содержать цифру', 'error');
            return;
        }
        if (newPassword !== confirmNewPassword) {
            showToast('Новые пароли не совпадают', 'error');
            return;
        }
        try {
            await changePassword(currentPassword, newPassword);
            showToast('Пароль успешно изменён', 'success');
            document.getElementById('changePasswordModal').classList.add('hidden');
            document.getElementById('changePasswordForm').reset();
        } catch (err) {
            console.error('Change password failed:', err);
            showToast('Ошибка: ' + err.message, 'error');
        }
    }
    // ===== Change Email =====
    async function handleChangeEmail(e) {
        e.preventDefault();
        const newEmail = document.getElementById('newEmail').value;
        const password = document.getElementById('emailConfirmPassword').value;
        if (!newEmail) {
            showToast('Введите новую почту', 'error');
            return;
        }
        if (!password) {
            showToast('Введите пароль для подтверждения', 'error');
            return;
        }
        if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(newEmail)) {
            showToast('Введите корректный email', 'error');
            return;
        }
        try {
            await changeEmail(newEmail, password);
            showToast('Email изменён', 'success');
            document.getElementById('changeEmailModal').classList.add('hidden');
            document.getElementById('changeEmailForm').reset();
        } catch (err) {
            console.error('Change email failed:', err);
            showToast('Ошибка: ' + err.message, 'error');
        }
    }
    // ===== Delete Profile =====
    async function handleDeleteProfile() {
        const password = document.getElementById('deleteConfirmPassword').value;
        if (!password) {
            showToast('Введите пароль для подтверждения', 'error');
            return;
        }
        // Additional confirmation
        if (!confirm('Вы уверены? Это действие нельзя отменить.')) {
            return;
        }
        try {
            await deleteProfile(password);
            showToast('Профиль удалён', 'success');
            // Logout and redirect
            setAuthToken(null);
            window.location.reload();
        } catch (err) {
            console.error('Delete profile failed:', err);
            showToast('Ошибка: ' + err.message, 'error');
        }
    }
    // ===== Training =====
    async function loadTrainingPlans() {
        if (window.AppModules) {
            await window.AppModules.TrainingModule.loadPlans();
        }
    }
    async function generatePlan() {
        if (window.AppModules) {
            await window.AppModules.TrainingModule.generatePlan();
        }
    }
    // ===== ML =====
    async function loadMLView() { }
    async function mlClassify() {
        try {
            const container = document.getElementById('mlResult');
            container.innerHTML = '<div style="text-align:center;padding:40px;color:var(--text-secondary);">Анализ...</div>';
            const result = await apiRequest('/ml/classify', { method: 'POST', body: '{}' });
            const classRu = result.predicted_class_ru || result.predicted_class || 'Не определено';
            const confidence = result.confidence ? Math.round(result.confidence * 100) : 0;
            container.innerHTML = `<div class="ml-classification">
<div class="class-label">Ваше состояние</div>
<div class="class-name">${classRu}</div>
${result.description ? `<p style="margin-top:12px;font-size:15px;color:var(--text-secondary);">${result.description}</p>` : ''}</div>`;
        } catch (err) {
            document.getElementById('mlResult').innerHTML = `<div class="empty-state"><div class="empty-icon">⚠️</div>
<h3>Не удалось проанализировать</h3><p>${err.message}</p></div>`;
        }
    }
    // ===== Achievements =====
    async function loadAchievements() {
        const container = document.getElementById('achievementsList');
        const compContainer = document.getElementById('competitionsList');
        if (!container || !compContainer) return;
        let achievements = [];
        try {
            const data = await getAchievements();
            achievements = (data && data.achievements) ? data.achievements : [];
        } catch (e) {
            console.warn('Failed to load achievements:', e);
        }
        const iconMap = {
            first_workout: '🏃',
            week_streak: '🔥',
            ten_workouts: '💪',
            fifty_workouts: '⭐',
            hundred_days: '📊',
            master_sport: '🏆',
        };
        container.innerHTML = achievements.map(a => {
            const icon = a.icon_url || iconMap[a.achievement_id] || '🏆';
            const unlocked = !!a.earned_date;
            return `
<div class="achievement-card ${unlocked ? 'unlocked' : 'locked'}">
<div class="achievement-icon">${icon}</div>
<div class="achievement-name">${a.title || ''}</div>
<div class="achievement-desc">${a.description || ''}</div>
${unlocked ? '<div class="achievement-progress">Получено</div>' : '<div class="achievement-progress">Заблокировано</div>'}
</div>
`;}).join('');
        if (!achievements.length) {
            container.innerHTML = '<p style="color: var(--text-secondary); text-align: center;">Нет достижений</p>';
        }
        // FIX: Убраны фейковые соревнования, заменены на персональные челленджи
        const competitions = [
            { name: 'Персональный рекорд', desc: 'Пройдите 10000 шагов за день', status: 'active', participants: 1, rank: null },
            { name: 'Серия тренировок', desc: 'Тренируйтесь 3 дня подряд', status: 'upcoming', participants: 1, rank: null },
            { name: 'Месяц активности', desc: 'Тренируйтесь 20 дней в месяце', status: 'upcoming', participants: 1, rank: null }
        ];
        const statusLabels = { active: 'Активно', upcoming: 'Скоро', finished: 'Завершено' };
        compContainer.innerHTML = competitions.map(c => `
<div class="competition-card">
<div class="competition-header">
<div class="competition-name">${c.name}</div>
<span class="competition-status ${c.status}">${statusLabels[c.status]}</span>
</div>
<div class="competition-desc">${c.desc}</div>
<div class="competition-meta">
<span>Персональный челлендж</span>
${c.rank ? `<span class="competition-rank">🏅 Место: ${c.rank}</span>` : ''}
</div>
</div>
`).join('');
        // Прогресс-график
        try {
            const progress = await getProgress();
            const progressData = progress?.progress_data || progress?.data || [];
            if (progressData.length > 0 && typeof Chart !== 'undefined') {
                const ctx = document.getElementById('progressChart')?.getContext('2d');
                if (ctx) {
                    const labels = progressData.map(p => p.date || p.week || '');
                    const values = progressData.map(p => p.completed_workouts ?? p.count ?? p.value ?? 0);
                    new Chart(ctx, {
                        type: 'bar',
                        data: {
                            labels,
                            datasets: [{
                                label: 'Тренировок',
                                data: values,
                                backgroundColor: 'rgba(255,55,95,0.6)',
                                borderRadius: 8
                            }]
                        },
                        options: {
                            responsive: true,
                            plugins: { legend: { display: false } },
                            scales: {
                                y: { beginAtZero: true, ticks: { color: '#8e8e93' }, grid: { color: '#2c2c2e' } },
                                x: { ticks: { color: '#8e8e93' }, grid: { display: false } }
                            }
                        }
                    });
                }
            }
        } catch (e) {
            console.warn('Failed to load progress chart:', e);
        }
    }
    function initDevicesView() {
        if (window.AppModules) {
            window.AppModules.DeviceModule.init();
        }
        // Загружаем список подключённых провайдеров при открытии вкладки
        loadConnectedProviders();
    }
    // ===== Diet View =====
    function initDietView() {
        if (window.AppModules) {
            window.AppModules.DietModule.loadDietPlan();
        }
    }
    // ===== Toast =====
    function showToast(msg, type = 'success') {
        const existing = document.querySelector('.toast');
        if (existing) existing.remove();
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = msg;
        document.body.appendChild(toast);
        setTimeout(() => toast.remove(), 3000);
    }
    // В конце файла, после других экспортов:
    window.disconnectProvider = disconnectProvider;
    // ===== Start =====
    init();
});