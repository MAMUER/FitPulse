import { useEffect, useMemo, useState } from 'react';
import { getProfile } from '../../utils/api';
import { calculateBMI } from '../../utils/validators';
import './Diet.css';

const MEAL_TEMPLATES = {
  balanced: {
    name: 'Сбалансированное',
    breakfast: [
      {
        name: 'Овсяная каша с бананом',
        kcal: 350,
        protein: 12,
        carbs: 56,
        fat: 7,
      },
      {
        name: 'Гречневая каша с яйцом',
        kcal: 380,
        protein: 18,
        carbs: 42,
        fat: 12,
      },
      {
        name: 'Тосты с авокадо и яйцом',
        kcal: 400,
        protein: 16,
        carbs: 30,
        fat: 24,
      },
    ],
    snack1: [
      { name: 'Йогурт с ягодами', kcal: 150, protein: 8, carbs: 22, fat: 4 },
      { name: 'Фрукты и орехи', kcal: 180, protein: 5, carbs: 20, fat: 10 },
      { name: 'Творог с мёдом', kcal: 200, protein: 15, carbs: 18, fat: 8 },
    ],
    lunch: [
      {
        name: 'Куриная грудка с рисом',
        kcal: 450,
        protein: 35,
        carbs: 50,
        fat: 10,
      },
      {
        name: 'Суп с овощами и курицей',
        kcal: 380,
        protein: 25,
        carbs: 30,
        fat: 15,
      },
      {
        name: 'Паста с томатным соусом',
        kcal: 420,
        protein: 18,
        carbs: 55,
        fat: 12,
      },
    ],
    snack2: [
      {
        name: 'Протеиновый коктейль',
        kcal: 200,
        protein: 25,
        carbs: 15,
        fat: 4,
      },
      {
        name: 'Яблоко с арахисовым маслом',
        kcal: 220,
        protein: 6,
        carbs: 25,
        fat: 12,
      },
      {
        name: 'Батончики из цельнозерна',
        kcal: 180,
        protein: 8,
        carbs: 25,
        fat: 7,
      },
    ],
    dinner: [
      {
        name: 'Запечённый лосось с овощами',
        kcal: 450,
        protein: 35,
        carbs: 20,
        fat: 25,
      },
      { name: 'Индейка с киноа', kcal: 420, protein: 32, carbs: 35, fat: 14 },
      { name: 'Салат с тунцом', kcal: 350, protein: 28, carbs: 15, fat: 20 },
    ],
  },
  high_protein: {
    name: 'Высокобелковое',
    breakfast: [
      {
        name: 'Омлет с сыром и ветчиной',
        kcal: 400,
        protein: 30,
        carbs: 4,
        fat: 28,
      },
      {
        name: 'Творожная запеканка',
        kcal: 380,
        protein: 28,
        carbs: 30,
        fat: 14,
      },
      { name: 'Сырники с гречкой', kcal: 420, protein: 25, carbs: 40, fat: 15 },
    ],
    snack1: [
      {
        name: 'Протеиновый коктейль',
        kcal: 200,
        protein: 30,
        carbs: 10,
        fat: 3,
      },
      { name: 'Творог с ягодами', kcal: 180, protein: 22, carbs: 12, fat: 5 },
      { name: 'Яйца всмятку', kcal: 170, protein: 12, carbs: 1, fat: 12 },
    ],
    lunch: [
      { name: 'Стейк с брокколи', kcal: 500, protein: 45, carbs: 15, fat: 30 },
      {
        name: 'Курица гриль с салатом',
        kcal: 450,
        protein: 40,
        carbs: 20,
        fat: 22,
      },
      { name: 'Тунец с спаржей', kcal: 400, protein: 38, carbs: 10, fat: 24 },
    ],
    snack2: [
      {
        name: 'Протеиновые крекеры',
        kcal: 220,
        protein: 20,
        carbs: 20,
        fat: 6,
      },
      { name: 'Творожная паста', kcal: 200, protein: 18, carbs: 8, fat: 10 },
      {
        name: 'Изолят сывороточного протеина',
        kcal: 180,
        protein: 35,
        carbs: 5,
        fat: 2,
      },
    ],
    dinner: [
      {
        name: 'Индейка с брюссельской капустой',
        kcal: 450,
        protein: 42,
        carbs: 18,
        fat: 22,
      },
      {
        name: 'Суп из чечевицы с курицей',
        kcal: 400,
        protein: 32,
        carbs: 35,
        fat: 10,
      },
      { name: 'Стейк семги', kcal: 420, protein: 35, carbs: 5, fat: 28 },
    ],
  },
  weight_loss: {
    name: 'Похудение',
    breakfast: [
      { name: 'Гречка с овощами', kcal: 280, protein: 12, carbs: 38, fat: 6 },
      {
        name: 'Овсяная каша на воде',
        kcal: 220,
        protein: 8,
        carbs: 38,
        fat: 4,
      },
      { name: 'Смузи с протеином', kcal: 250, protein: 20, carbs: 25, fat: 5 },
    ],
    snack1: [
      { name: 'Яблоко', kcal: 80, protein: 0, carbs: 20, fat: 0 },
      { name: 'Морковь и сельдерей', kcal: 60, protein: 1, carbs: 12, fat: 0 },
      { name: 'Ягодный салат', kcal: 100, protein: 2, carbs: 22, fat: 0 },
    ],
    lunch: [
      {
        name: 'Салат с куриной грудкой',
        kcal: 320,
        protein: 30,
        carbs: 15,
        fat: 14,
      },
      { name: 'Суп овощной', kcal: 180, protein: 8, carbs: 20, fat: 4 },
      {
        name: 'Запечённая индейка с брокколи',
        kcal: 350,
        protein: 32,
        carbs: 12,
        fat: 16,
      },
    ],
    snack2: [
      { name: 'Греческий йогурт', kcal: 120, protein: 12, carbs: 8, fat: 4 },
      {
        name: 'Протеиновый батончик',
        kcal: 180,
        protein: 15,
        carbs: 20,
        fat: 6,
      },
      { name: 'Орехи 30г', kcal: 180, protein: 5, carbs: 4, fat: 16 },
    ],
    dinner: [
      { name: 'Салат с тунцом', kcal: 280, protein: 28, carbs: 10, fat: 14 },
      { name: 'Тофу с овощами', kcal: 250, protein: 20, carbs: 15, fat: 12 },
      {
        name: 'Крем-суп из кабачков',
        kcal: 220,
        protein: 10,
        carbs: 15,
        fat: 12,
      },
    ],
  },
};

export default function Diet() {
  const [profile, setProfile] = useState(null);
  const [loading, setLoading] = useState(true);
  const [allergies, setAllergies] = useState('');
  const [dislikes, setDislikes] = useState('');
  const [mealCount, setMealCount] = useState(5);
  const [firstMealTime, setFirstMealTime] = useState('08:00');
  const [template, setTemplate] = useState('balanced');
  const [meals, setMeals] = useState([]);

  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  const loadProfile = async () => {
    try {
      const data = await getProfile();
      setProfile(data);
      const p = data.profile || data;
      setAllergies((p.allergies || []).join(', '));
      setDislikes((p.contraindications || []).join(', '));
      const goal = p.goals?.[0] || '';
      calculateBMI(p.height_cm, p.weight_kg);
      if (goal === 'weight_loss') setTemplate('weight_loss');
      else if (goal === 'muscle_gain') setTemplate('high_protein');
      else setTemplate('balanced');
    } catch (e) {
      console.error('Failed to load profile:', e);
    } finally {
      setLoading(false);
    }
  };

  const nutrition = useMemo(() => {
    if (!profile) return null;
    const p = profile.profile || profile;
    const weight = p.weight_kg || 70;
    const height = p.height_cm || 170;
    const age = p.age || 30;
    const gender = p.gender || 'male';
    const fitness = p.fitness_level || 'beginner';
    const goal = p.goals?.[0] || '';

    const bmr =
      10 * weight + 6.25 * height - 5 * age + (gender === 'male' ? 5 : -161);
    const activityMultipliers = {
      beginner: 1.375,
      intermediate: 1.55,
      advanced: 1.725,
    };
    const tdee = bmr * (activityMultipliers[fitness] || 1.375);
    const goalAdjustments = {
      weight_loss: -400,
      muscle_gain: 300,
      endurance: 100,
    };
    const calories = Math.max(
      1200,
      Math.round(tdee + (goalAdjustments[goal] || 0))
    );

    let proteinPct, fatPct, carbsPct;
    if (goal === 'weight_loss') {
      proteinPct = 0.35;
      fatPct = 0.3;
      carbsPct = 0.35;
    } else if (goal === 'muscle_gain') {
      proteinPct = 0.3;
      fatPct = 0.25;
      carbsPct = 0.45;
    } else {
      proteinPct = 0.25;
      fatPct = 0.3;
      carbsPct = 0.45;
    }

    const proteinG = Math.round((calories * proteinPct) / 4);
    const fatG = Math.round((calories * fatPct) / 9);
    const carbsG = Math.round((calories * carbsPct) / 4);

    return {
      bmr: Math.round(bmr),
      tdee: Math.round(tdee),
      calories,
      proteinG,
      fatG,
      carbsG,
      fitness,
      goal,
    };
  }, [profile]);

  const allergyList = useMemo(
    () =>
      allergies
        .split(',')
        .map((s) => s.trim().toLowerCase())
        .filter(Boolean),
    [allergies]
  );
  const dislikeList = useMemo(
    () =>
      dislikes
        .split(',')
        .map((s) => s.trim().toLowerCase())
        .filter(Boolean),
    [dislikes]
  );

  const filterMeals = useMemo(() => {
    return (mealList) => {
      return mealList.filter((m) => {
        if (allergyList.some((a) => a && m.name.toLowerCase().includes(a)))
          return false;
        if (dislikeList.some((d) => d && m.name.toLowerCase().includes(d)))
          return false;
        return true;
      });
    };
  }, [allergyList, dislikeList]);

  useEffect(() => {
    if (!nutrition) return;
    const selectedTemplate =
      MEAL_TEMPLATES[template] || MEAL_TEMPLATES.balanced;
    const mealKeys = ['breakfast', 'snack1', 'lunch', 'snack2', 'dinner'];
    const selectedMealKeys = mealKeys.slice(0, mealCount);

    const [hours, minutes] = firstMealTime.split(':').map(Number);
    const startMinutes = hours * 60 + minutes;
    const windowMinutes = 14 * 60;
    const step = mealCount > 1 ? windowMinutes / (mealCount - 1) : 0;

    const generated = selectedMealKeys.map((key, idx) => {
      const options = filterMeals(selectedTemplate[key]);
      const meal = options[Math.floor(Math.random() * options.length)] || {
        // NOSONAR javascript:S2245 - UI-only meal selection, not security context
        name: '—',
        kcal: 0,
        protein: 0,
        carbs: 0,
        fat: 0,
      };
      const timeMinutes =
        mealCount > 1 ? startMinutes + idx * step : startMinutes;
      const h = Math.floor(timeMinutes / 60) % 24;
      const m = timeMinutes % 60;
      const time = `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
      return { ...meal, time };
    });

    setMeals(generated);
  }, [nutrition, template, mealCount, firstMealTime, filterMeals]);

  const totals = useMemo(() => {
    return meals.reduce(
      (acc, m) => ({
        kcal: acc.kcal + (m.kcal || 0),
        protein: acc.protein + (m.protein || 0),
        carbs: acc.carbs + (m.carbs || 0),
        fat: acc.fat + (m.fat || 0),
      }),
      { kcal: 0, protein: 0, carbs: 0, fat: 0 }
    );
  }, [meals]);

  if (loading) return <div className='loading'>Загрузка...</div>;
  if (!nutrition)
    return <div className='view active'>Ошибка загрузки профиля</div>;

  return (
    <div className='view active'>
      <div className='diet-summary'>
        <div className='diet-calories'>{nutrition.calories}</div>
        <div className='diet-label'>
          ккал в день · BMR {nutrition.bmr} ·{' '}
          {nutrition.fitness === 'beginner'
            ? 'Начинающий'
            : nutrition.fitness === 'intermediate'
              ? 'Средний'
              : 'Продвинутый'}
        </div>
        <div className='diet-macros'>
          <div className='macro-item'>
            <div className='macro-value'>{nutrition.proteinG}г</div>
            <div className='macro-label'>Белки</div>
          </div>
          <div className='macro-item'>
            <div className='macro-value'>{nutrition.fatG}г</div>
            <div className='macro-label'>Жиры</div>
          </div>
          <div className='macro-item'>
            <div className='macro-value'>{nutrition.carbsG}г</div>
            <div className='macro-label'>Углеводы</div>
          </div>
        </div>
      </div>

      <div className='diet-controls'>
        <h3>Предпочтения питания</h3>
        <div className='form-group'>
          <label>Шаблон рациона</label>
          <div className='meal-template-selector'>
            {Object.entries(MEAL_TEMPLATES).map(([key, tpl]) => (
              <button
                key={key}
                type='button'
                className={`meal-template-btn ${template === key ? 'selected' : ''}`}
                onClick={() => setTemplate(key)}
              >
                {tpl.name}
              </button>
            ))}
          </div>
        </div>
        <div className='form-group'>
          <label>Количество приёмов пищи</label>
          <select
            value={mealCount}
            onChange={(e) => setMealCount(Number(e.target.value))}
          >
            {[3, 4, 5, 6].map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </div>
        <div className='form-group'>
          <label>Время первого приёма</label>
          <input
            type='time'
            value={firstMealTime}
            onChange={(e) => setFirstMealTime(e.target.value)}
          />
        </div>
        <div className='form-group'>
          <label>Аллергии (через запятую)</label>
          <input
            type='text'
            value={allergies}
            onChange={(e) => setAllergies(e.target.value)}
            placeholder='Например: орехи, лактоза'
          />
        </div>
        <div className='form-group'>
          <label>Нелюбимые продукты (через запятую)</label>
          <input
            type='text'
            value={dislikes}
            onChange={(e) => setDislikes(e.target.value)}
            placeholder='Например: брокколи, рыба'
          />
        </div>
      </div>

      <div className='today-section'>
        <h3>План питания на сегодня</h3>
        <div id='dietMealsList'>
          {meals.length === 0 ? (
            <div className='empty-state'>
              <div className='empty-icon'>🍽️</div>
              <h3>Нет подходящих блюд</h3>
              <p>Измените настройки аллергий и предпочтений</p>
            </div>
          ) : (
            meals.map((meal, idx) => (
              <div key={idx} className='meal-card'>
                <div className='meal-time'>{meal.time}</div>
                <div className='meal-name'>{meal.name}</div>
                <div className='meal-details'>
                  {meal.kcal} ккал · Б:{meal.protein}г · Ж:{meal.fat}г · У:
                  {meal.carbs}г
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {meals.length > 0 && (
        <div className='diet-totals'>
          <div>Итого за день</div>
          <div className='totals-row'>
            <div className='totals-item'>
              <div className='totals-value'>{totals.kcal} ккал</div>
              <div className='totals-label'>Калории</div>
            </div>
            <div className='totals-item'>
              <div className='totals-value'>{totals.protein}г</div>
              <div className='totals-label'>Белки</div>
            </div>
            <div className='totals-item'>
              <div className='totals-value'>{totals.fat}г</div>
              <div className='totals-label'>Жиры</div>
            </div>
            <div className='totals-item'>
              <div className='totals-value'>{totals.carbs}г</div>
              <div className='totals-label'>Углеводы</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
