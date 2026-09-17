import { useApp } from '../contexts/AppContext';

export default function Training() {
  const { state, t, stopWorkout, startWorkout, go } = useApp();
  const active = !!state.workoutStartDate;

  return (
    <section className='training'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Тренировка', 'Workout')}</div>
          <div className='panel-sub'>
            {active
              ? t('В процессе', 'In progress')
              : t('Не начата', 'Not started')}
          </div>
        </div>
        <div className='panel-body'>
          {active && (
            <div className='metric'>
              <div className='metric-val'>
                {state.workoutDuration || 0}
                <span className='muted'> {t('сек', 'sec')}</span>
              </div>
              <div className='metric-label'>
                {t('Длительность', 'Duration')}
              </div>
            </div>
          )}
          {active ? (
            <button className='danger full' onClick={stopWorkout}>
              {t('Завершить тренировку', 'End workout')}
            </button>
          ) : (
            <button className='primary full' onClick={startWorkout}>
              {t('Начать тренировку', 'Start workout')}
            </button>
          )}
          <button className='secondary full' onClick={() => go('home')}>
            {t('На главную', 'Go home')}
          </button>
        </div>
      </div>
    </section>
  );
}
