import { useEffect, useRef, useState } from 'react';
import Modal from '../components/Modal';
import MonthSwitcher from '../components/MonthSwitcher';
import WaterTracker from '../components/WaterTracker';
import { useApp } from '../contexts/AppContext';

const WEEKDAYS = {
  ru: ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'],
  en: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'],
};

const METRICS = [
  {
    key: 'pulse',
    icon: 'fa-heart-pulse',
    value: '72',
    unit: 'bpm',
    label: ['Пульс', 'Pulse'],
    color: 'var(--red)',
  },
  {
    key: 'sleep',
    icon: 'fa-moon',
    value: '7.2',
    unit: ['часа', 'hours'],
    label: ['Сон', 'Sleep'],
    color: 'var(--purple)',
  },
  {
    key: 'oxygen',
    icon: 'fa-lungs',
    value: '98',
    unit: '%',
    label: ['Кислород', 'Oxygen'],
    color: 'var(--blue)',
  },
  {
    key: 'stress',
    icon: 'fa-brain',
    value: '32',
    unit: ['уровень', 'level'],
    label: ['Стресс', 'Stress'],
    color: 'var(--orange)',
  },
];

export default function Home() {
  const {
    state,
    t,
    getMonthName,
    go,
    toggleTheme,
    openMetric,
    changeCalendar,
    openCalendarPage,
    openStory,
    showAllStories,
    closeStory,
    nextStory,
    prevStory,
    loadBiometrics,
    loadTrainingPlans,
    loadAchievements,
    loadProgress,
    nextHack,
    loadFriendRequests,
    respondFriendRequest,
  } = useApp();
  const storyStrip = useRef(null);
  const [notifOpen, setNotifOpen] = useState(false);
  const [guestBannerDismissed, setGuestBannerDismissed] = useState(false);

  useEffect(() => {
    loadBiometrics();
    loadTrainingPlans();
    loadAchievements();
    loadProgress();
    loadFriendRequests();
  }, [
    loadBiometrics,
    loadTrainingPlans,
    loadAchievements,
    loadProgress,
    loadFriendRequests,
  ]);

  const friendRequests = state.friendRequests || [];
  const hasUnread = friendRequests.length > 0;

  useEffect(() => {
    const timer = setInterval(() => nextHack(), 10000);
    return () => clearInterval(timer);
  }, [nextHack]);

  const profileName = state.profile?.name || '';
  const stories = state.stories || [];
  const storySeen = state.storySeen || [];
  const now = new Date();
  const monthIndex = state.calendarMonthIndex ?? now.getMonth();
  const year = state.calendarYear ?? now.getFullYear();
  const firstDay = new Date(year, monthIndex, 1);
  const startOffset = (firstDay.getDay() + 6) % 7;
  const daysInMonth = new Date(year, monthIndex + 1, 0).getDate();
  const calendarEvents = state.calendarEvents || {};
  const today = now.getDate();

  // Реальные значения из подключённого устройства (state.biometrics), а не
  // заглушки. Если данных нет — показываем «—», а не выдуманные числа.
  const bioRecords = state.biometrics || [];
  const recordType = (r) =>
    String(r.metric_type ?? r.metricType ?? r.type ?? '').toLowerCase();
  const recordValue = (r) => {
    const v = r.value ?? r.Value;
    return typeof v === 'number' ? v : Number.parseFloat(v);
  };
  const recordTime = (r) => {
    const raw = r.timestamp ?? r.measured_at ?? r.date ?? r.Timestamp;
    const ms = raw ? new Date(raw).getTime() : 0;
    return Number.isFinite(ms) ? ms : 0;
  };
  const latestOf = (...types) => {
    let best = null;
    let bestTime = -1;
    for (const r of bioRecords) {
      const rt = recordType(r);
      if (!types.some((tt) => rt === tt || rt.includes(tt))) continue;
      const tm = recordTime(r);
      if (tm >= bestTime) {
        bestTime = tm;
        best = recordValue(r);
      }
    }
    return best;
  };
  const fmt = (v, digits = 0) => {
    if (v == null || Number.isNaN(v)) return '—';
    const s = Number(v).toFixed(digits);
    return s.replace(/\.0+$/, '');
  };
  const metricValue = (key) => {
    switch (key) {
      case 'pulse':
        return fmt(latestOf('heart_rate', 'heart', 'pulse'));
      case 'sleep':
        return fmt(latestOf('sleep', 'sleep_hours'), 1);
      case 'oxygen':
        return fmt(latestOf('spo2', 'oxygen'));
      case 'stress':
        return fmt(latestOf('stress'));
      default:
        return '—';
    }
  };
  const stepsVal = fmt(latestOf('steps', 'step_count', 'шаги'));
  const kcalVal = fmt(latestOf('calories', 'kcal', 'calories_burned'));
  const sleepVal = fmt(latestOf('sleep', 'sleep_hours'), 1);
  const workoutMinVal =
    state.trainingMinutes != null ? fmt(state.trainingMinutes) : '—';
  // Готовность (readiness): если есть пульс/стресс — 100−стресс, иначе «—».
  const stressVal = latestOf('stress');
  const readinessScore =
    stressVal != null ? Math.max(0, 100 - Number(stressVal)) : '—';

  const scrollStories = (direction) => {
    storyStrip.current?.scrollBy({
      left: direction * 180,
      behavior: 'smooth',
    });
  };

  const openMonth = () => {
    openCalendarPage(true);
    go('calendar');
  };

  const selectedStoryIndex = stories.indexOf(state.selectedStory);

  return (
    <section className='home'>
      {state.guest && !guestBannerDismissed && (
        <output className='fp-guest-banner'>
          <i className='fas fa-circle-info' />
          <span>
            {t(
              'Гостевой режим: данные не сохраняются на сервере. Войдите, чтобы пользоваться всеми функциями.',
              'Guest mode: data is not saved to the server. Sign in to use all features.'
            )}
          </span>
          <button
            type='button'
            className='fp-guest-banner-close'
            aria-label={t('Закрыть', 'Close')}
            onClick={() => setGuestBannerDismissed(true)}
          >
            <i className='fas fa-times' />
          </button>
        </output>
      )}
      <header className='topbar'>
        <div>
          <div className='eyebrow'>FITPULSE · {t('СЕГОДНЯ', 'TODAY')}</div>
          <h1>
            {t('Привет', 'Hello')}
            {profileName ? `, ${profileName}` : '!'}
          </h1>
        </div>
        <div className='row' style={{ gap: 7 }}>
          <button
            type='button'
            className={`notification${hasUnread ? ' has-unread' : ''}`}
            aria-label={t('Уведомления', 'Notifications')}
            onClick={() => setNotifOpen(true)}
          >
            <i className='fas fa-bell' />
          </button>
          <button
            type='button'
            className='icon'
            aria-label={t('Сменить тему', 'Toggle theme')}
            onClick={toggleTheme}
          >
            <i
              className={`fas ${state.theme === 'dark' ? 'fa-sun' : 'fa-moon'}`}
            />
          </button>
          <button
            type='button'
            className='photo'
            aria-label={t('Профиль', 'Profile')}
            onClick={() => go('profile')}
          >
            {state.profileImage ? (
              <img src={state.profileImage} alt='' />
            ) : (
              profileName[0] || 'Я'
            )}
          </button>
        </div>
      </header>

      {!state.surveyCompleted && (
        <div className='guest-banner'>
          <strong>
            {t(
              'Персонализация не завершена',
              'Personalization is not complete'
            )}
          </strong>
          <span className='muted'>
            {t(
              'FitPulse пока использует базовые рекомендации. Завершите опрос, чтобы учитывать ваши цели, ограничения и предпочтения.',
              'FitPulse is using basic recommendations. Complete the survey to account for your goals, restrictions and preferences.'
            )}
          </span>
          <button
            type='button'
            className='secondary full'
            style={{ marginTop: 8 }}
            onClick={() => go('survey')}
          >
            {t('Продолжить опрос', 'Continue survey')}
          </button>
        </div>
      )}

      <section className='hero'>
        <div className='eyebrow'>FITPULSE DAILY RHYTHM</div>
        <h2>{t('Готов к движению', 'Ready to move')}</h2>
        <p>
          {t(
            'Сегодня тело готово к силовой нагрузке и полноценному восстановлению.',
            'Today your body is ready for strength training and full recovery.'
          )}
        </p>
        <button
          type='button'
          className='primary'
          onClick={() => go('training')}
        >
          {t('Начать тренировку', 'Start workout')}{' '}
          <i className='fas fa-arrow-right' />
        </button>
        <div className='readiness'>
          <span>{readinessScore}</span>
        </div>
      </section>

      <div className='between section'>
        <h3>{t('Истории', 'Stories')}</h3>
        <button type='button' className='link' onClick={showAllStories}>
          {t('Все истории', 'All stories')}
        </button>
      </div>
      <div className='fp-story-carousel'>
        <button
          type='button'
          className='fp-story-arrow left'
          aria-label={t('Предыдущие истории', 'Previous stories')}
          onClick={() => scrollStories(-1)}
        >
          <i className='fas fa-chevron-left' />
        </button>
        <div className='fp-story-strip' ref={storyStrip}>
          {stories.map((story, index) => (
            <button
              type='button'
              key={story.id || story.title}
              className={`fp-story-card ${storySeen[index] ? 'seen' : ''}`}
              style={{ backgroundImage: `url('${story.image}')` }}
              onClick={() => openStory(index)}
            >
              <span className='fp-story-seen-dot' aria-hidden='true' />
              <strong>{story.title}</strong>
              <small>{story.description}</small>
            </button>
          ))}
        </div>
        <button
          type='button'
          className='fp-story-arrow right'
          aria-label={t('Следующие истории', 'Next stories')}
          onClick={() => scrollStories(1)}
        >
          <i className='fas fa-chevron-right' />
        </button>
      </div>

      <div className='between section'>
        <h3>{t('Биоритм сегодня', 'Biorhythm today')}</h3>
        <button type='button' className='link' onClick={() => go('body')}>
          {t('Открыть всё', 'View all')}
        </button>
      </div>
      <section className='metrics'>
        {METRICS.map((metric) => (
          <button
            type='button'
            key={metric.key}
            className='card metric'
            style={{ '--metric-color': metric.color }}
            onClick={() => openMetric(metric.key)}
          >
            <i className={`fas ${metric.icon}`} />
            <strong>{metricValue(metric.key)}</strong>
            <small>
              {Array.isArray(metric.unit) ? t(...metric.unit) : metric.unit}
            </small>
            <p>{t(...metric.label)}</p>
          </button>
        ))}
      </section>

      <div className='between section'>
        <h3>{t('Календарь', 'Calendar')}</h3>
        <button type='button' className='link' onClick={openMonth}>
          {t('Все дни', 'All days')}
        </button>
      </div>
      <section className='card'>
        <MonthSwitcher
          label={`${getMonthName(monthIndex)} ${year}`}
          onPrev={() => changeCalendar(-1)}
          onNext={() => changeCalendar(1)}
        />
        <div className='week' style={{ marginTop: 10 }}>
          {(WEEKDAYS[state.language] || WEEKDAYS.ru).map((weekday) => (
            <div className='day' key={weekday}>
              {weekday}
            </div>
          ))}
        </div>
        <div className='month home-month'>
          {Array.from({ length: startOffset }, (_, index) => (
            <span className='muted' key={`empty-${index}`} />
          ))}
          {Array.from({ length: daysInMonth }, (_, index) => {
            const day = index + 1;
            const dateKey = `${year}-${String(monthIndex + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
            const hasEvent = (calendarEvents[dateKey] || []).length > 0;
            const isToday =
              day === today &&
              monthIndex === now.getMonth() &&
              year === now.getFullYear();
            return (
              <button
                type='button'
                key={day}
                className={`${hasEvent ? 'has-event' : ''} ${isToday ? 'active' : ''}`}
                aria-label={`${t('Открыть день', 'Open day')} ${day}`}
                onClick={openMonth}
              >
                {day}
              </button>
            );
          })}
        </div>
      </section>

      <div className='between section'>
        <h3>{t('Активность сегодня', "Today's activity")}</h3>
        <button type='button' className='link' onClick={() => go('body')}>
          {t('Подробнее', 'Details')}
        </button>
      </div>
      <section className='card'>
        <div className='legend'>
          <span>
            <i className='fas fa-circle' style={{ color: 'var(--green)' }} />
            {stepsVal} {t('шаг', 'steps')}
          </span>
          <span>
            <i className='fas fa-circle' style={{ color: 'var(--blue)' }} />
            {kcalVal} {t('ккал', 'kcal')}
          </span>
          <span>
            <i className='fas fa-circle' style={{ color: 'var(--orange)' }} />
            {workoutMinVal} {t('мин тренировки', 'min workout')}
          </span>
          <span>
            <i className='fas fa-circle' style={{ color: 'var(--purple)' }} />
            {sleepVal} {t('ч сна', 'h sleep')}
          </span>
        </div>
      </section>

      <WaterTracker />

      <div className='section'>
        <h3>{t('Лайфхак', 'Life hack')}</h3>
      </div>
      {(() => {
        const hacks = state.hackList || [];
        const hack = hacks[(state.hackIndex || 0) % (hacks.length || 1)];
        if (!hack) return null;
        return (
          <div className='fp-hack'>
            <strong>
              <i
                className={`fas ${hack[0]}`}
                style={{ color: 'var(--gold)' }}
              />{' '}
              {hack[1]}
            </strong>
            <p>{hack[2]}</p>
            <button
              type='button'
              aria-label={t('Следующий лайфхак', 'Next tip')}
              onClick={nextHack}
            >
              <i className='fas fa-arrow-right' />
            </button>
          </div>
        );
      })()}

      {state.selectedStory && (
        <div className='fp-story-viewer'>
          <button
            type='button'
            aria-hidden='true'
            tabIndex={-1}
            className='fp-modal-backdrop'
            onClick={closeStory}
          />
          <dialog
            open
            className='fp-story-view fp-dialog'
            aria-label={state.selectedStory.title}
            style={{ backgroundImage: `url('${state.selectedStory.image}')` }}
          >
            <div className='fp-story-progress'>
              {stories.map((story, index) => (
                <i
                  key={story.id || story.title}
                  className={index <= selectedStoryIndex ? 'active' : ''}
                />
              ))}
            </div>
            <button
              type='button'
              className='fp-story-close'
              aria-label={t('Закрыть историю', 'Close story')}
              onClick={closeStory}
            >
              ×
            </button>
            <button
              type='button'
              className='fp-story-view-arrow left'
              aria-label={t('Предыдущая история', 'Previous story')}
              onClick={prevStory}
            >
              <i className='fas fa-chevron-left' />
            </button>
            <button
              type='button'
              className='fp-story-view-arrow right'
              aria-label={t('Следующая история', 'Next story')}
              onClick={nextStory}
            >
              <i className='fas fa-chevron-right' />
            </button>
            <div className='fp-story-info'>
              <h2>{state.selectedStory.title}</h2>
              <p>{state.selectedStory.description}</p>
            </div>
          </dialog>
        </div>
      )}

      {notifOpen && (
        <Modal
          className='fp-profile-modal'
          label={t('Уведомления', 'Notifications')}
          onClose={() => setNotifOpen(false)}
        >
          <section className='fp-profile-box'>
            <div className='between'>
              <div>
                <div className='eyebrow'>
                  {t('УВЕДОМЛЕНИЯ', 'NOTIFICATIONS')}
                </div>
                <h3>{t('Заявки в друзья', 'Friend requests')}</h3>
              </div>
              <button
                type='button'
                className='icon'
                aria-label={t('Закрыть', 'Close')}
                onClick={() => setNotifOpen(false)}
              >
                ×
              </button>
            </div>
            {friendRequests.length === 0 ? (
              <p className='muted'>
                {t('Новых уведомлений нет', 'No new notifications')}
              </p>
            ) : (
              friendRequests.map((req) => (
                <div className='body-row' key={req.id}>
                  <span>
                    <b>{req.display_name || t('Участник', 'Member')}</b>
                  </span>
                  <div className='row-buttons'>
                    <button
                      type='button'
                      className='secondary'
                      onClick={() => respondFriendRequest(req.id, 'accept')}
                    >
                      {t('Принять', 'Accept')}
                    </button>
                    <button
                      type='button'
                      className='danger'
                      onClick={() => respondFriendRequest(req.id, 'decline')}
                    >
                      {t('Отклонить', 'Decline')}
                    </button>
                  </div>
                </div>
              ))
            )}
          </section>
        </Modal>
      )}
    </section>
  );
}
