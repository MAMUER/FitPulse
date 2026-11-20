import { useEffect } from 'react';
import { useApp } from '../contexts/AppContext';

export default function Leaderboard() {
  const { state, t, loadLeaderboard, syncLeaderboard, reportUser } = useApp();

  useEffect(() => {
    // Отправляем свои очки и подтягиваем общий рейтинг.
    syncLeaderboard();
    loadLeaderboard();
  }, [syncLeaderboard, loadLeaderboard]);

  const entries = state.leaderboard || [];

  return (
    <section className='leaderboard'>
      <div className='page-title'>
        <div className='eyebrow'>LEADERBOARD</div>
        <h2>{t('Лидерборд', 'Leaderboard')}</h2>
        <p>
          {t(
            'Лучшие по очкам за тренировки и активность.',
            'Top users by workout and activity points.'
          )}
        </p>
      </div>

      {entries.length === 0 ? (
        <section className='card'>
          <p className='muted' style={{ fontSize: 11 }}>
            {t('Пока нет участников', 'No participants yet')}
          </p>
        </section>
      ) : (
        <section className='card'>
          {entries.map((entry, index) => (
            <div
              className={`body-row${entry.me ? ' me' : ''}`}
              key={entry.user_id || index}
            >
              <span className='lb-rank'>{index + 1}</span>
              <span>
                <b>{entry.display_name || t('Участник', 'Member')}</b>
                {entry.me ? ` · ${t('вы', 'you')}` : ''}
              </span>
              <b>
                {entry.points} {t('очков', 'pts')}
              </b>
              {!entry.me && (
                <button
                  type='button'
                  className='icon'
                  aria-label={t('Пожаловаться', 'Report')}
                  onClick={() => reportUser(entry.user_id)}
                >
                  <i className='fas fa-flag' />
                </button>
              )}
            </div>
          ))}
        </section>
      )}
    </section>
  );
}
