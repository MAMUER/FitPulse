import { useEffect } from 'react';
import Modal from '../components/Modal';
import MonthSwitcher from '../components/MonthSwitcher';
import { useApp } from '../contexts/AppContext';
import { getDateKey, getEventsForMonth, getWeekDays } from '../utils/helpers';

const QUICK_ENTRIES = [
  [
    ['gym', 'fa-dumbbell'],
    ['pool', 'fa-swimmer'],
    ['recovery', 'fa-heart'],
  ],
  [
    ['work', 'fa-briefcase'],
    ['study', 'fa-graduation-cap'],
    ['note', 'fa-pen'],
  ],
];

const QUICK_LABELS = {
  gym: ['Зал', 'Gym'],
  pool: ['Бассейн', 'Pool'],
  recovery: ['Восстановление', 'Recovery'],
  work: ['Работа', 'Work'],
  study: ['Учёба', 'Study'],
  note: ['Заметка', 'Note'],
};

export default function Calendar() {
  const {
    state,
    t,
    getMonthName,
    loadCalendarEvents,
    changeCalendar,
    quickAddEvent,
    openCalendarPage,
    closeCalendarPage,
    selectCalendarDay,
    closeDateModal,
    addEvent,
    deleteCalendarEvent,
  } = useApp();

  useEffect(() => {
    loadCalendarEvents();
  }, [loadCalendarEvents]);

  const year = state.calendarYear ?? new Date().getFullYear();
  const monthIndex = state.calendarMonthIndex ?? new Date().getMonth();
  const calendarEvents = state.calendarEvents || {};
  const today = new Date();
  const weekDays = getWeekDays(state.language);
  const daysInMonth = new Date(year, monthIndex + 1, 0).getDate();
  const startOffset = (new Date(year, monthIndex, 1).getDay() + 6) % 7;
  const monthEventsByDay = getEventsForMonth(calendarEvents, year, monthIndex);
  const selectedDay = state.dateModalDate ?? state.selectedCalendarDay;

  const monthEvents = Object.entries(calendarEvents)
    .flatMap(([key, list]) =>
      Array.isArray(list)
        ? list.map((event, index) => ({ key, index, ...event }))
        : []
    )
    .filter((event) => {
      const [eventYear, eventMonth] = String(event.key).split('-').map(Number);
      return eventYear === year && eventMonth === monthIndex + 1;
    })
    .sort((a, b) => String(a.time || '').localeCompare(String(b.time || '')));

  const selectedKey =
    selectedDay != null ? getDateKey(year, monthIndex, selectedDay) : null;
  const selectedEvents = selectedKey ? calendarEvents[selectedKey] || [] : [];

  return (
    <section className='calendar'>
      <div className='page-title'>
        <div className='eyebrow'>PLANNER</div>
        <h2>{t('Календарь', 'Calendar')}</h2>
        <p>
          {t(
            'Дела, встречи, питание, тренировки и восстановление.',
            'Tasks, meetings, meals, workouts and recovery.'
          )}
        </p>
        <button
          type='button'
          className='primary'
          style={{ marginTop: 10 }}
          onClick={() => openCalendarPage(true)}
        >
          <i className='fas fa-calendar-alt' />{' '}
          {t('Открыть календарь', 'Open calendar')}
        </button>
      </div>

      <div className='section'>
        <h3>{t('Быстрые записи', 'Quick entries')}</h3>
      </div>
      {QUICK_ENTRIES.map((row) => (
        <div className='form-row' key={row[0][0]}>
          {row.map(([type, icon]) => (
            <button
              type='button'
              className='secondary'
              key={type}
              onClick={() => quickAddEvent(type)}
            >
              <i className={`fas ${icon}`} /> {t(...QUICK_LABELS[type])}
            </button>
          ))}
        </div>
      ))}

      <section className='card'>
        <MonthSwitcher
          label={`${getMonthName(monthIndex)} ${year}`}
          onPrev={() => changeCalendar(-1)}
          onNext={() => changeCalendar(1)}
        />
        <div className='week' style={{ marginTop: 10 }}>
          {weekDays.map((day) => (
            <div className='day' key={day} style={{ fontSize: 8 }}>
              {day}
            </div>
          ))}
        </div>
        <div className='month' style={{ marginTop: 8 }}>
          {Array.from({ length: startOffset }, (_, index) => (
            <span className='muted' key={`empty-${index}`} />
          ))}
          {Array.from({ length: daysInMonth }, (_, index) => {
            const day = index + 1;
            const hasEvent = (monthEventsByDay[day] || []).length > 0;
            const isToday =
              day === today.getDate() &&
              monthIndex === today.getMonth() &&
              year === today.getFullYear();
            return (
              <button
                type='button'
                key={day}
                className={`${hasEvent ? 'has-event' : ''} ${
                  isToday ? 'active' : ''
                } ${selectedDay === day ? 'selected' : ''}`}
                onClick={() => selectCalendarDay(day)}
              >
                {day}
              </button>
            );
          })}
        </div>
      </section>

      {state.calendarOpen && (
        <section className='fp-calendar-page'>
          <div className='fp-calendar-head'>
            <div>
              <div className='eyebrow'>{t('ПЛАНИРОВЩИК', 'PLANNER')}</div>
              <h2>
                {getMonthName(monthIndex)} {year}
              </h2>
            </div>
            <button
              type='button'
              className='icon'
              aria-label={t('Закрыть', 'Close')}
              onClick={closeCalendarPage}
            >
              ×
            </button>
          </div>

          <div className='fp-calendar-toolbar'>
            <button
              type='button'
              className='secondary'
              onClick={() => changeCalendar(-1)}
            >
              ‹ {t('Назад', 'Back')}
            </button>
            <button
              type='button'
              className='secondary'
              onClick={() => openCalendarPage(true)}
            >
              {t('Сегодня', 'Today')}
            </button>
            <button
              type='button'
              className='secondary'
              onClick={() => changeCalendar(1)}
            >
              {t('Вперёд', 'Next')} ›
            </button>
          </div>

          <div className='week' style={{ marginTop: 10 }}>
            {weekDays.map((day) => (
              <div className='day' key={day}>
                {day}
              </div>
            ))}
          </div>
          <div className='fp-calendar-dates' style={{ marginTop: 8 }}>
            {Array.from({ length: startOffset }, (_, index) => (
              <div
                className='fp-calendar-date'
                aria-hidden='true'
                key={`pad-${index}`}
                style={{ visibility: 'hidden' }}
              />
            ))}
            {Array.from({ length: daysInMonth }, (_, index) => {
              const day = index + 1;
              const list = monthEventsByDay[day] || [];
              const isToday =
                day === today.getDate() &&
                monthIndex === today.getMonth() &&
                year === today.getFullYear();
              return (
                <button
                  type='button'
                  className={`fp-calendar-date ${list.length ? 'active' : ''} ${
                    selectedDay === day ? 'selected' : ''
                  }`}
                  aria-label={`${day} ${getMonthName(monthIndex)}`}
                  key={day}
                  onClick={() => selectCalendarDay(day)}
                >
                  {day}
                  <span className='fp-dots'>
                    {list.slice(0, 4).map((event, dotIndex) => (
                      <i
                        className='fp-dot'
                        key={`${day}-dot-${dotIndex}`}
                        style={{ background: event.color || '#19d8da' }}
                      />
                    ))}
                  </span>
                  {isToday ? <small>{t('Сегодня', 'Today')}</small> : null}
                </button>
              );
            })}
          </div>

          <div className='card' style={{ marginTop: 10 }}>
            <h3>{t('Правила и гигиена', 'Rules & hygiene')}</h3>
            <p className='muted'>
              {t(
                'Соблюдай технику, начинай с разминки и используй чистый инвентарь.',
                'Use proper form, warm up first and keep equipment clean.'
              )}
            </p>
          </div>

          <div className='section'>
            <div className='between'>
              <h3>{t('События месяца', 'Month events')}</h3>
              <button
                type='button'
                className='primary'
                onClick={() =>
                  selectCalendarDay(selectedDay || today.getDate())
                }
              >
                + {t('Добавить', 'Add')}
              </button>
            </div>
          </div>
          <div>
            {monthEvents.length === 0 ? (
              <p className='muted' style={{ padding: '12px 0' }}>
                {t(
                  'Событий нет. Добавьте первое.',
                  'No events yet. Add the first one.'
                )}
              </p>
            ) : (
              monthEvents.map((event) => (
                <div className='fp-event' key={`${event.key}-${event.index}`}>
                  <span
                    className='fp-event-color'
                    style={{ '--event-color': event.color || '#19d8da' }}
                  />
                  <div className='fp-event-content'>
                    <strong>{event.title}</strong>
                    <small>
                      {[event.date, event.time, event.info]
                        .filter(Boolean)
                        .join(' · ')}
                    </small>
                  </div>
                  <button
                    type='button'
                    className='secondary'
                    onClick={() => deleteCalendarEvent(event.key, event.index)}
                  >
                    {t('Удалить', 'Delete')}
                  </button>
                </div>
              ))
            )}
          </div>
        </section>
      )}

      {state.showDateModal && selectedDay != null && (
        <Modal
          className='fp-profile-modal'
          label={`${selectedDay} ${getMonthName(monthIndex)}`}
          onClose={closeDateModal}
        >
          <section className='fp-profile-box'>
            <div className='between'>
              <div>
                <div className='eyebrow'>
                  {getMonthName(monthIndex)} {year}
                </div>
                <h3>
                  {selectedDay} {getMonthName(monthIndex)}
                </h3>
              </div>
              <button
                type='button'
                className='icon'
                aria-label={t('Закрыть', 'Close')}
                onClick={closeDateModal}
              >
                ×
              </button>
            </div>
            {selectedEvents.length === 0 ? (
              <p className='muted' style={{ padding: '14px 0' }}>
                {t('Нет событий на эту дату', 'No events on this date')}
              </p>
            ) : (
              selectedEvents.map((event, index) => (
                <div className='fp-event' key={`${selectedKey}-${index}`}>
                  <span
                    className='fp-event-color'
                    style={{ '--event-color': event.color || '#19d8da' }}
                  />
                  <div className='fp-event-content'>
                    <strong>{event.title}</strong>
                    <small>
                      {[event.time, event.info].filter(Boolean).join(' · ')}
                    </small>
                  </div>
                  <button
                    type='button'
                    className='secondary'
                    onClick={() => deleteCalendarEvent(selectedKey, index)}
                  >
                    {t('Удалить', 'Delete')}
                  </button>
                </div>
              ))
            )}
            <div className='field' style={{ marginTop: 10 }}>
              <label htmlFor='eventTitle'>{t('Название', 'Title')}</label>
              <input
                id='eventTitle'
                placeholder={t('Новое событие', 'New event')}
              />
            </div>
            <div className='form-row'>
              <div className='field'>
                <label htmlFor='eventType'>{t('Тип', 'Type')}</label>
                <select id='eventType' defaultValue='training'>
                  <option value='training'>
                    {t('Тренировка', 'Training')}
                  </option>
                  <option value='food'>{t('Питание', 'Meal')}</option>
                  <option value='recovery'>
                    {t('Восстановление', 'Recovery')}
                  </option>
                  <option value='meeting'>{t('Встреча', 'Meeting')}</option>
                  <option value='note'>{t('Заметка', 'Note')}</option>
                </select>
              </div>
              <div className='field'>
                <label htmlFor='eventTime'>{t('Время', 'Time')}</label>
                <input id='eventTime' type='time' defaultValue='19:00' />
              </div>
            </div>
            <button
              type='button'
              className='primary full'
              style={{ marginTop: 10 }}
              onClick={() => addEvent()}
            >
              + {t('Добавить событие', 'Add event')}
            </button>
          </section>
        </Modal>
      )}

      <p className='muted' style={{ marginTop: 12, fontSize: 10 }}>
        {t('Выбранная дата', 'Selected date')}: {selectedDay || today.getDate()}{' '}
        {getMonthName(monthIndex)}
      </p>
    </section>
  );
}
