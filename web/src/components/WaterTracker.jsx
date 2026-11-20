import { useApp } from '../contexts/AppContext';

export default function WaterTracker() {
  const { state, t, drinkWater, removeWater } = useApp();
  const intake = state.waterIntake || 0;
  const goal = state.waterGoal || 8;
  const progress = goal > 0 ? Math.min(100, (intake / goal) * 100) : 0;

  return (
    <div className='water-tracker'>
      <div className='water-icon'>
        <i className='fas fa-droplet' />
      </div>
      <div className='water-info'>
        <strong>{t('Вода', 'Water')}</strong>
        <small>
          {intake} / {goal} {t('стаканов', 'glasses')}
        </small>
      </div>
      <div className='water-progress'>
        <div style={{ width: `${progress}%` }} />
      </div>
      <button
        type='button'
        className='water-btn'
        aria-label={t('Убрать стакан воды', 'Remove a glass of water')}
        onClick={removeWater}
        disabled={intake <= 0}
      >
        <i className='fas fa-minus' />
      </button>
      <button
        type='button'
        className='water-btn'
        aria-label={t('Добавить стакан воды', 'Add a glass of water')}
        onClick={drinkWater}
      >
        <i className='fas fa-plus' />
      </button>
    </div>
  );
}
