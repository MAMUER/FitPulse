import { useEffect, useState } from 'react';
import EmptyState from '../components/EmptyState';
import Panel from '../components/Panel';
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
    content = <EmptyState text={t('Загрузка...', 'Loading...')} />;
  } else if (achievements.length === 0) {
    content = <EmptyState text={t('Нет достижений', 'No achievements')} />;
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
      <Panel
        title={t('Достижения', 'Achievements')}
        right={
          <button
            type='button'
            className='secondary'
            onClick={() => go('home')}
          >
            {t('Назад', 'Back')}
          </button>
        }
      >
        <div className='panel-body'>{content}</div>
      </Panel>
    </section>
  );
}
