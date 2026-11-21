import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';

// Achievements — отдельная страница достижений. Раньше кнопки «Достижения» и
// «Серия» в профиле просто скроллили к пустому блоку, а отдельного роута не было.
export default function Achievements() {
  const { state, t, go, loadAchievements, showAchievement } = useApp();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    Promise.resolve(loadAchievements()).finally(() => {
      if (!cancelled) setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, [loadAchievements]);

  const achievements =
    state.achievementsBackend && state.achievementsBackend.length > 0
      ? state.achievementsBackend
      : state.achievements || [];

  const total = achievements.length;
  const done = achievements.filter((item) => item.done || item.achieved).length;
  const pct = total > 0 ? Math.round((done / total) * 100) : 0;
  const streak = state.streak || 0;
  const maxStreak = state.maxStreak || 0;

  return (
    <section className='achievements-page'>
      <div className='page-title'>
        <div className='eyebrow'>FITPULSE ACHIEVEMENTS</div>
        <h2>{t('Достижения', 'Achievements')}</h2>
        <p>
          {t(
            'Награды за тренировки, серию и дисциплину.',
            'Rewards for workouts, streaks and consistency.'
          )}
        </p>
      </div>

      <div className='fp-achievements-summary'>
        <div className='fp-stat'>
          <strong>
            {done}/{total}
          </strong>
          <small>{t('получено', 'unlocked')}</small>
        </div>
        <div className='fp-stat'>
          <strong>{pct}%</strong>
          <small>{t('прогресс', 'progress')}</small>
        </div>
        <div className='fp-stat'>
          <strong>{streak}</strong>
          <small>{t('серия', 'streak')}</small>
        </div>
        <div className='fp-stat'>
          <strong>{maxStreak}</strong>
          <small>{t('рекорд', 'best')}</small>
        </div>
      </div>

      <div className='fp-progress-line'>
        <i style={{ width: `${pct}%` }} />
      </div>

      {loading ? (
        <p className='muted'>{t('Загрузка...', 'Loading...')}</p>
      ) : total === 0 ? (
        <p className='muted'>{t('Нет достижений', 'No achievements')}</p>
      ) : (
        <div className='grid-2'>
          {achievements.map((achievement, index) => {
            const isDone = Boolean(achievement.done || achievement.achieved);
            const icon = achievement.icon || 'fa-trophy';
            const key = achievement.id ?? achievement.name ?? index;
            return (
              <button
                type='button'
                key={key}
                className={`achievement-card ${isDone ? 'done' : ''}`}
                onClick={() => showAchievement?.(achievement.name)}
              >
                <i className={`fas ${icon}`} />
                <strong>{achievement.name}</strong>
                <small>{achievement.desc || achievement.description}</small>
              </button>
            );
          })}
        </div>
      )}

      <button
        type='button'
        className='secondary full'
        onClick={() => go('home')}
      >
        <i className='fas fa-arrow-left' /> {t('Назад', 'Back')}
      </button>
    </section>
  );
}
