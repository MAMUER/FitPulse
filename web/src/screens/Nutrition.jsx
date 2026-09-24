import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';

export default function Nutrition() {
  const { state, t, go, loadMeals, createMeal, removeMeal } = useApp();
  const [mealName, setMealName] = useState('');
  const [mealCalories, setMealCalories] = useState('');
  const [mealTime, setMealTime] = useState('');
  const meals = state.mealsList || [];
  const total = meals.reduce((s, m) => s + (m.calories || 0), 0);

  useEffect(() => {
    loadMeals();
  }, [loadMeals]);

  const handleAdd = async () => {
    const name = mealName.trim() || 'Блюдо';
    const calories = Number.parseInt(mealCalories, 10) || 0;
    const time =
      mealTime ||
      new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    await createMeal(name, calories, time);
    setMealName('');
    setMealCalories('');
    setMealTime('');
  };

  const handleRemove = async (mealId) => {
    await removeMeal(mealId);
  };

  return (
    <section className='nutrition'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Питание', 'Nutrition')}</div>
          <div className='panel-sub'>
            {t('Калории', 'Calories')}: {total} / {state.calorieGoal || 2200}
          </div>
        </div>
        <div className='panel-body'>
          {meals.length === 0 ? (
            <div className='muted'>{t('Нет блюд', 'No meals')}</div>
          ) : (
            meals.map((m, i) => (
              <div key={m.id || i} className='list-row'>
                <div>
                  <div className='list-title'>{m.name}</div>
                  <div className='muted'>
                    {m.calories} {t('ккал', 'kcal')} · {m.time || ''}
                  </div>
                </div>
                <button
                  className='secondary'
                  onClick={() => handleRemove(m.id)}
                >
                  {t('Удалить', 'Remove')}
                </button>
              </div>
            ))
          )}
          <div className='add-meal-form'>
            <div className='field'>
              <label>{t('Название', 'Name')}</label>
              <input
                value={mealName}
                onChange={(e) => setMealName(e.target.value)}
                placeholder={t('Например: Овсянка', 'e.g. Oatmeal')}
              />
            </div>
            <div className='field'>
              <label>{t('Калории', 'Calories')}</label>
              <input
                type='number'
                value={mealCalories}
                onChange={(e) => setMealCalories(e.target.value)}
                placeholder='0'
              />
            </div>
            <div className='field'>
              <label>{t('Время', 'Time')}</label>
              <input
                type='time'
                value={mealTime}
                onChange={(e) => setMealTime(e.target.value)}
              />
            </div>
            <button className='primary full' onClick={handleAdd}>
              {t('Добавить блюдо', 'Add meal')}
            </button>
          </div>
          <button className='secondary full' onClick={() => go('home')}>
            {t('На главную', 'Go home')}
          </button>
        </div>
      </div>
    </section>
  );
}
