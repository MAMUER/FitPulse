import { useEffect } from 'react';
import { useApp } from '../contexts/AppContext';

const QUESTIONS = [
  {
    title: ['Аллергии', 'Allergies'],
    key: 'allergies',
    multi: true,
    options: [
      ['Орехи', 'Nuts'],
      ['Молоко', 'Milk'],
      ['Яйца', 'Eggs'],
      ['Глютен', 'Gluten'],
      ['Морепродукты', 'Seafood'],
      ['Соя', 'Soy'],
      ['Нет аллергий', 'No allergies'],
    ],
  },
  {
    title: ['Ограничения по здоровью', 'Health restrictions'],
    key: 'restrictions',
    multi: true,
    options: [
      ['Диабет', 'Diabetes'],
      ['Гипертония', 'Hypertension'],
      ['Сердечные', 'Heart conditions'],
      ['Почечные', 'Kidney conditions'],
      ['Печёночные', 'Liver conditions'],
      ['Ортопедические', 'Orthopedic conditions'],
      ['Нет ограничений', 'No restrictions'],
    ],
  },
  {
    title: ['Цели занятий', 'Fitness goals'],
    key: 'goals',
    multi: false,
    options: [
      ['Похудеть', 'Lose weight'],
      ['Набрать мышечную массу', 'Build muscle'],
      ['Поддерживать форму', 'Stay fit'],
      ['Повысить выносливость', 'Improve endurance'],
      ['Улучшить здоровье', 'Improve health'],
      ['Снизить стресс', 'Reduce stress'],
    ],
  },
  {
    title: ['Предпочтения в еде', 'Diet preferences'],
    key: 'diet',
    options: [
      ['Обычное', 'Regular'],
      ['Вегетарианское', 'Vegetarian'],
      ['Веганское', 'Vegan'],
      ['Кето', 'Keto'],
      ['Безглютеновое', 'Gluten-free'],
      ['Средиземноморское', 'Mediterranean'],
    ],
  },
  {
    title: ['Уровень активности', 'Activity level'],
    key: 'activity',
    options: [
      ['Сидячий', 'Sedentary'],
      ['Лёгкий (1--2 раза/нед)', 'Light (1--2 times/week)'],
      ['Средний (3--4 раза/нед)', 'Moderate (3--4 times/week)'],
      ['Высокий (5--6 раз/нед)', 'High (5--6 times/week)'],
      ['Профессиональный спорт', 'Professional sports'],
    ],
  },
  {
    title: ['Сон (часов)', 'Sleep (hours)'],
    key: 'sleepHours',
    options: [
      ['Менее 6 ч', 'Under 6 hours'],
      ['6--7 ч', '6--7 hours'],
      ['7--8 ч', '7--8 hours'],
      ['8--9 ч', '8--9 hours'],
      ['Более 9 ч', 'Over 9 hours'],
    ],
  },
  {
    title: ['Питьевой режим', 'Water intake'],
    key: 'waterIntake',
    options: [
      ['Менее 1 л', 'Under 1 L'],
      ['1--1.5 л', '1--1.5 L'],
      ['1.5--2 л', '1.5--2 L'],
      ['Более 2 л', 'Over 2 L'],
    ],
  },
  {
    title: ['Предпочтения по тренировкам', 'Preferred workouts'],
    key: 'preferredWorkouts',
    multi: true,
    options: [
      ['Силовые', 'Strength training'],
      ['Кардио', 'Cardio'],
      ['Йога', 'Yoga'],
      ['Пилатес', 'Pilates'],
      ['Плавание', 'Swimming'],
      ['Бег', 'Running'],
      ['Функциональные', 'Functional training'],
    ],
  },
  {
    title: ['Дни тренировок в неделю', 'Training days per week'],
    key: 'trainingDays',
    options: [
      ['1--2 дня', '1--2 days'],
      ['3 дня', '3 days'],
      ['4 дня', '4 days'],
      ['5 дней', '5 days'],
      ['6--7 дней', '6--7 days'],
    ],
  },
];

// Progress dot state: done before the current step, active on it, empty after.
function dotState(index, step) {
  if (index < step) return 'done';
  if (index === step) return 'active';
  return '';
}

export default function Survey() {
  const { state, t, go, surveySelect, surveyNext, surveyPrev, deferSurvey } =
    useApp();
  const step = state.survey?.step || 0;
  const question = QUESTIONS[step];

  useEffect(() => {
    if (!question) go('home');
  }, [question, go]);

  if (!question) {
    return null;
  }

  const selected = state.survey?.[question.key] || (question.multi ? [] : '');
  const handleNext = async () => {
    const complete = Array.isArray(selected)
      ? selected.length > 0
      : Boolean(selected);
    if (!complete) {
      await surveyNext();
      return;
    }
    await surveyNext();
    if (step === QUESTIONS.length - 1) go('home');
  };

  const handleDefer = () => {
    deferSurvey();
    go('home');
  };

  return (
    <section className='auth survey'>
      <div className='auth-brand'>
        <h1>{t('Персонализированный опрос', 'Personalized survey')}</h1>
        <p>
          {t(
            'Расскажи о себе, чтобы я мог подобрать идеальную программу',
            'Tell me about yourself so I can choose the perfect program'
          )}
        </p>
        <p className='survey-step'>
          {step + 1} / {QUESTIONS.length}
        </p>
      </div>
      <div className='survey-progress'>
        {QUESTIONS.map((item, index) => (
          <div key={item.key} className={`dot ${dotState(index, step)}`} />
        ))}
      </div>
      <h3>{t(...question.title)}</h3>
      <p className='muted'>
        {question.multi
          ? t('Выберите все подходящие варианты', 'Select all that apply')
          : t('Выберите один вариант', 'Select one option')}
      </p>
      <div className='survey-options'>
        {question.options.map(([ru, en]) => {
          const value = question.multi
            ? selected.includes(ru)
            : selected === ru;
          return (
            <label key={ru}>
              <input
                type={question.multi ? 'checkbox' : 'radio'}
                name={`survey_${question.key}`}
                value={ru}
                checked={value}
                onChange={() => surveySelect(question.key, ru, question.multi)}
              />
              {t(ru, en)}
            </label>
          );
        })}
      </div>
      <div className='form-row' style={{ marginTop: 14 }}>
        {step > 0 && (
          <button type='button' className='secondary' onClick={surveyPrev}>
            {t('Назад', 'Back')}
          </button>
        )}
        <button type='button' className='primary full' onClick={handleNext}>
          {step < QUESTIONS.length - 1
            ? t('Далее', 'Next')
            : t('Завершить', 'Complete')}
        </button>
        <button type='button' className='secondary full' onClick={handleDefer}>
          {t('Заполнить позже', 'Complete later')}
        </button>
      </div>
    </section>
  );
}
