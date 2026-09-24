import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';

export default function Training() {
  const {
    state,
    t,
    go,
    startWorkout,
    loadTrainingPlans,
    generatePlan,
    getPlanDetails,
  } = useApp();
  const [tab, setTab] = useState('plans');
  const [generating, setGenerating] = useState(false);
  const plans = state.trainingPlans || [];

  useEffect(() => {
    loadTrainingPlans();
  }, [loadTrainingPlans]);

  const handleGenerate = async () => {
    setGenerating(true);
    await generatePlan({
      durationWeeks: 4,
      availableDays: [1, 3, 5],
      class: 'endurance_basic',
      confidence: 0.8,
    });
    setGenerating(false);
    setTab('plans');
  };

  return (
    <section className='training'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Тренировки', 'Workouts')}</div>
          <div className='tabs'>
            <button
              type='button'
              className={`tab-btn ${tab === 'plans' ? 'active' : ''}`}
              onClick={() => setTab('plans')}
            >
              {t('Планы', 'Plans')}
            </button>
            <button
              type='button'
              className={`tab-btn ${tab === 'start' ? 'active' : ''}`}
              onClick={() => setTab('start')}
            >
              {t('Таймер', 'Timer')}
            </button>
          </div>
        </div>
        <div className='panel-body'>
          {tab === 'plans' && (
            <>
              <button
                type='button'
                className='primary full'
                onClick={handleGenerate}
                disabled={generating}
              >
                {generating
                  ? t('Генерация...', 'Generating...')
                  : t('Создать план', 'Generate plan')}
              </button>
              {plans.length === 0 ? (
                <div className='muted'>{t('Нет планов', 'No plans')}</div>
              ) : (
                <div className='plans-list'>
                  {plans.map((plan) => (
                    <div key={plan.plan_id} className='list-row plan-row'>
                      <div>
                        <div className='list-title'>
                          {plan.plan_data?.name || plan.plan_id}
                        </div>
                        <div className='muted'>
                          {plan.training_goal ||
                            plan.plan_data?.training_goal ||
                            ''}{' '}
                          · {plan.duration_weeks || 4} {t('недель', 'weeks')}
                        </div>
                      </div>
                      <button
                        type='button'
                        className='secondary'
                        onClick={() => getPlanDetails(plan.plan_id)}
                      >
                        {t('Подробнее', 'Details')}
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}
          {tab === 'start' && (
            <>
              <button
                type='button'
                className='primary full'
                onClick={() => startWorkout('Тренировка')}
              >
                {t('Начать тренировку', 'Start workout')}
              </button>
              <button
                type='button'
                className='secondary full'
                onClick={() => go('home')}
              >
                {t('На главную', 'Go home')}
              </button>
            </>
          )}
        </div>
      </div>
    </section>
  );
}
