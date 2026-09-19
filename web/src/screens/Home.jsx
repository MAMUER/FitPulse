import { useNavigate } from 'react-router-dom';
import { useApp } from '../contexts/AppContext';

export default function Home() {
  const { state, t, drinkWater, startWorkout, go } = useApp();
  const navigate = useNavigate();
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
