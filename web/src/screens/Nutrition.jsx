import { useEffect, useState } from 'react';
import Modal from '../components/Modal';
import WaterTracker from '../components/WaterTracker';
import { useApp } from '../contexts/AppContext';

const SLOTS = ['breakfast', 'lunch', 'snack', 'dinner'];

const FALLBACK_MEAL = {
  label: '',
  icon: 'fa-utensils',
  time: '',
  title: '',
  kcal: 0,
  macros: '',
};

export default function Nutrition() {
  const {
    state,
    t,
    go,
    loadMeals,
    openMeal,
    closeMeal,
    selectMeal,
    saveCustomMeal,
    setNutritionPeriod,
    nutritionSettings,
    quickAI,
    createMeal,
    removeMeal,
  } = useApp();

  const [query, setQuery] = useState('');
  const [mealName, setMealName] = useState('');
  const [mealCalories, setMealCalories] = useState('');
  const [mealTime, setMealTime] = useState('');

  useEffect(() => {
    loadMeals();
  }, [loadMeals]);

  const now = new Date();
  const locale = state.language === 'en' ? 'en-US' : 'ru-RU';
  const meals = state.meals || {};
  const alternatives = state.alternatives || {};
  const period = state.nutritionPeriod || 'day';
  const data = state.nutritionData?.[period] || {
    calories: 0,
    goal: 0,
    protein: 0,
    fats: 0,
    carbs: 0,
  };
  const percentage = data.goal
    ? Math.min(100, Math.round((data.calories / data.goal) * 100))
    : 0;
  const diary = state.mealsList || [];
  const selectedMealSlot = state.selectedMeal;
  const selectedMealData = selectedMealSlot
    ? { ...FALLBACK_MEAL, ...(meals[selectedMealSlot] || {}) }
    : null;
  const mealOptions = selectedMealSlot
    ? [
        [
          selectedMealData.title,
          selectedMealData.kcal,
          selectedMealData.macros,
          selectedMealData.image,
        ],
        ...(alternatives[selectedMealSlot] || []),
      ]
    : [];

  const mealFor = (slot) => ({ ...FALLBACK_MEAL, ...(meals[slot] || {}) });

  const matchesQuery = (title) =>
    !query.trim() ||
    String(title || '')
      .toLowerCase()
      .includes(query.trim().toLowerCase());

  const handleAddToDiary = async () => {
    const name = mealName.trim() || t('Блюдо', 'Dish');
    const calories = Number.parseInt(mealCalories, 10) || 0;
    const time =
      mealTime ||
      new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    await createMeal(name, calories, time);
    setMealName('');
    setMealCalories('');
    setMealTime('');
  };

  const askAIMenu = () => {
    go('ai');
    quickAI(t('Составь меню на сегодня', 'Make a menu for today'));
  };

  return (
    <section className='nutrition'>
      <div className='page-title'>
        <div className='eyebrow'>SMART NUTRITION</div>
        <h2>{t('Питание', 'Nutrition')}</h2>
        <p>
          {t(
            'Сформируй меню на день, неделю или месяц.',
            'Plan your menu for a day, week or month.'
          )}
        </p>
      </div>

      <section className='date-card'>
        <div>
          <div className='eyebrow'>{t('СЕГОДНЯ', 'TODAY')}</div>
          <strong>{now.toLocaleDateString(locale, { weekday: 'long' })}</strong>
          <p>{now.toLocaleDateString(locale, { day: 'numeric', month: 'long' })}</p>
        </div>
        <div className='date-number'>{now.getDate()}</div>
      </section>

      <WaterTracker />

      <div className='chips'>
        <button type='button' className='chip active'>
          {t('Цель', 'Goal')}: {t('форма', 'form')}
        </button>
        <button type='button' className='chip' onClick={nutritionSettings}>
          {t('Персональные настройки', 'Personal settings')}
        </button>
        <button type='button' className='chip'>
          {data.goal || 2350} {t('ккал', 'kcal')}
        </button>
      </div>

      <section className='fp-nutrition-panel'>
        <h3>{t('План питания', 'Nutrition plan')}</h3>
        <div className='fp-period-tabs'>
          {[
            ['day', t('День', 'Day')],
            ['week', t('Неделя', 'Week')],
            ['month', t('Месяц', 'Month')],
          ].map(([key, label]) => (
            <button
              type='button'
              key={key}
              className={period === key ? 'active' : ''}
              onClick={() => setNutritionPeriod(key)}
            >
              {label}
            </button>
          ))}
        </div>
        <div className='fp-main-number'>
          <div>
            <strong>{Number(data.calories || 0).toLocaleString('ru-RU')}</strong>
            <small>{t('ккал в периоде', 'kcal in period')}</small>
          </div>
          <small>
            {t('цель', 'goal')}: {Number(data.goal || 0).toLocaleString('ru-RU')}{' '}
            {t('ккал', 'kcal')}
          </small>
        </div>
        <div className='fp-progress-line'>
          <i style={{ width: `${percentage}%` }} />
        </div>
        <div className='fp-stats'>
          <div className='fp-stat'>
            <strong>{data.protein} г</strong>
            <small>{t('белок', 'protein')}</small>
          </div>
          <div className='fp-stat'>
            <strong>{data.fats} г</strong>
            <small>{t('жиры', 'fats')}</small>
          </div>
          <div className='fp-stat'>
            <strong>{data.carbs} г</strong>
            <small>{t('углеводы', 'carbs')}</small>
          </div>
        </div>
      </section>

      <div className='between section'>
        <h3>{t('Меню на сегодня', "Today's menu")}</h3>
        <button type='button' className='link' onClick={askAIMenu}>
          {t('Спросить AI', 'Ask AI')}
        </button>
      </div>
      <section className='meal-plan'>
        {SLOTS.map((slot) => {
          const meal = mealFor(slot);
          const optional = slot === 'snack';
          return (
            <article
              className={`meal ${optional ? 'optional' : ''}`}
              key={slot}
            >
              <div className='meal-icon'>
                <i className={`fas ${meal.icon}`} />
              </div>
              <div className='meal-info'>
                <strong>
                  {meal.label}
                  {optional ? ` · ${t('опционально', 'optional')}` : ''}
                </strong>
                <small>
                  {meal.title} · {meal.kcal} {t('ккал', 'kcal')} · {meal.macros}
                </small>
              </div>
              <span className='meal-time'>{meal.time}</span>
              <button
                type='button'
                className='change'
                onClick={() => openMeal(slot)}
              >
                {t('Сменить', 'Change')}
              </button>
            </article>
          );
        })}
      </section>

      <div className='section'>
        <div className='between'>
          <h3>{t('Предложения AI', 'AI suggestions')}</h3>
          <span style={{ color: 'var(--blue)', fontSize: 9 }}>
            {t('выбери блюдо', 'choose a dish')}
          </span>
        </div>
      </div>
      <div className='field' style={{ marginTop: 10 }}>
        <label htmlFor='mealSearch'>{t('Поиск блюда', 'Search dishes')}</label>
        <input
          id='mealSearch'
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t('Например: омлет', 'E.g. omelet')}
        />
      </div>
      <div className='ai-suggestions'>
        {['breakfast', 'lunch', 'dinner'].map((slot) => {
          const suggestions = (alternatives[slot] || []).filter((item) =>
            matchesQuery(item[0])
          );
          if (!suggestions.length) return null;
          return (
            <div className='card' style={{ padding: 11 }} key={slot}>
              <div className='eyebrow'>{mealFor(slot).label}</div>
              {suggestions.map((item) => (
                <article className='ai-suggestion' key={item[0]}>
                  <img src={item[3]} alt={item[0]} />
                  <div className='ai-suggestion-info'>
                    <strong>{item[0]}</strong>
                    <small>
                      {item[1]} {t('ккал', 'kcal')} · {item[2]}
                    </small>
                  </div>
                  <button
                    type='button'
                    className='choose'
                    onClick={() =>
                      selectMeal(slot, item[0], item[1], item[2], item[3])
                    }
                  >
                    {t('Выбрать', 'Choose')}
                  </button>
                </article>
              ))}
            </div>
          );
        })}
      </div>

      <div className='warning'>
        <i className='fas fa-sparkles' />
        <span>
          {t(
            'AI учитывает калории, белок, цель питания и выбранные блюда.',
            'AI considers calories, protein, nutrition goals and selected dishes.'
          )}
        </span>
      </div>

      <section className='card form-card'>
        <h3>{t('Добавить своё блюдо', 'Add your own meal')}</h3>
        <label htmlFor='customName'>{t('Название', 'Name')}</label>
        <input
          id='customName'
          placeholder={t('Например, домашний омлет', 'E.g. homemade omelet')}
        />
        <label htmlFor='customKcal'>{t('Калории', 'Calories')}</label>
        <input id='customKcal' type='number' placeholder='420' />
        <label htmlFor='customMacros'>{t('КБЖУ', 'Macros')}</label>
        <input
          id='customMacros'
          placeholder={t('Б 30 г · Ж 18 г · У 20 г', 'P 30g · F 18g · C 20g')}
        />
        <label htmlFor='customSlot'>{t('Приём пищи', 'Meal type')}</label>
        <select id='customSlot' defaultValue='breakfast'>
          <option value='breakfast'>{t('Завтрак', 'Breakfast')}</option>
          <option value='lunch'>{t('Обед', 'Lunch')}</option>
          <option value='dinner'>{t('Ужин', 'Dinner')}</option>
          <option value='snack'>{t('Полдник', 'Snack')}</option>
        </select>
        <button
          type='button'
          className='primary full'
          style={{ marginTop: 13 }}
          onClick={saveCustomMeal}
        >
          {t('Сохранить в меню и календарь', 'Save to menu and calendar')}
        </button>
      </section>

      <div className='section'>
        <h3>{t('Дневник питания', 'Nutrition diary')}</h3>
      </div>
      <section className='card'>
        {diary.length === 0 ? (
          <p className='muted' style={{ fontSize: 10 }}>
            {t('Записей пока нет', 'No entries yet')}
          </p>
        ) : (
          diary.map((entry, index) => (
            <div className='body-row' key={entry.id || index}>
              <span>
                <b>{entry.name}</b>
                {` · ${entry.calories || 0} ${t('ккал', 'kcal')}`}
                {entry.time ? ` · ${entry.time}` : ''}
              </span>
              <button
                type='button'
                className='icon'
                aria-label={t('Удалить', 'Remove')}
                onClick={() => removeMeal(entry.id)}
              >
                <i className='fas fa-trash' />
              </button>
            </div>
          ))
        )}
        <div className='form-row' style={{ marginTop: 10 }}>
          <input
            aria-label={t('Название', 'Name')}
            value={mealName}
            onChange={(event) => setMealName(event.target.value)}
            placeholder={t('Например: Овсянка', 'e.g. Oatmeal')}
          />
          <input
            aria-label={t('Калории', 'Calories')}
            type='number'
            value={mealCalories}
            onChange={(event) => setMealCalories(event.target.value)}
            placeholder='0'
          />
          <input
            aria-label={t('Время', 'Time')}
            type='time'
            value={mealTime}
            onChange={(event) => setMealTime(event.target.value)}
          />
        </div>
        <button
          type='button'
          className='secondary full'
          style={{ marginTop: 8 }}
          onClick={handleAddToDiary}
        >
          {t('Добавить в дневник', 'Add to diary')}
        </button>
      </section>

      {selectedMealSlot && selectedMealData && (
        <Modal
          className='fp-profile-modal'
          label={t('Выбери блюдо', 'Choose a dish')}
          onClose={closeMeal}
        >
          <section className='fp-profile-box'>
            <div className='between'>
              <div>
                <div className='eyebrow'>
                  {String(selectedMealData.label).toUpperCase()}
                </div>
                <h3>{t('Выбери блюдо', 'Choose a dish')}</h3>
              </div>
              <button
                type='button'
                className='icon'
                aria-label={t('Закрыть', 'Close')}
                onClick={closeMeal}
              >
                ×
              </button>
            </div>
            <div className='warning' style={{ margin: '0 0 12px' }}>
              <i className='fas fa-sparkles' />
              <span>
                {t(
                  'AI подобрал варианты по калориям, белку и твоей цели.',
                  'AI selected options based on calories, protein and your goal.'
                )}
              </span>
            </div>
            {mealOptions.map((item, index) => (
              <article className='ai-suggestion' key={item[0] || index}>
                <img src={item[3]} alt={item[0]} />
                <div className='ai-suggestion-info'>
                  <strong>{item[0]}</strong>
                  <small>
                    {item[1]} {t('ккал', 'kcal')} · {item[2]}
                  </small>
                </div>
                <button
                  type='button'
                  className='choose'
                  onClick={() =>
                    selectMeal(
                      selectedMealSlot,
                      item[0],
                      item[1],
                      item[2],
                      item[3]
                    )
                  }
                >
                  {index === 0 ? t('Сейчас', 'Current') : t('Выбрать', 'Choose')}
                </button>
              </article>
            ))}
          </section>
        </Modal>
      )}
    </section>
  );
}
