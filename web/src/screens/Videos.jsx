import { useApp } from '../contexts/AppContext';

export default function Videos() {
  const { state, t, go } = useApp();
  const list = state.trainingVideos || [
    { title: 'Кардио', minutes: 20, kcal: 180 },
    { title: 'Силовая', minutes: 30, kcal: 240 },
    { title: 'Йога', minutes: 25, kcal: 120 },
    { title: 'Растяжка', minutes: 15, kcal: 90 },
  ];

  return (
    <section className='videos'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>
            {t('Видео-тренировки', 'Video workouts')}
          </div>
        </div>
        <div className='panel-body'>
          {list.map((v, i) => (
            <div key={i} className='list-row'>
              <div>
                <div className='list-title'>{v.title}</div>
                <div className='muted'>
                  {v.minutes} {t('мин', 'min')} · {v.kcal} {t('ккал', 'kcal')}
                </div>
              </div>
              <button className='primary' onClick={() => go('training')}>
                {t('Старт', 'Start')}
              </button>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
