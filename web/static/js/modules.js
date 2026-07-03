// FitPulse Modules — Devices, Training, Diet
// Mobile web app UI logic
const AppModules = (() => {
    // ===== State =====
    let currentUser = null;
    let selectedDevice = null;
    // ===== Device Module =====
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
    const DeviceModule = {
        devices: [
            { type: 'apple_watch', name: 'Apple Watch', icon: '⌚', capabilities: 'Пульс, ЭКГ, SpO₂, Сон' },
            { type: 'samsung_galaxy_watch', name: 'Samsung Galaxy Watch', icon: '⌚', capabilities: 'Пульс, ЭКГ, SpO₂, Температура' },
            { type: 'huawei_watch_d2', name: 'Huawei Watch D2', icon: '⌚', capabilities: 'Пульс, Давление, ЭКГ, SpO₂' },
            { type: 'amazfit_trex3', name: 'Amazfit T-Rex 3', icon: '⌚', capabilities: 'Пульс, SpO₂, Сон' }
        ],
        init() {
            this.renderDeviceSelector();
            this.bindEvents();
            this.renderConnectedDevices();
        },
        renderDeviceSelector() {
            const container = document.getElementById('deviceSelector');
            if (!container) return;
            container.innerHTML = this.devices.map(d => `
<div class="device-option" data-type="${d.type}">
<div class="device-icon">${d.icon}</div>
<div class="device-name">${d.name}</div>
<div class="device-capabilities">${d.capabilities}</div>
</div>
`).join('');
        },
        bindEvents() {
            document.addEventListener('click', (e) => {
                const deviceOption = e.target.closest('.device-option');
                if (deviceOption) {
                    document.querySelectorAll('.device-option').forEach(el => el.classList.remove('selected'));
                    deviceOption.classList.add('selected');
                    selectedDevice = deviceOption.dataset.type;
                    // Показываем какое устройство выбрано
                    const deviceName = this.devices.find(d => d.type === selectedDevice)?.name || 'Устройство';
                    window.AppModules.showToast(`${deviceName} выбрано. Нажмите "Подключить устройство"`, 'info');
                }
            });
            // Кнопка "Подключить устройство"
            document.getElementById('connectDeviceBtn')?.addEventListener('click', async () => {
                if (!selectedDevice) {
                    showToast('Выберите устройство из списка выше', 'error');
                    return;
                }
                const btn = document.getElementById('connectDeviceBtn');
                if (btn) {
                    btn.disabled = true;
                    btn.textContent = 'Подключение...';
                }
                try {
                    const profile = await getProfile();
                    const userId = profile.profile?.user_id || profile.user_id;
                    const result = await this.connectDevice(selectedDevice, userId);
                    showToast(`${this.devices.find(d => d.type === selectedDevice)?.name} подключено!`, 'success');

                    // Отображаем device_id и device_token
                    const container = document.getElementById('connectedDevicesList');
                    if (container && result) {
                        const credsDiv = document.createElement('div');
                        credsDiv.style.cssText = 'margin-top: 12px; padding: 12px; background: var(--bg-input); border-radius: var(--radius-sm); font-size: 12px;';
                        credsDiv.innerHTML = `
        <strong>Данные</strong><br>
        <code>device_id: ${result.device_id || 'N/A'}</code><br>
        <code>device_token: ${result.device_token || 'N/A'}</code>
    `;
                        container.appendChild(credsDiv);
                    }

                    // Обновляем список устройств
                    this.renderConnectedDevices();
                } catch (err) {
                    showToast('Ошибка подключения: ' + err.message, 'error');
                } finally {
                    if (btn) {
                        btn.disabled = false;
                        btn.textContent = 'Подключить устройство';
                    }
                }
            });
        },
        async connectDevice(deviceType, userId) {
            // FIX: Используем apiRequest вместо fetch, путь без /api/v1
            try {
                const data = await window.apiRequest('/devices/register', {
                    method: 'POST',
                    body: JSON.stringify({ device_type: deviceType, user_id: userId })
                });
                return data;
            } catch (err) {
                throw new Error(err.message || 'Ошибка регистрации устройства');
            }
        },
        async renderConnectedDevices() {
            const container = document.getElementById('connectedDevicesList');
            if (!container) return;
            try {
                // FIX: Используем apiRequest вместо fetch, путь без /api/v1
                const data = await window.apiRequest('/devices', { method: 'GET' });
                const devices = data.devices || [];
                if (devices.length === 0) {
                    container.innerHTML = `
<div style="text-align:center; padding:24px 16px; color:var(--text-secondary);">
<div style="font-size:48px; margin-bottom:12px;">⌚</div>
<div style="font-size:15px; font-weight:600; margin-bottom:8px; color:var(--text-primary);">
Нет подключённых устройств
</div>
<div style="font-size:13px; line-height:1.5; max-width:280px; margin:0 auto;">
Выберите устройство из списка ниже и нажмите «Подключить устройство»
</div>
</div>
`;
                    return;
                }
                const deviceNames = {
                    apple_watch: 'Apple Watch',
                    samsung_galaxy_watch: 'Samsung Galaxy Watch',
                    huawei_watch_d2: 'Huawei Watch D2',
                    amazfit_trex3: 'Amazfit T-Rex 3'
                };
                const deviceIcons = {
                    apple_watch: '⌚',
                    samsung_galaxy_watch: '⌚',
                    huawei_watch_d2: '⌚',
                    amazfit_trex3: '⌚'
                };
                container.innerHTML = devices.map(d => `
<div class="device-item">
<div class="device-icon">${deviceIcons[d.device_type] || '⌚'}</div>
<div class="device-info">
<div class="device-name">${deviceNames[d.device_type] || d.device_type}</div>
<div class="device-date">Подключено: ${new Date(d.created_at).toLocaleDateString('ru-RU')}</div>
</div>
</div>
`).join('');
            } catch (err) {
                console.error('Failed to load devices:', err);
                container.innerHTML = `<div class="error">Не удалось загрузить устройства</div>`;
            }
        }
    };
    // ===== Training Module =====
    const TrainingModule = {
        dayNames: ['Воскресенье', 'Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота'],
        shortDay: ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'],
        async loadPlans() {
            const container = document.getElementById('plansList');
            if (!container) {
                return;
            }
            try {
                let data = await getTrainingPlans();
                if (typeof data === 'string') {
                    data = JSON.parse(data);
                }
                const plans = (data && data.plans) || [];
                if (plans.length === 0) {
                    container.innerHTML = `
<div class="empty-state">
<div class="empty-icon">🏃</div>
<h3>Нет активных программ</h3>
<p>AI создаст персональный план на основе ваших данных</p>
</div>
`;
                    return;
                }
                container.innerHTML = `<div class="loading">Загрузка программ...</div>`;
                const dayNames = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'];
                const trainingTypes = {
                    'cardio': '🏃 Кардио',
                    'strength': '💪 Силовая',
                    'recovery': '🧘 Восстановление',
                    'endurance': '🏃 Выносливость',
                    'hiit': 'HIIT'
                };
                let allPlansHtml = '';
                for (const plan of plans) {
                    let planDetails;
                    try {
                        let planData = await getPlan(plan.plan_id);
                        if (typeof planData === 'string') {
                            planData = JSON.parse(planData);
                        }
                        planDetails = planData?.plan;
                    } catch (e) {
                        console.error('Failed to get plan details:', e);
                        planDetails = null;
                    }
                    const planData = plan?.plan_data || {};
                    const fullData = planDetails?.plan_data || planData;
                    const weeks = fullData?.weeks || [];
                    let weeksHtml = '';
                    if (weeks.length > 0) {
                        weeks.forEach(week => {
                            const days = week.days || [];
                            let daysHtml = '';
                            days.forEach(day => {
                                const typeLabel = trainingTypes[day.training_type] || '';
                                const exercises = day.exercises || [];
                                let exercisesHtml = '';
                                if (exercises.length > 0) {
                                    exercisesHtml = '<ul class="exercise-list">' +
                                        exercises.map(ex => `<li>${EXERCISE_NAME_MAP[ex.exercise_name] || ex.exercise_name || ''} ${ex.sets ? `${ex.sets}x${ex.reps}` : ''} ${ex.duration ? `${ex.duration}мін` : ''}</li>`).join('') +
                                        '</ul>';
                                }
                                daysHtml += `
<div class="day-card ${day.is_rest_day ? 'rest-day' : ''}">
<div class="day-header">
<span class="day-name">${dayNames[day.day_of_week] || ''}</span>
<span class="day-type">${day.is_rest_day ? '😴 Отдых' : typeLabel}</span>
</div>
${exercisesHtml}
${day.notes ? `<p class="day-notes">${day.notes}</p>` : ''}
</div>
`;
                            });
                            weeksHtml += `
<div class="week-section">
<h4>Неделя ${week.week_number}</h4>
<div class="days-grid">${daysHtml}</div>
</div>
`;
                        });
                    } else {
                        weeksHtml = `
<div class="week-section">
<p>Программа: ${planData.name || 'Персонализированная программа'}</p>
<p>Цель: ${plan.training_goal || 'Общая тренировка'}</p>
<p>Длительность: ${plan.duration_weeks || 4} недель</p>
</div>
`;
                    }
                    allPlansHtml += `
<div class="plan-full">
<div class="plan-header">
<h3>${fullData.name || 'Персонализированная программа'}</h3>
<span class="plan-status">${status}</span>
</div>
<p class="plan-dates">📅 ${plan.start_date ? new Date(plan.start_date).toLocaleDateString('ru-RU') : '—'} — ${plan.end_date ? new Date(plan.end_date).toLocaleDateString('ru-RU') : '—'}</p>
${weeksHtml}
</div>
`;
                }
                container.innerHTML = allPlansHtml;
            } catch (err) {
                console.error('Failed to load plans:', err);
                container.innerHTML = `<div class="empty-state"><p>Не удалось загрузить планы</p></div>`;
            }
        },
        async generatePlan() {
            const container = document.getElementById('plansList');
            if (!container) return;
            // Show loading
            container.innerHTML = `
<div class="empty-state">
<div class="spinner"></div>
<p>AI генерирует персональный план...</p>
</div>
`;
            try {
                // First, classify current state
                let trainingClass = '';
                let confidence = 0.5;
                try {
                    const classifyRes = await window.apiRequest('/ml/classify', { method: 'POST', body: '{}' });
                    trainingClass = classifyRes.predicted_class || 'recovery';
                    confidence = classifyRes.confidence || 0.5;
                } catch {
                    trainingClass = 'recovery';
                }
                // Get profile for context
                let profile;
                try {
                    profile = await getProfile();
                } catch (err) {
                    if (err.message && err.message.includes('Не найдено')) {
                        throw new Error('Профиль не найден. Попробуйте перезайти в систему.');
                    }
                    throw err;
                }
                const p = profile.profile || profile;
                // Build user_profile object matching backend expectations
                const userProfile = {};
                if (p.age) userProfile.age = p.age;
                if (p.gender) userProfile.gender = p.gender;
                if (p.weight_kg) userProfile.weight = p.weight_kg;
                if (p.height_cm) userProfile.height = p.height_cm;
                if (p.fitness_level) userProfile.fitness_level = p.fitness_level;
                if (p.goals && Array.isArray(p.goals)) userProfile.goals = p.goals;
                if (p.sleep_hours) userProfile.sleep_hours = p.sleep_hours;
                if (p.nutrition) userProfile.nutrition = p.nutrition;
                // Use the Training service endpoint to generate and save a plan
                const plan = await window.apiRequest('/training/generate', {
                    method: 'POST',
                    body: JSON.stringify({
                        class: trainingClass,
                        confidence: confidence,
                        duration_weeks: 4,
                        available_days: [1, 3, 5]
                    })
                });
                showToast('Тренировочный план сгенерирован!', 'success');
                await this.loadPlans();
            } catch (err) {
                console.error('Failed to generate plan:', err);
                container.innerHTML = `
<div class="empty-state">
<div class="empty-icon">⚠️</div>
<h3>Ошибка генерации</h3>
<p>${err.message}</p>
<p style="margin-top:8px; font-size:13px; color:var(--text-tertiary);">
Убедитесь, что ML-сервис запущен
</p>
</div>
`;
            }
        },
        renderPlanDetail(plan) {
            if (!plan || !plan.weeks) return;
            let html = `<h3>${plan.name || 'Тренировочный план'}</h3>`;
            for (const week of plan.weeks) {
                html += `<h4>Неделя ${week.week_number}</h4>`;
                for (const day of week.days || []) {
                    const dayName = this.dayNames[day.day_of_week] || `День ${day.day_of_week}`;
                    html += `<div class="training-plan-card">`;
                    html += `<div class="plan-day-header">`;
                    html += `<div class="plan-day-name">${dayName}</div>`;
                    html += `<div class="plan-day-type">${day.training_type || (day.is_rest_day ? 'Отдых' : 'Тренировка')}</div>`;
                    html += `</div>`;
                    if (day.is_rest_day) {
                        html += `<p style="color: var(--text-secondary); text-align: center; padding: 16px;">😴 День отдыха</p>`;
                    } else if (day.exercises && day.exercises.length > 0) {
                        day.exercises.forEach((ex, i) => {
                            const metaParts = [];
                            if (ex.sets && ex.reps) metaParts.push(`${ex.sets}×${ex.reps}`);
                            if (ex.duration_minutes) metaParts.push(`${ex.duration_minutes} мин`);
                            if (ex.rest_seconds) metaParts.push(`${ex.rest_seconds}с отдых`);
                            html += `
<div class="exercise-item">
<div class="exercise-number">${i + 1}</div>
<div class="exercise-details">
<div class="exercise-name">${EXERCISE_NAME_MAP[ex.exercise_name] || ex.exercise_name || ex.name || 'Упражнение'}</div>
<div class="exercise-meta">${metaParts.join(' • ')}</div>
</div>
</div>
`;
                        });
                    }
                    html += `</div>`;
                }
            }
            return html;
        }
    };
    // ===== Diet Module =====
    const DietModule = {
        mealTemplates: {
            balanced: {
                breakfast: [
                    { name: 'Овсянка с бананом и мёдом', kcal: 350, protein: 12, carbs: 60, fat: 8 },
                    { name: 'Омлет с овощами и тостом', kcal: 380, protein: 22, carbs: 30, fat: 18 },
                    { name: 'Гречневая каша с молоком', kcal: 320, protein: 14, carbs: 55, fat: 6 },
                ],
                snack1: [
                    { name: 'Яблоко + миндаль (30г)', kcal: 200, protein: 6, carbs: 22, fat: 10 },
                    { name: 'Греческий йогурт', kcal: 150, protein: 15, carbs: 10, fat: 5 },
                ],
                lunch: [
                    { name: 'Куриная грудка с рисом и салатом', kcal: 550, protein: 40, carbs: 60, fat: 15 },
                    { name: 'Говядина с гречкой и овощами', kcal: 580, protein: 38, carbs: 55, fat: 18 },
                    { name: 'Рыба (лосось) с бурым рисом', kcal: 520, protein: 35, carbs: 50, fat: 18 },
                ],
                snack2: [
                    { name: 'Протеиновый батончик', kcal: 200, protein: 20, carbs: 22, fat: 8 },
                    { name: 'Творог с ягодами', kcal: 180, protein: 18, carbs: 15, fat: 5 },
                ],
                dinner: [
                    { name: 'Индейка с овощами на пару', kcal: 400, protein: 35, carbs: 25, fat: 18 },
                    { name: 'Запечённая треска с брокколи', kcal: 350, protein: 30, carbs: 20, fat: 15 },
                    { name: 'Куриное филе с авокадо-салатом', kcal: 420, protein: 32, carbs: 15, fat: 22 },
                ],
            },
            high_protein: {
                breakfast: [
                    { name: 'Омлет из 4 яиц с курицей', kcal: 450, protein: 40, carbs: 5, fat: 28 },
                    { name: 'Протеиновые панкейки', kcal: 380, protein: 35, carbs: 30, fat: 12 },
                ],
                snack1: [
                    { name: 'Протеиновый коктейль', kcal: 200, protein: 30, carbs: 8, fat: 4 },
                ],
                lunch: [
                    { name: 'Двойная порция курицы с рисом', kcal: 650, protein: 55, carbs: 55, fat: 18 },
                ],
                snack2: [
                    { name: 'Творог 5% + орехи', kcal: 250, protein: 22, carbs: 10, fat: 14 },
                ],
                dinner: [
                    { name: 'Стейк из лосося с овощами', kcal: 500, protein: 40, carbs: 15, fat: 28 },
                ],
            },
            weight_loss: {
                breakfast: [
                    { name: 'Овсянка на воде с ягодами', kcal: 220, protein: 8, carbs: 40, fat: 4 },
                ],
                snack1: [
                    { name: 'Огурец + хумус', kcal: 100, protein: 4, carbs: 12, fat: 4 },
                ],
                lunch: [
                    { name: 'Куриный суп с овощами', kcal: 300, protein: 25, carbs: 30, fat: 8 },
                ],
                snack2: [
                    { name: 'Зелёное яблоко', kcal: 70, protein: 0, carbs: 18, fat: 0 },
                ],
                dinner: [
                    { name: 'Запечённая белая рыба с салатом', kcal: 280, protein: 30, carbs: 10, fat: 12 },
                ],
            }
        },
        /**
        * Mifflin-St Jeor formula
        * Men: 10*weight + 6.25*height - 5*age + 5
        * Women: 10*weight + 6.25*height - 5*age - 161
        */
        calculateBMR(weightKg, heightCm, age, gender) {
            const bmr = 10 * weightKg + 6.25 * heightCm - 5 * age;
            return gender === 'male' ? bmr + 5 : bmr - 161;
        },
        calculate(profile) {
            const age = profile.age || 30;
            const gender = profile.gender || 'male';
            const heightCm = profile.height_cm || 175;
            const weightKg = profile.weight_kg || 75;
            const fitnessLevel = profile.fitness_level || 'intermediate';
            const goals = profile.goals || [];
            // Activity multiplier based on fitness level
            const multipliers = { beginner: 1.375, intermediate: 1.55, advanced: 1.725 };
            const activityFactor = multipliers[fitnessLevel] || 1.55;
            // Goal adjustment
            let goalAdjust = 0;
            if (goals.includes('weight_loss')) goalAdjust = -400;
            if (goals.includes('muscle_gain')) goalAdjust = 300;
            if (goals.includes('endurance')) goalAdjust = 100;
            const bmr = this.calculateBMR(weightKg, heightCm, age, gender);
            let tdee = Math.round(bmr * activityFactor + goalAdjust);
            // Ensure minimum calories
            tdee = Math.max(tdee, 1200);
            // Macro split based on goal
            let proteinRatio, fatRatio, carbsRatio;
            if (goals.includes('weight_loss')) {
                proteinRatio = 0.35; fatRatio = 0.30; carbsRatio = 0.35;
            } else if (goals.includes('muscle_gain')) {
                proteinRatio = 0.30; fatRatio = 0.25; carbsRatio = 0.45;
            } else {
                proteinRatio = 0.25; fatRatio = 0.30; carbsRatio = 0.45;
            }
            const proteinGrams = Math.round((tdee * proteinRatio) / 4);
            const fatGrams = Math.round((tdee * fatRatio) / 9);
            const carbsGrams = Math.round((tdee * carbsRatio) / 4);
            // Pick diet type
            let dietType = 'balanced';
            if (goals.includes('weight_loss')) dietType = 'weight_loss';
            else if (goals.includes('muscle_gain')) dietType = 'high_protein';
            return { tdee, bmr: Math.round(bmr), proteinGrams, fatGrams, carbsGrams, dietType, goals, fitnessLevel };
        },
        async loadDietPlan() {
            const container = document.getElementById('dietPlanContainer');
            if (!container) return;
            try {
                const profile = await getProfile();
                const p = profile.profile || profile;
                // Проверяем, заполнен ли профиль
                const hasProfile = p.age && p.height_cm && p.weight_kg && p.gender;
                if (!hasProfile) {
                    container.innerHTML = `
<div class="empty-state">
<div class="empty-icon">🍽️</div>
<h3>Заполните профиль</h3>
<p>Укажите возраст, рост, вес и пол во вкладке «Профиль» для расчёта плана питания</p>
</div>
`;
                    return;
                }
                const diet = this.calculate(p);

                // FIX: Получаем настройки диеты из UI
                const allergies = (document.getElementById('dietAllergies')?.value || '').split(',').map(s => s.trim().toLowerCase()).filter(Boolean);
                const dislikes = (document.getElementById('dietDislikes')?.value || '').split(',').map(s => s.trim().toLowerCase()).filter(Boolean);
                const mealsCount = parseInt(document.getElementById('dietMealsCount')?.value || '4', 10);
                const firstMealTime = document.getElementById('dietFirstMealTime')?.value || '08:00';

                // Pick meals based on diet type
                const meals = this.mealTemplates[diet.dietType] || this.mealTemplates.balanced;

                // FIX: Фильтруем блюда по аллергиям и нелюбимым продуктам
                const randomMeal = (arr) => {
                    const filtered = arr.filter(meal => {
                        const name = meal.name.toLowerCase();
                        return !allergies.some(a => name.includes(a)) && !dislikes.some(d => name.includes(d));
                    });
                    if (filtered.length === 0) return arr[Math.floor(Math.random() * arr.length)]; // Fallback, если все блюда исключены
                    return filtered[Math.floor(Math.random() * filtered.length)];
                };

                const breakfast = randomMeal(meals.breakfast);
                const snack1 = randomMeal(meals.snack1);
                const lunch = randomMeal(meals.lunch);
                const snack2 = randomMeal(meals.snack2);
                const dinner = randomMeal(meals.dinner);

                // FIX: Генерируем расписание на основе количества приемов пищи и времени начала
                const mealSchedule = this.generateMealSchedule(mealsCount, firstMealTime);
                const mealNames = ['Завтрак', 'Перекус', 'Обед', 'Полдник', 'Ужин', 'Поздний перекус'];
                const mealIcons = ['🌅', '🍎', '☀️', '🥜', '🌙', '🌌'];
                const selectedMeals = [breakfast, snack1, lunch, snack2, dinner];

                let mealsHtml = '';
                for (let i = 0; i < Math.min(mealsCount, selectedMeals.length); i++) {
                    const meal = selectedMeals[i];
                    const time = mealSchedule[i] || '--:--';
                    mealsHtml += `
    <div class="meal-card">
        <div class="meal-time">${mealIcons[i]} ${time} — ${mealNames[i]}</div>
        <div class="meal-name">${meal.name}</div>
        <div class="meal-details">${meal.kcal} ккал • ${meal.protein}г белка • ${meal.carbs}г углеводов • ${meal.fat}г жиров</div>
    </div>
    `;
                }

                // Подсчет итогов только для выбранных приемов пищи
                const totalKcal = selectedMeals.slice(0, mealsCount).reduce((sum, m) => sum + m.kcal, 0);
                const totalProtein = selectedMeals.slice(0, mealsCount).reduce((sum, m) => sum + m.protein, 0);
                const totalCarbs = selectedMeals.slice(0, mealsCount).reduce((sum, m) => sum + m.carbs, 0);
                const totalFat = selectedMeals.slice(0, mealsCount).reduce((sum, m) => sum + m.fat, 0);

                container.innerHTML = `
<div class="diet-summary">
<div class="diet-calories">${diet.tdee.toLocaleString()}</div>
<div class="diet-label">калорий в день (расчёт по Миффлину-Сан Жеору)</div>
<div class="diet-macros">
<div class="macro-item">
<div class="macro-value">${diet.proteinGrams}г</div>
<div class="macro-label">Белки</div>
</div>
<div class="macro-item">
<div class="macro-value">${diet.carbsGrams}г</div>
<div class="macro-label">Углеводы</div>
</div>
<div class="macro-item">
<div class="macro-value">${diet.fatGrams}г</div>
<div class="macro-label">Жиры</div>
</div>
</div>
</div>
<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-bottom: 16px; padding: 0 4px;">
<div style="background: var(--bg-card); border-radius: var(--radius-sm); padding: 10px; text-align: center;">
<div style="font-size: 12px; color: var(--text-secondary);">Базовый обмен</div>
<div style="font-size: 18px; font-weight: 700; color: var(--blue);">${diet.bmr} ккал</div>
</div>
<div style="background: var(--bg-card); border-radius: var(--radius-sm); padding: 10px; text-align: center;">
<div style="font-size: 12px; color: var(--text-secondary);">Уровень</div>
<div style="font-size: 18px; font-weight: 700; color: var(--green);">${diet.fitnessLevel}</div>
</div>
</div>
${mealsHtml}
<div style="text-align: center; padding: 12px; color: var(--text-secondary); font-size: 13px;">
Итого: ${totalKcal} ккал •
${totalProtein}г белка •
${totalCarbs}г углеводов •
${totalFat}г жиров
</div>
`;
            } catch (err) {
                console.error('Failed to load diet plan:', err);
                container.innerHTML = `
<div class="empty-state">
<div class="empty-icon">🍽️</div>
<h3>Не удалось загрузить диету</h3>
<p>Заполните профиль для расчёта питания</p>
</div>
`;
            }
        },
        // FIX: Генерация расписания приемов пищи
        generateMealSchedule(count, startTime) {
            const [startHour, startMinute] = startTime.split(':').map(Number);
            const intervalMinutes = Math.floor((14 * 60) / (count - 1 || 1)); // 14 часов активности
            const schedule = [];
            for (let i = 0; i < count; i++) {
                const totalMinutes = startHour * 60 + startMinute + i * intervalMinutes;
                const hour = Math.floor(totalMinutes / 60) % 24;
                const minute = totalMinutes % 60;
                schedule.push(`${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`);
            }
            return schedule;
        }
    };
    const HealthModule = {
        CONDITIONS: {
            allergy: 'Аллергия',
            disease: 'Заболевание',
            disability: 'Инвалидность',
            other: 'Другое'
        },
        empty(label) {
            return `<div class="empty-state"><div class="empty-icon">📋</div><h3>Нет данных</h3><p>${label}</p></div>`;
        },
        renderConditions(list) {
            const container = document.getElementById('healthConditionsList');
            if (!container) return;
            if (!list || !list.length) {
                container.innerHTML = this.empty('Добавьте особенности здоровья');
                return;
            }
            container.innerHTML = list.map(item => `
                <div class="health-card" data-id="${item.id}">
                    <div class="health-card-header">
                        <strong>${item.condition_name}</strong>
                        <span class="badge">${this.CONDITIONS[item.condition_type] || item.condition_type}</span>
                    </div>
                    <div class="health-card-meta">
                        ${item.severity ? `<span>Серьёзность: ${item.severity}</span>` : ''}
                        ${item.diagnosed_at ? `<span>Дата: ${item.diagnosed_at}</span>` : ''}
                        <span>Активно: ${item.is_active ? 'Да' : 'Нет'}</span>
                    </div>
                    ${item.notes ? `<p class="health-card-notes">${item.notes}</p>` : ''}
                    <div class="health-card-actions">
                        <button class="btn-danger-ghost" data-action="delete" data-id="${item.id}">Удалить</button>
                    </div>
                </div>
            `).join('');
        },
        renderBodyComposition(list) {
            const container = document.getElementById('bodyCompositionList');
            if (!container) return;
            if (!list || !list.length) {
                container.innerHTML = this.empty('Добавьте запись веса/состава тела');
                return;
            }
            container.innerHTML = list.map(item => `
                <div class="health-card" data-id="${item.id}">
                    <div class="health-card-header">
                        <strong>${item.recorded_at ? new Date(item.recorded_at).toLocaleString('ru-RU') : 'Запись'}</strong>
                        <span class="badge">${item.source === 'okok' ? 'OKOK' : 'Ручной ввод'}</span>
                    </div>
                    <div class="health-card-meta">
                        ${item.weight_kg ? `<span>Вес: ${item.weight_kg} кг</span>` : ''}
                        ${item.height_cm ? `<span>Рост: ${item.height_cm} см</span>` : ''}
                        ${item.bmi ? `<span>ИМТ: ${item.bmi}</span>` : ''}
                    </div>
                    <div class="health-card-meta">
                        ${item.body_fat_percentage ? `<span>Жир: ${item.body_fat_percentage}%</span>` : ''}
                        ${item.muscle_mass_percentage ? `<span>Мышцы: ${item.muscle_mass_percentage}%</span>` : ''}
                    </div>
                </div>
            `).join('');
        },
        renderMenstrualCycles(list) {
            const container = document.getElementById('menstrualCyclesList');
            if (!container) return;
            if (!list || !list.length) {
                container.innerHTML = this.empty('Добавьте запись цикла');
                return;
            }
            container.innerHTML = list.map(item => `
                <div class="health-card" data-id="${item.id}">
                    <div class="health-card-header">
                        <strong>${item.cycle_start_date}${item.cycle_end_date ? ` — ${item.cycle_end_date}` : ''}</strong>
                        <span class="badge">${item.flow_intensity || '—'}</span>
                    </div>
                    <div class="health-card-meta">
                        ${item.symptoms && item.symptoms.length ? `<span>Симптомы: ${item.symptoms.join(', ')}</span>` : ''}
                        ${item.moods && item.moods.length ? `<span>Настроение: ${item.moods.join(', ')}</span>` : ''}
                    </div>
                    ${item.notes ? `<p class="health-card-notes">${item.notes}</p>` : ''}
                    <div class="health-card-actions">
                        <button class="btn-danger-ghost" data-action="delete-cycle" data-id="${item.id}">Удалить</button>
                    </div>
                </div>
            `).join('');
        },
        async loadConditions() {
            try {
                const data = await window.listHealthConditions();
                this.renderConditions(data.conditions || []);
            } catch (err) {
                console.error('Failed to load health conditions:', err);
            }
        },
        async loadBodyComposition() {
            try {
                const data = await window.listBodyComposition();
                this.renderBodyComposition(data.records || []);
            } catch (err) {
                console.error('Failed to load body composition:', err);
            }
        },
        async loadMenstrualCycles() {
            try {
                const data = await window.listMenstrualCycles();
                this.renderMenstrualCycles(data.cycles || []);
            } catch (err) {
                console.error('Failed to load menstrual cycles:', err);
            }
        },
        async loadAll() {
            await this.loadConditions();
            await this.loadBodyComposition();
            await this.loadMenstrualCycles();
        },
        bindEvents() {
            document.getElementById('addConditionBtn')?.addEventListener('click', () => {
                const name = prompt('Название состояния:');
                if (!name) return;
                const type = prompt('Тип (allergy, disease, disability, other):', 'other') || 'other';
                window.upsertHealthCondition({ condition_type: type, condition_name: name, is_active: true })
                    .then(() => this.loadConditions())
                    .catch(err => showToast('Ошибка: ' + err.message, 'error'));
            });
            document.getElementById('addBodyCompositionBtn')?.addEventListener('click', () => {
                const weightKg = parseFloat(prompt('Вес, кг:'));
                if (!weightKg || weightKg <= 0) return;
                const heightCm = prompt('Рост, см:');
                window.createBodyComposition({
                    weight_kg: weightKg,
                    height_cm: heightCm ? parseInt(heightCm, 10) : null,
                    source: 'manual'
                }).then(() => this.loadBodyComposition())
                  .catch(err => showToast('Ошибка: ' + err.message, 'error'));
            });
            document.getElementById('addMenstrualCycleBtn')?.addEventListener('click', () => {
                const startDate = prompt('Дата начала цикла (YYYY-MM-DD):');
                if (!startDate) return;
                window.createMenstrualCycle({ cycle_start_date: startDate })
                    .then(() => this.loadMenstrualCycles())
                    .catch(err => showToast('Ошибка: ' + err.message, 'error'));
            });
            document.getElementById('syncFloBtn')?.addEventListener('click', async () => {
                const accessToken = prompt('Flo access_token:');
                const refreshToken = prompt('Flo refresh_token:');
                if (!accessToken) return;
                try {
                    const res = await window.syncFlo(accessToken, refreshToken || '');
                    showToast(`Синхронизировано циклов: ${res.synced_cycles || 0}`, 'success');
                } catch (err) {
                    showToast('Ошибка синхронизации Flo: ' + err.message, 'error');
                }
            });
            document.getElementById('syncOKOKBtn')?.addEventListener('click', async () => {
                const accessToken = prompt('OKOK access_token:');
                const refreshToken = prompt('OKOK refresh_token:');
                if (!accessToken) return;
                try {
                    const res = await window.syncOKOK(accessToken, refreshToken || '');
                    showToast(`Синхронизировано записей: ${res.synced_records || 0}`, 'success');
                } catch (err) {
                    showToast('Ошибка синхронизации OKOK: ' + err.message, 'error');
                }
            });
            document.getElementById('healthView')?.addEventListener('click', (e) => {
                const deleteBtn = e.target.closest('[data-action="delete"]');
                const deleteCycleBtn = e.target.closest('[data-action="delete-cycle"]');
                if (deleteBtn) {
                    const id = deleteBtn.dataset.id;
                    if (!confirm('Удалить запись?')) return;
                    window.deleteHealthCondition(id).then(() => this.loadConditions()).catch(err => showToast('Ошибка: ' + err.message, 'error'));
                }
                if (deleteCycleBtn) {
                    const id = deleteCycleBtn.dataset.id;
                    if (!confirm('Удалить цикл?')) return;
                    window.deleteMenstrualCycle(id).then(() => this.loadMenstrualCycles()).catch(err => showToast('Ошибка: ' + err.message, 'error'));
                }
            });
        }
    };
    // ===== Toast Notifications =====
    function showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `module-toast ${type}`;
        toast.textContent = message;
        document.body.appendChild(toast);
        setTimeout(() => {
            toast.remove();
        }, 3000);
    }
    // ===== Init =====
    function init(user) {
        currentUser = user;
        DeviceModule.init();
        if (HealthModule && HealthModule.bindEvents) {
            HealthModule.bindEvents();
        }
    }
    return {
        init,
        DeviceModule,
        TrainingModule,
        DietModule,
        HealthModule,
        showToast
    };
})();
// Export to global scope
window.AppModules = AppModules;