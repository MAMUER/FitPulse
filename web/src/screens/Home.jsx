import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import Panel from '../components/Panel';
import { useApp } from '../contexts/AppContext';

export default function Home() {
  const {
    state,
    t,
    drinkWater,
    startWorkout,
    go,
    loadBiometrics,
    loadTrainingPlans,
    loadAchievements,
    loadProgress,
  } = useApp();
  const navigate = useNavigate();

  useEffect(() => {
    loadBiometrics();
    loadTrainingPlans();
    loadAchievements();
    loadProgress();
  }, [loadBiometrics, loadTrainingPlans, loadAchievements, loadProgress]);

  const biometrics = state.biometrics || [];
  const latestHR = biometrics.find((b) => b.metric_type === 'heart_rate');
  const latestSpO2 = biometrics.find((b) => b.metric_type === 'spo2');
  const latestTemp = biometrics.find((b) => b.metric_type === 'temperature');
  const bmi =
    state.height > 0 && state.weight > 0
      ? +(state.weight / (state.height / 100) ** 2).toFixed(1)
      : null;

  const bmiText = bmi ? `ИМТ ${bmi}` : 'ИМТ —';
  const weightSubtitle = state.weight ? `${state.weight} кг · ${bmiText}` : '—';

  return (
    <section className='home'>
      <Panel
        title={t('Прогресс', 'Progress')}
        subtitle={weightSubtitle}
        right={
          <button type='button' className='link' onClick={() => go('body')}>
            {t('Профиль тела', 'Body profile')}
          </button>
        }
      >
        <div className='panel-body'>
          <div className='metric'>
            <div className='metric-val'>
              {state.waterCount || 0}
              <span className='muted'> / {state.waterGoal || 2000}</span>
            </div>
            <div className='metric-label'>{t('Вода, мл', 'Water, ml')}</div>
            <button type='button' className='primary full' onClick={drinkWater}>
              +250 {t('мл', 'ml')}
            </button>
          </div>
          {latestHR && (
            <div className='metric'>
              <div className='metric-val'>{latestHR.value}</div>
              <div className='metric-label'>{t('Пульс', 'Heart rate')}</div>
            </div>
          )}
          {latestSpO2 && (
            <div className='metric'>
              <div className='metric-val'>{latestSpO2.value}%</div>
              <div className='metric-label'>{t('SpO2', 'SpO2')}</div>
            </div>
          )}
          {latestTemp && (
            <div className='metric'>
              <div className='metric-val'>{latestTemp.value}°</div>
              <div className='metric-label'>
                {t('Температура', 'Temperature')}
              </div>
            </div>
          )}
        </div>
      </Panel>
      <Panel title={t('Тренировки', 'Workouts')}>
        <div className='panel-body'>
          <button type='button' className='primary full' onClick={startWorkout}>
            {t('Начать тренировку', 'Start workout')}
          </button>
          <button
            type='button'
            className='secondary full'
            onClick={() => go('training')}
          >
            {t('Планы тренировок', 'Training plans')}
          </button>
        </div>
      </Panel>
      <Panel
        title={t('Рекомендации', 'Recommendations')}
        subtitle={t('На основе ваших данных', 'Based on your data')}
      >
        <div className='panel-body'>
          <button
            type='button'
            className='secondary full'
            onClick={() => go('videos')}
          >
            {t('Видео-тренировки', 'Video workouts')}
          </button>
          <button
            type='button'
            className='secondary full'
            onClick={() => go('ai')}
          >
            {t('AI-советник', 'AI Advisor')}
          </button>
        </div>
      </Panel>
      <Panel
        title={t('Цели', 'Goals')}
        subtitle={t('Персональные рекомендации', 'Personal recommendations')}
        right={
          <button
            type='button'
            className='link'
            onClick={() => navigate('/consent')}
          >
            {t('Политика', 'Policy')}
          </button>
        }
      />
    </section>
  );
}
