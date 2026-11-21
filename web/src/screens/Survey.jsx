import { useEffect } from 'react';
import { useApp } from '../contexts/AppContext';
import { QUESTIONS } from '../data/surveyQuestions';

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
