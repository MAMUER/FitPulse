import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
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
  const [_biometricsLoaded, setBiometricsLoaded] = useState(false);

  useEffect(() => {
    loadBiometrics();
    loadTrainingPlans();
    loadAchievements();
    loadProgress();
    setBiometricsLoaded(true);
  }, [loadBiometrics, loadTrainingPlans, loadAchievements, loadProgress]);

  const biometrics = state.biometrics || [];
  const latestHR = biometrics.find((b) => b.metric_type === 'heart_rate');
  const latestSpO2 = biometrics.find((b) => b.metric_type === 'spo2');
  const latestTemp = biometrics.find((b) => b.metric_type === 'temperature');
  const bmi =
    state.height > 0 && state.weight > 0
      ? +(state.weight / (state.height / 100) ** 2).toFixed(1)
      : null;

  return (
    <section className='home'>
      <div className='panel'>
        <div className='panel-head'>
          <div>
            <div className='panel-title'>{t('Прогресс', 'Progress')}</div>
            <div className='panel-sub'>
              {state.weight ? `${state.weight} кг` : '—'} ·{' '}
              {bmi ? `ИМТ ${bmi}` : 'ИМТ —'}
            </div>
          </div>
          <a className='link' onClick={() => go('body')}>
            {t('Профиль тела', 'Body profile')}
          </a>
        </div>
        <div className='panel-body'>
          <div className='metric'>
            <div className='metric-val'>
              {state.waterCount || 0}
              <span className='muted'> / {state.waterGoal || 2000}</span>
            </div>
            <div className='metric-label'>{t('Вода, мл', 'Water, ml')}</div>
            <button className='primary full' onClick={drinkWater}>
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
      </div>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Тренировки', 'Workouts')}</div>
        </div>
        <div className='panel-body'>
          <button className='primary full' onClick={startWorkout}>
            {t('Начать тренировку', 'Start workout')}
          </button>
          <button className='secondary full' onClick={() => go('training')}>
            {t('Планы тренировок', 'Training plans')}
          </button>
        </div>
      </div>
      <div className='panel'>
        <div className='panel-head'>
          <div>
            <div className='panel-title'>
              {t('Рекомендации', 'Recommendations')}
            </div>
            <div className='panel-sub'>
              {t('На основе ваших данных', 'Based on your data')}
            </div>
          </div>
        </div>
        <div className='panel-body'>
          <button className='secondary full' onClick={() => go('videos')}>
            {t('Видео-тренировки', 'Video workouts')}
          </button>
          <button className='secondary full' onClick={() => go('ai')}>
            {t('AI-советник', 'AI Advisor')}
          </button>
        </div>
      </div>
      <div className='panel'>
        <div className='panel-head'>
          <div>
            <div className='panel-title'>{t('Цели', 'Goals')}</div>
            <div className='panel-sub'>
              {t('Персональные рекомендации', 'Personal recommendations')}
            </div>
          </div>
          <a className='link' onClick={() => navigate('/consent')}>
            {t('Политика', 'Policy')}
          </a>
        </div>
      </div>
    </section>
  );
}
