import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';

export default function Achievements() {
  const { state, t, go, loadAchievements } = useApp();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadAchievements().then(() => setLoading(false));
  }, [loadAchievements]);

  const achievements =
    state.achievementsBackend && state.achievementsBackend.length > 0
      ? state.achievementsBackend
      : state.achievements || [];

  let content;
  if (loading) {
    content = <div className='muted'>{t('Загрузка...', 'Loading...')}</div>;
  } else if (achievements.length === 0) {
    content = (
      <div className='muted'>{t('Нет достижений', 'No achievements')}</div>
    );
  } else {
    content = (
      <div className='achievements-grid'>
        {achievements.map((ach, i) => (
          <div
            key={ach.id || ach.name || i}
            className={`achievement-card ${ach.done || ach.achieved ? 'done' : ''}`}
          >
            <div className='achievement-icon'>
              <i className={`fas ${ach.icon || 'fa-trophy'}`}></i>
            </div>
            <div className='achievement-name'>{ach.name}</div>
            <div className='achievement-desc'>
              {ach.desc || ach.description || ''}
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <section className='achievements'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Достижения', 'Achievements')}</div>
          <button className='secondary' onClick={() => go('home')}>
            {t('Назад', 'Back')}
          </button>
        </div>
        <div className='panel-body'>{content}</div>
      </div>
    </section>
  );
}
