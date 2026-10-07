import { useEffect } from 'react';
import { useApp } from '../contexts/AppContext';

export default function Videos() {
  const { state, t, go, loadVideos } = useApp();

  useEffect(() => {
    loadVideos();
  }, [loadVideos]);

  const videos =
    state.videos && state.videos.length > 0
      ? state.videos
      : state.trainingVideos || [];

  return (
    <section className='videos'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>
            {t('Видео-тренировки', 'Video workouts')}
          </div>
        </div>
        <div className='panel-body'>
          {videos.length === 0 ? (
            <div className='muted'>{t('Нет видео', 'No videos')}</div>
          ) : (
            videos.map((v, i) => (
              <div key={v.id || i} className='list-row'>
                <div>
                  <div className='list-title'>{v.title}</div>
                  <div className='muted'>
                    {v.minutes || v.duration || ''} {t('мин', 'min')} ·{' '}
                    {v.kcal || v.calories || 0} {t('ккал', 'kcal')}
                  </div>
                </div>
                <button
                  type='button'
                  className='primary'
                  onClick={() => go('training')}
                >
                  {t('Старт', 'Start')}
                </button>
              </div>
            ))
          )}
        </div>
      </div>
    </section>
  );
}
