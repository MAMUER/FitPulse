import { useEffect, useState } from 'react';
import EmptyState from '../components/EmptyState';
import ListItem from '../components/ListItem';
import Panel from '../components/Panel';
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
      <Panel
        title={t('Тренировки', 'Workouts')}
        right={
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
        }
      >
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
                <EmptyState text={t('Нет планов', 'No plans')} />
              ) : (
                <div className='plans-list'>
                  {plans.map((plan) => (
                    <ListItem
                      key={plan.plan_id}
                      title={plan.plan_data?.name || plan.plan_id}
                      subtitle={
                        plan.training_goal ||
                        plan.plan_data?.training_goal ||
                        ''
                      }
                      right={
                        <button
                          type='button'
                          className='secondary'
                          onClick={() => getPlanDetails(plan.plan_id)}
                        >
                          {t('Подробнее', 'Details')}
                        </button>
                      }
                    />
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
      </Panel>
    </section>
  );
}
