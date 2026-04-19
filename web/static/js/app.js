document.addEventListener('DOMContentLoaded', () => {
    console.log('[APP] Loaded');

    // ===== State =====
    const state = {
        currentView: 'dashboard',
        heartChart: null,
        isAdmin: false,
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
        ml: 'AI Анализ',
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
        if (verifyForm) verifyForm.classList.add('hidden');
        clearErrors();
        console.log('[APP] Auth screen shown');
    }

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

    function showMainApp() {
        authScreen.classList.remove('active');
        mainScreen.classList.add('active');
        mainScreen.classList.remove('hidden');
        switchView('dashboard');
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
        });
        if (profWeight) profWeight.addEventListener('input', () => {
            setFieldError(profWeight, document.getElementById('profWeightError'), validators.weight(profWeight.value));
        });

        // ===== Change Password =====
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
    }

    function clearErrors() {
        ['loginError', 'registerError', 'authError'].forEach(id => {
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
                if (classifyRes.predicted_class_ru) {
                    document.getElementById('aiRecommendation').textContent = classifyRes.predicted_class_ru;
                    document.getElementById('aiDescription').textContent =
                        `Уверенность: ${Math.round(classifyRes.confidence * 100)}% | ${classifyRes.description || ''}`;
                }
            } catch {
                document.getElementById('aiRecommendation').textContent = 'Нужно больше данных';
                document.getElementById('aiDescription').textContent = 'Добавьте биометрические данные для AI-анализа';
            }
        } catch (err) {
            console.error('Dashboard load failed:', err);
        }
    }

    // ===== Profile =====
    async function loadProfile() {
        try {
            const profile = await getProfile();
            const p = profile.profile || profile;
            // Никнейм — используем full_name с бэкенда
            document.getElementById('profNickname').value = p.full_name || '';
            document.getElementById('profAge').value = p.age || '';
            document.getElementById('profGender').value = p.gender || '';
            document.getElementById('profHeight').value = p.height_cm || '';
            document.getElementById('profWeight').value = p.weight_kg || '';
            document.getElementById('profFitness').value = p.fitness_level || '';

            // Питание — select
            if (p.nutrition) {
                document.getElementById('profNutrition').value = p.nutrition;
            }

            // Сон — показываем с устройства
            if (p.sleep_hours) {
                const sleepDisplay = document.getElementById('profSleepDisplay');
                const sleepValue = document.getElementById('profSleepValue');
                if (sleepDisplay && sleepValue) {
                    sleepValue.textContent = p.sleep_hours + ' ч';
                }
            }

            // Цель — radio (одна)
            const goal = Array.isArray(p.goals) && p.goals.length > 0 ? p.goals[0] : '';
            document.querySelectorAll('.goals-grid input[type="radio"]').forEach(radio => {
                radio.checked = radio.value === goal;
            });
        } catch (err) {
            console.error('Profile load failed:', err);
            // Если профиль не найден (404), возможно сессия устарела
            if (err.message && err.message.includes('Не найдено')) {
                showToast('Сессия устарела. Пожалуйста, перезайдите в систему.', 'error');
                // Автоматически очищаем токен и предлагаем перезайти
                setTimeout(() => {
                    if (confirm('Ваша сессия устарела. Перезайти?')) {
                        setAuthToken(null);
                        window.location.reload();
                    }
                }, 1000);
            }
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

        const data = {
            full_name: nickname,
            age: ageVal ? parseInt(ageVal, 10) : null,
            gender: document.getElementById('profGender').value || null,
            height_cm: heightVal ? parseInt(heightVal, 10) : null,
            weight_kg: weightVal ? parseFloat(weightVal) : null,
            fitness_level: document.getElementById('profFitness').value || null,
            nutrition: document.getElementById('profNutrition').value || null,
            goals,
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
    async function loadMLView() {}

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
                <div class="confidence">Уверенность: ${confidence}%</div>
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

        // Заглушки достижений
        const achievements = [
            { icon: '🏃', name: 'Первый шаг', desc: 'Завершите первую тренировку', unlocked: false },
            { icon: '🔥', name: 'Серия 7 дней', desc: 'Тренируйтесь 7 дней подряд', unlocked: false },
            { icon: '💪', name: '10 тренировок', desc: 'Завершите 10 тренировок', unlocked: false },
            { icon: '⭐', name: '50 тренировок', desc: 'Завершите 50 тренировок', unlocked: false },
            { icon: '🎯', name: 'Точная цель', desc: 'Достигните поставленной цели', unlocked: false },
            { icon: '📊', name: 'Аналитик', desc: 'Записывайте данные 30 дней', unlocked: false },
        ];

        container.innerHTML = achievements.map(a => `
            <div class="achievement-card ${a.unlocked ? 'unlocked' : 'locked'}">
                <div class="achievement-icon">${a.icon}</div>
                <div class="achievement-name">${a.name}</div>
                <div class="achievement-desc">${a.desc}</div>
                ${a.unlocked ? '<div class="achievement-progress">✅ Получено</div>' : '<div class="achievement-progress">🔒 Заблокировано</div>'}
            </div>
        `).join('');

        // Заглушки соревнований
        const competitions = [
            { name: 'Неделя активности', desc: 'Кто наберёт больше шагов за неделю', status: 'active', participants: 124, rank: null },
            { name: 'Марафон выносливости', desc: '30 дней кардио тренировок', status: 'upcoming', participants: 89, rank: null },
            { name: 'Новогодний челлендж', desc: 'Тренировки каждый день в январе', status: 'finished', participants: 256, rank: 42 },
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
                    <span>👥 ${c.participants} участников</span>
                    ${c.rank ? `<span class="competition-rank">🏅 Место: ${c.rank}</span>` : ''}
                </div>
            </div>
        `).join('');
    }

    // ===== Devices View =====
    function initDevicesView() {
        if (window.AppModules) {
            window.AppModules.DeviceModule.init();
        }
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

    // ===== Start =====
    init();
});
