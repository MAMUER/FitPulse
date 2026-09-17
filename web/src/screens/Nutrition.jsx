import { useApp } from '../contexts/AppContext';

export default function Nutrition() {
  const { state, t, addMeal, removeMeal } = useApp();
  const meals = state.meals || [];
  const total = meals.reduce((s, m) => s + (m.calories || 0), 0);

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
          {meals.map((m, i) => (
            <div key={i} className='list-row'>
              <div>
                <div className='list-title'>{m.name}</div>
                <div className='muted'>
                  {m.calories} {t('ккал', 'kcal')} · {m.time || ''}
                </div>
              </div>
              <button className='secondary' onClick={() => removeMeal(i)}>
                {t('Удалить', 'Remove')}
              </button>
            </div>
          ))}
          <button className='secondary full' onClick={() => addMeal('Завтрак')}>
            {t('Добавить завтрак', 'Add breakfast')}
          </button>
          <button className='secondary full' onClick={() => addMeal('Обед')}>
            {t('Добавить обед', 'Add lunch')}
          </button>
          <button className='secondary full' onClick={() => addMeal('Ужин')}>
            {t('Добавить ужин', 'Add dinner')}
          </button>
        </div>
      </div>
    </section>
  );
}
