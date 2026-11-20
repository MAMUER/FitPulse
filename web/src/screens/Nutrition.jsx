import { useEffect, useState } from 'react';
import Modal from '../components/Modal';
import WaterTracker from '../components/WaterTracker';
import { useApp } from '../contexts/AppContext';

const SLOTS = ['breakfast', 'lunch', 'snack', 'dinner'];

const SLOT_LABELS = {
  breakfast: ['Завтрак', 'Breakfast'],
  lunch: ['Обед', 'Lunch'],
  snack: ['Полдник', 'Snack'],
  dinner: ['Ужин', 'Dinner'],
};

const FALLBACK_MEAL = {
  label: '',
  icon: 'fa-utensils',
  time: '',
  title: '',
  kcal: 0,
  macros: '',
  image: '',
};

const EMPTY_GOALS_FORM = { calories: '', protein: '', fat: '', carbs: '' };

const EMPTY_DISH_FORM = {
  name: '',
  kcal: '',
  protein: '',
  fat: '',
  carbs: '',
  slot: 'breakfast',
  image: '',
};

function macrosLine(dish) {
  const p = dish.protein_g ?? 0;
  const f = dish.fat_g ?? 0;
  const c = dish.carbs_g ?? 0;
  return `Б ${p} г · Ж ${f} г · У ${c} г`;
}

export default function Nutrition() {
  const {
    state,
    t,
    go,
    notify,
    canEdit,
    loadMeals,
    loadDishes,
    openMeal,
    closeMeal,
    selectMeal,
    setNutritionPeriod,
    quickAI,
    createMeal,
    removeMeal,
    addDishToCatalog,
    saveNutritionGoals,
  } = useApp();

  const [query, setQuery] = useState('');
  const [mealName, setMealName] = useState('');
  const [mealCalories, setMealCalories] = useState('');
  const [mealTime, setMealTime] = useState('');
  const [dishForm, setDishForm] = useState(EMPTY_DISH_FORM);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [goalsForm, setGoalsForm] = useState(EMPTY_GOALS_FORM);

  useEffect(() => {
    loadMeals();
    void loadDishes();
  }, [loadMeals, loadDishes]);

  const now = new Date();
  const locale = state.language === 'en' ? 'en-US' : 'ru-RU';
  const meals = state.meals || {};
  const dishes = state.dishes || [];
  const period = state.nutritionPeriod || 'day';
  const data = state.nutritionData?.[period] || {
    calories: 0,
    goal: 0,
    protein: 0,
    fats: 0,
    carbs: 0,
  };
  const diary = state.mealsList || [];
  const goals = state.nutritionGoals || {};
  const goalCalories =
    goals.calories || state.profile?.goalCalories || data.goal || 0;
  const goalLabel =
    (Array.isArray(state.survey?.goals) && state.survey.goals[0]) ||
    state.profile?.goal ||
    t('не задана', 'not set');

  const displayCalories = data.calories;
  const percentage = goalCalories
    ? Math.min(100, Math.round((displayCalories / goalCalories) * 100))
    : 0;

  const selectedMealSlot = state.selectedMeal;
  const selectedMealData = selectedMealSlot
    ? { ...FALLBACK_MEAL, ...meals[selectedMealSlot] }
    : null;

  const mealFor = (slot) => ({ ...FALLBACK_MEAL, ...meals[slot] });

  const matchesQuery = (title) =>
    !query.trim() ||
    String(title || '')
      .toLowerCase()
      .includes(query.trim().toLowerCase());

  const filteredDishes = dishes.filter((dish) => matchesQuery(dish.name || ''));

  const handleAddToDiary = async () => {
    const name = mealName.trim();
    if (!name) {
      notify(t('Введите название блюда', 'Enter a dish name'));
      return;
    }
    const calories = Number.parseInt(mealCalories, 10) || 0;
    const time =
      mealTime ||
      new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    const ok = await createMeal(name, calories, time);
    if (ok) {
      setMealName('');
      setMealCalories('');
      setMealTime('');
    }
  };

  const askAIMenu = () => {
    go('ai');
    quickAI(t('Составь меню на сегодня', 'Make a menu for today'));
  };

  const openSettings = () => {
    setGoalsForm({
      calories: goalCalories ? String(goalCalories) : '',
      protein: goals.protein ? String(goals.protein) : '',
      fat: goals.fats ? String(goals.fats) : '',
      carbs: goals.carbs ? String(goals.carbs) : '',
    });
    setSettingsOpen(true);
  };

  const submitSettings = () => {
    saveNutritionGoals({
      calories: Number(goalsForm.calories) || 0,
      protein: Number(goalsForm.protein) || 0,
      fats: Number(goalsForm.fat) || 0,
      carbs: Number(goalsForm.carbs) || 0,
    });
    setSettingsOpen(false);
  };

  const updateDishField = (field) => (event) => {
    const value = event.target.value;
    setDishForm((prev) => ({ ...prev, [field]: value }));
  };

  const onDishPhoto = (event) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (file.size > 300_000) {
      notify(
        t('Фото слишком большое (макс. 300 КБ)', 'Photo too large (max 300 KB)')
      );
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      // reader.result — string | ArrayBuffer | null; берём только строку (data URL).
      const result = reader.result;
      const image = typeof result === 'string' ? result : '';
      setDishForm((prev) => ({ ...prev, image }));
    };
    reader.readAsDataURL(file);
  };

  const submitDish = async () => {
    const name = dishForm.name.trim();
    if (!name) {
      notify(t('Введите название блюда', 'Enter a dish name'));
      return;
    }
    const ok = await addDishToCatalog({
      name,
      kcal: Number.parseInt(dishForm.kcal, 10) || 0,
      protein_g: Number.parseInt(dishForm.protein, 10) || 0,
      fat_g: Number.parseInt(dishForm.fat, 10) || 0,
      carbs_g: Number.parseInt(dishForm.carbs, 10) || 0,
      meal_slot: dishForm.slot || 'any',
      image_url: dishForm.image || '',
    });
    if (ok) setDishForm(EMPTY_DISH_FORM);
  };

  const chooseDish = (slot, dish) => {
    selectMeal(
      slot,
      dish.name,
      dish.kcal,
      macrosLine(dish),
      dish.image_url || ''
    );
  };

  const slotDishes = selectedMealSlot
    ? dishes.filter(
        (dish) =>
          matchesQuery(dish.name) &&
          (!dish.meal_slot ||
            dish.meal_slot === 'any' ||
            dish.meal_slot === selectedMealSlot)
      )
    : [];

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
          <p>
            {now.toLocaleDateString(locale, { day: 'numeric', month: 'long' })}
          </p>
        </div>
        <div className='date-number'>{now.getDate()}</div>
      </section>

      <WaterTracker />

      <div className='chips'>
        <button type='button' className='chip active' onClick={openSettings}>
          {t('Цель', 'Goal')}: {goalLabel}
        </button>
        <button type='button' className='chip' onClick={openSettings}>
          {t('Персональные настройки', 'Personal settings')}
        </button>
        <button type='button' className='chip' onClick={openSettings}>
          {goalCalories || '—'} {t('ккал', 'kcal')}
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
            <strong>
              {Number(displayCalories || 0).toLocaleString('ru-RU')}
            </strong>
            <small>{t('ккал в периоде', 'kcal in period')}</small>
          </div>
          <small>
            {t('цель', 'goal')}:{' '}
            {Number(goalCalories || 0).toLocaleString('ru-RU')}{' '}
            {t('ккал', 'kcal')}
          </small>
        </div>
        <div className='fp-progress-line'>
          <i style={{ width: `${percentage}%` }} />
        </div>
        <div className='fp-stats'>
          <div className='fp-stat'>
            <strong>
              {goals.protein || data.protein} {t('г', 'g')}
            </strong>
            <small>{t('белок', 'protein')}</small>
          </div>
          <div className='fp-stat'>
            <strong>
              {goals.fats || data.fats} {t('г', 'g')}
            </strong>
            <small>{t('жиры', 'fats')}</small>
          </div>
          <div className='fp-stat'>
            <strong>
              {goals.carbs || data.carbs} {t('г', 'g')}
            </strong>
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
                  {meal.title || t('не выбрано', 'not chosen')} · {meal.kcal}{' '}
                  {t('ккал', 'kcal')}
                  {meal.macros ? ` · ${meal.macros}` : ''}
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
        <h3>{t('Каталог блюд', 'Dish catalog')}</h3>
      </div>
      <div className='field' style={{ marginTop: 10 }}>
        <label htmlFor='mealSearch'>
          {t('Поиск блюда в каталоге', 'Search the dish catalog')}
        </label>
        <input
          id='mealSearch'
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t('Например: омлет', 'E.g. omelet')}
        />
      </div>
      <div className='ai-suggestions'>
        {filteredDishes.length === 0 ? (
          <p className='muted' style={{ fontSize: 11 }}>
            {query.trim()
              ? t('Ничего не найдено по запросу', 'Nothing matches your query')
              : t(
                  'Каталог пуст. Добавьте своё блюдо ниже.',
                  'The catalog is empty. Add your own dish below.'
                )}
          </p>
        ) : (
          filteredDishes.map((dish) => (
            <article className='ai-suggestion' key={dish.id}>
              {dish.image_url ? (
                <img src={dish.image_url} alt={dish.name} />
              ) : (
                <div className='ai-suggestion-info'>
                  <i className='fas fa-utensils' />
                </div>
              )}
              <div className='ai-suggestion-info'>
                <strong>{dish.name}</strong>
                <small>
                  {dish.kcal} {t('ккал', 'kcal')} · {macrosLine(dish)}
                </small>
              </div>
              <button
                type='button'
                className='choose'
                onClick={() =>
                  chooseDish(
                    dish.meal_slot && dish.meal_slot !== 'any'
                      ? dish.meal_slot
                      : 'breakfast',
                    dish
                  )
                }
              >
                {t('Выбрать', 'Choose')}
              </button>
            </article>
          ))
        )}
      </div>

      <section className='card form-card'>
        <h3>{t('Добавить своё блюдо', 'Add your own dish')}</h3>
        <p className='muted' style={{ fontSize: 10, marginTop: -4 }}>
          {t(
            'Блюдо сохранится в ваш личный каталог и будет видно только вам.',
            'The dish is saved to your personal catalog and is visible only to you.'
          )}
        </p>
        <label htmlFor='customName'>{t('Название', 'Name')}</label>
        <input
          id='customName'
          value={dishForm.name}
          onChange={updateDishField('name')}
          placeholder={t('Например, домашний омлет', 'E.g. homemade omelet')}
        />
        <label htmlFor='customKcal'>
          {t('Калории, ккал', 'Calories, kcal')}
        </label>
        <input
          id='customKcal'
          type='number'
          inputMode='numeric'
          value={dishForm.kcal}
          onChange={updateDishField('kcal')}
          placeholder='420'
        />
        <div className='form-row'>
          <div className='field'>
            <label htmlFor='customProtein'>{t('Белки, г', 'Protein, g')}</label>
            <input
              id='customProtein'
              type='number'
              inputMode='numeric'
              value={dishForm.protein}
              onChange={updateDishField('protein')}
              placeholder='30'
            />
          </div>
          <div className='field'>
            <label htmlFor='customFat'>{t('Жиры, г', 'Fat, g')}</label>
            <input
              id='customFat'
              type='number'
              inputMode='numeric'
              value={dishForm.fat}
              onChange={updateDishField('fat')}
              placeholder='18'
            />
          </div>
          <div className='field'>
            <label htmlFor='customCarbs'>{t('Углеводы, г', 'Carbs, g')}</label>
            <input
              id='customCarbs'
              type='number'
              inputMode='numeric'
              value={dishForm.carbs}
              onChange={updateDishField('carbs')}
              placeholder='20'
            />
          </div>
        </div>
        <label htmlFor='customSlot'>
          {t('Приём пищи (необязательно)', 'Meal type (optional)')}
        </label>
        <select
          id='customSlot'
          value={dishForm.slot}
          onChange={updateDishField('slot')}
        >
          <option value='any'>{t('Любой', 'Any')}</option>
          <option value='breakfast'>{t('Завтрак', 'Breakfast')}</option>
          <option value='lunch'>{t('Обед', 'Lunch')}</option>
          <option value='dinner'>{t('Ужин', 'Dinner')}</option>
          <option value='snack'>{t('Полдник', 'Snack')}</option>
        </select>
        <label className='upload-btn' style={{ marginTop: 10 }}>
          <i className='fas fa-camera' />{' '}
          {dishForm.image
            ? t('Фото добавлено — заменить', 'Photo added — replace')
            : t('Добавить фото (необязательно)', 'Add photo (optional)')}
          <input type='file' accept='image/*' onChange={onDishPhoto} />
        </label>
        {dishForm.image ? (
          <img
            src={dishForm.image}
            alt=''
            style={{
              width: 64,
              height: 64,
              objectFit: 'cover',
              borderRadius: 8,
              marginTop: 8,
            }}
          />
        ) : null}
        {canEdit && (
          <button
            type='button'
            className='primary full'
            style={{ marginTop: 13 }}
            onClick={submitDish}
          >
            {t('Добавить в каталог', 'Add to catalog')}
          </button>
        )}
      </section>

      <div className='section'>
        <h3>{t('Дневник питания', 'Nutrition diary')}</h3>
        <p className='muted' style={{ fontSize: 10, marginTop: -4 }}>
          {t(
            'Сюда попадают блюда, которые вы съели сегодня.',
            'Meals you have eaten today appear here.'
          )}
        </p>
      </div>
      <section className='card'>
        {diary.length === 0 ? (
          <p className='muted' style={{ fontSize: 11 }}>
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
            inputMode='numeric'
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
        {canEdit && (
          <button
            type='button'
            className='secondary full'
            style={{ marginTop: 8 }}
            onClick={handleAddToDiary}
          >
            {t('Добавить в дневник', 'Add to diary')}
          </button>
        )}
      </section>

      {settingsOpen && (
        <Modal
          className='fp-profile-modal'
          label={t('Настройки питания', 'Nutrition settings')}
          onClose={() => setSettingsOpen(false)}
        >
          <section className='fp-profile-box'>
            <div className='between'>
              <div>
                <div className='eyebrow'>{t('ЦЕЛИ', 'GOALS')}</div>
                <h3>{t('Настройки питания', 'Nutrition settings')}</h3>
              </div>
              <button
                type='button'
                className='icon'
                aria-label={t('Закрыть', 'Close')}
                onClick={() => setSettingsOpen(false)}
              >
                ×
              </button>
            </div>
            <label htmlFor='goalCalories'>
              {t('Калории в день, ккал', 'Daily calories, kcal')}
            </label>
            <input
              id='goalCalories'
              type='number'
              inputMode='numeric'
              value={goalsForm.calories}
              onChange={(event) =>
                setGoalsForm((prev) => ({
                  ...prev,
                  calories: event.target.value,
                }))
              }
              placeholder='2350'
            />
            <div className='form-row'>
              <div className='field'>
                <label htmlFor='goalProtein'>
                  {t('Белки, г', 'Protein, g')}
                </label>
                <input
                  id='goalProtein'
                  type='number'
                  inputMode='numeric'
                  value={goalsForm.protein}
                  onChange={(event) =>
                    setGoalsForm((prev) => ({
                      ...prev,
                      protein: event.target.value,
                    }))
                  }
                  placeholder='130'
                />
              </div>
              <div className='field'>
                <label htmlFor='goalFat'>{t('Жиры, г', 'Fat, g')}</label>
                <input
                  id='goalFat'
                  type='number'
                  inputMode='numeric'
                  value={goalsForm.fat}
                  onChange={(event) =>
                    setGoalsForm((prev) => ({
                      ...prev,
                      fat: event.target.value,
                    }))
                  }
                  placeholder='70'
                />
              </div>
              <div className='field'>
                <label htmlFor='goalCarbs'>
                  {t('Углеводы, г', 'Carbs, g')}
                </label>
                <input
                  id='goalCarbs'
                  type='number'
                  inputMode='numeric'
                  value={goalsForm.carbs}
                  onChange={(event) =>
                    setGoalsForm((prev) => ({
                      ...prev,
                      carbs: event.target.value,
                    }))
                  }
                  placeholder='250'
                />
              </div>
            </div>
            <button
              type='button'
              className='primary full'
              style={{ marginTop: 13 }}
              onClick={submitSettings}
            >
              {t('Сохранить', 'Save')}
            </button>
          </section>
        </Modal>
      )}

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
                  {(SLOT_LABELS[selectedMealSlot] || [
                    selectedMealSlot,
                    selectedMealSlot,
                  ])[state.language === 'en' ? 1 : 0].toUpperCase()}
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
            <p className='muted' style={{ fontSize: 10, marginTop: -6 }}>
              {t(
                'Выберите блюдо из вашего каталога.',
                'Pick a dish from your catalog.'
              )}
            </p>
            {slotDishes.length === 0 ? (
              <p className='muted' style={{ fontSize: 11 }}>
                {t(
                  'Нет блюд для этого приёма пищи. Добавьте блюдо в каталог.',
                  'No dishes for this meal. Add a dish to the catalog.'
                )}
              </p>
            ) : (
              slotDishes.map((dish) => (
                <article className='ai-suggestion' key={dish.id}>
                  {dish.image_url ? (
                    <img src={dish.image_url} alt={dish.name} />
                  ) : (
                    <div className='ai-suggestion-info'>
                      <i className='fas fa-utensils' />
                    </div>
                  )}
                  <div className='ai-suggestion-info'>
                    <strong>{dish.name}</strong>
                    <small>
                      {dish.kcal} {t('ккал', 'kcal')} · {macrosLine(dish)}
                    </small>
                  </div>
                  <button
                    type='button'
                    className='choose'
                    onClick={() => chooseDish(selectedMealSlot, dish)}
                  >
                    {t('Выбрать', 'Choose')}
                  </button>
                </article>
              ))
            )}
          </section>
        </Modal>
      )}
    </section>
  );
}
