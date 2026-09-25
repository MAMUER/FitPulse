import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';
import Panel from '../components/Panel';
import ListItem from '../components/ListItem';
import EmptyState from '../components/EmptyState';

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
      <Panel
        title={t('Питание', 'Nutrition')}
        subtitle={`${t('Калории', 'Calories')}: ${total} / ${state.calorieGoal || 2200}`}
      >
        <div className='panel-body'>
          {meals.length === 0 ? (
            <EmptyState text={t('Нет блюд', 'No meals')} />
          ) : (
            meals.map((m, i) => (
              <ListItem
                key={m.id || i}
                title={m.name}
                subtitle={`${m.calories} ${t('ккал', 'kcal')} · ${m.time || ''}`}
                right={
                  <button
                    type='button'
                    className='secondary'
                    onClick={() => handleRemove(m.id)}
                  >
                    {t('Удалить', 'Remove')}
                  </button>
                }
              />
            ))
          )}
          <div className='add-meal-form'>
            <div className='field'>
              <label htmlFor='mealName'>{t('Название', 'Name')}</label>
              <input
                id='mealName'
                value={mealName}
                onChange={(e) => setMealName(e.target.value)}
                placeholder={t('Например: Овсянка', 'e.g. Oatmeal')}
              />
            </div>
            <div className='field'>
              <label htmlFor='mealCalories'>{t('Калории', 'Calories')}</label>
              <input
                id='mealCalories'
                type='number'
                value={mealCalories}
                onChange={(e) => setMealCalories(e.target.value)}
                placeholder='0'
              />
            </div>
            <div className='field'>
              <label htmlFor='mealTime'>{t('Время', 'Time')}</label>
              <input
                id='mealTime'
                type='time'
                value={mealTime}
                onChange={(e) => setMealTime(e.target.value)}
              />
            </div>
            <button type='button' className='primary full' onClick={handleAdd}>
              {t('Добавить блюдо', 'Add meal')}
            </button>
          </div>
          <button
            type='button'
            className='secondary full'
            onClick={() => go('home')}
          >
            {t('На главную', 'Go home')}
          </button>
        </div>
      </Panel>
    </section>
  );
}
