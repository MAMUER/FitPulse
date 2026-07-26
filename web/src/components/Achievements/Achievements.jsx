import { useState, useEffect, useRef } from 'react';
import { Chart } from 'chart.js/auto';
import { getAchievements, getProgress } from '../../utils/api';
import './Achievements.css';

export default function Achievements() {
  const [achievements, setAchievements] = useState([]);
  const [progressData, setProgressData] = useState([]);
  const chartRef = useRef(null);
  const chartInstance = useRef(null);

  useEffect(() => {
    loadAchievements();
  }, []);

  useEffect(() => {
    if (progressData.length > 0 && chartRef.current) {
      if (chartInstance.current) chartInstance.current.destroy();
      const ctx = chartRef.current.getContext('2d');
      chartInstance.current = new Chart(ctx, {
        type: 'bar',
        data: {
          labels: progressData.map(p => p.date || p.week || ''),
          datasets: [{
            label: 'Тренировок',
            data: progressData.map(p => p.completed_workouts ?? p.count ?? p.value ?? 0),
            backgroundColor: 'rgba(255,55,95,0.6)',
            borderRadius: 8
          }]
        },
        options: {
          responsive: true,
          plugins: { legend: { display: false } },
          scales: {
            y: { beginAtZero: true, ticks: { color: '#8e8e93' }, grid: { color: '#2c2c2e' } },
            x: { ticks: { color: '#8e8e93' }, grid: { display: false } }
          }
        }
      });
    }
  }, [progressData]);

  const loadAchievements = async () => {
    try {
      const data = await getAchievements();
      setAchievements(data?.achievements || []);
      const progress = await getProgress();
      setProgressData(progress?.progress_data || progress?.data || []);
    } catch (e) {
      console.error('Failed to load achievements:', e);
    }
  };

  const iconMap = {
    first_workout: '🏃',
    week_streak: '🔥',
    ten_workouts: '💪',
    fifty_workouts: '⭐',
    hundred_days: '📊',
    master_sport: '🏆',
  };

  const competitions = [
    { name: 'Персональный рекорд', desc: 'Пройдите 10000 шагов за день', status: 'active', participants: 1, rank: null },
    { name: 'Серия тренировок', desc: 'Тренируйтесь 3 дня подряд', status: 'upcoming', participants: 1, rank: null },
    { name: 'Месяц активности', desc: 'Тренируйтесь 20 дней в месяце', status: 'upcoming', participants: 1, rank: null }
  ];

  const statusLabels = { active: 'Активно', upcoming: 'Скоро', finished: 'Завершено' };

  return (
    <div className="view active">
      <h3>🏆 Достижения</h3>
      <div id="achievementsList" className="achievements-grid">
        {achievements.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)', textAlign: 'center' }}>Нет достижений</p>
        ) : (
          achievements.map(a => {
            const unlocked = !!a.earned_date;
            return (
              <div key={a.achievement_id} className={`achievement-card ${unlocked ? 'unlocked' : 'locked'}`}>
                <div className="achievement-icon">{a.icon_url || iconMap[a.achievement_id] || '🏆'}</div>
                <div className="achievement-name">{a.title || ''}</div>
                <div className="achievement-desc">{a.description || ''}</div>
                <div className="achievement-progress">{unlocked ? 'Получено' : 'Заблокировано'}</div>
              </div>
            );
          })
        )}
      </div>

      <h3 style={{ marginTop: 24 }}>🏅 Персональные челленджи</h3>
      <div id="competitionsList" className="competitions-list">
        {competitions.map(c => (
          <div key={c.name} className="competition-card">
            <div className="competition-header">
              <div className="competition-name">{c.name}</div>
              <span className={`competition-status ${c.status}`}>{statusLabels[c.status]}</span>
            </div>
            <div className="competition-desc">{c.desc}</div>
            <div className="competition-meta">
              <span>Персональный челлендж</span>
              {c.rank && <span className="competition-rank">🏅 Место: {c.rank}</span>}
            </div>
          </div>
        ))}
      </div>

      <h3 style={{ marginTop: 24 }}>📈 Прогресс</h3>
      <div style={{ background: 'var(--bg-card)', borderRadius: 'var(--radius-lg)', padding: 16, marginTop: 12 }}>
        <canvas ref={chartRef} id="progressChart" height={220} />
      </div>
    </div>
  );
}
