import { useState, useEffect, useMemo } from 'react';
import {
  addDays,
  endOfMonth,
  endOfWeek,
  format,
  getWeekDays,
  isSameDay,
  isSameMonth,
  startOfMonth,
  startOfWeek,
} from '../utils/helpers';
import { useApp } from '../contexts/AppContext';

export default function Calendar() {
  const { state, t, addEvent, deleteCalendarEvent, loadCalendarEvents } = useApp();
  const [currentMonth] = useState(new Date());
  const [selectedDay, setSelectedDay] = useState(null);
  const [eventTitle, setEventTitle] = useState('');
  const [eventType, setEventType] = useState('training');
  const [eventTime, setEventTime] = useState('19:00');
  const [eventDesc, setEventDesc] = useState('');
  const [showForm, setShowForm] = useState(false);

  const eventsList = state.calendarEventsList || [];

  useEffect(() => {
    loadCalendarEvents();
  }, [loadCalendarEvents]);

  const monthStart = startOfMonth(currentMonth);
  const monthEnd = endOfMonth(currentMonth);
  const calStart = startOfWeek(monthStart);
  const calEnd = endOfWeek(monthEnd);
  const days = [];
  let day = calStart;
  while (day <= calEnd) {
    days.push(day);
    day = addDays(day, 1);
  }
  const weekDays = getWeekDays(state.language);

  const eventsForDay = useMemo(() => {
    if (!selectedDay) return [];
    const dateStr = format(selectedDay, 'yyyy-MM-dd');
    return eventsList.filter((e) => e.date === dateStr);
  }, [selectedDay, eventsList]);

  const handleDayClick = (d) => {
    setSelectedDay(d);
    setShowForm(false);
  };

  const handleAddEvent = async () => {
    if (!eventTitle.trim()) return;
    await addEvent({
      title: eventTitle,
      type: eventType,
      time: eventTime,
      description: eventDesc,
    });
    setEventTitle('');
    setEventDesc('');
    setEventType('training');
    setEventTime('19:00');
    setShowForm(false);
  };

  const handleDeleteEvent = async (eventId) => {
    const key = format(selectedDay, 'yyyy-MM-dd');
    const idx = eventsForDay.findIndex((e) => e.id === eventId);
    if (idx >= 0) {
      await deleteCalendarEvent(key, idx);
    }
  };

  const typeLabels = {
    training: t('Тренировка', 'Training'),
    food: t('Питание', 'Meal'),
    recovery: t('Восстановление', 'Recovery'),
    meeting: t('Встреча', 'Meeting'),
    note: t('Заметка', 'Note'),
  };
  const typeColors = {
    training: '#6fae20',
    food: '#f5d45d',
    recovery: '#ff6375',
    meeting: '#a78bfa',
    note: '#19d8da',
  };

  return (
    <section className='calendar'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Календарь', 'Calendar')}</div>
          <div className='panel-sub'>
            {format(monthStart, 'MMMM yyyy', state.language)}
          </div>
        </div>
        <div className='panel-body'>
          <div className='cal-grid'>
            {weekDays.map((d) => (
              <div key={d} className='cal-cell cal-head'>
                {d}
              </div>
            ))}
            {days.map((d) => {
              const inMonth = isSameMonth(d, monthStart);
              const isSelected = selectedDay && isSameDay(d, selectedDay);
              const dayEvents = eventsList.filter((e) => e.date === format(d, 'yyyy-MM-dd'));
              return (
                <button
                  key={d.getTime()}
                  type="button"
                  className={
                    'cal-cell' +
                    (inMonth ? '' : ' muted') +
                    (isSelected ? ' selected' : '') +
                    (dayEvents.length > 0 ? ' has-events' : '')
                  }
                  onClick={() => handleDayClick(d)}
                >
                  {format(d, 'd', state.language)}
                  {dayEvents.length > 0 && (
                    <span className='cal-dots'>
                      {dayEvents.slice(0, 3).map((ev) => (
                        <span
                          key={ev.id}
                          className='cal-dot'
                          style={{ background: typeColors[ev.type] || '#19d8da' }}
                        />
                      ))}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
          <div className='cal-actions'>
            {selectedDay ? (
              <>
                <div className='muted'>
                  {format(selectedDay, 'd MMMM yyyy', state.language)}
                </div>
                <button
                  className='primary'
                  onClick={() => setShowForm(!showForm)}
                >
                  {showForm ? t('Отмена', 'Cancel') : t('Добавить событие', 'Add event')}
                </button>
              </>
            ) : (
              <div className='muted'>{t('Выберите день', 'Select a day')}</div>
            )}
          </div>
          {showForm && selectedDay && (
            <div className='event-form'>
              <div className='field'>
                <label>{t('Название', 'Title')}</label>
                <input
                  value={eventTitle}
                  onChange={(e) => setEventTitle(e.target.value)}
                  placeholder={t('Название события', 'Event title')}
                />
              </div>
              <div className='field'>
                <label>{t('Тип', 'Type')}</label>
                <select value={eventType} onChange={(e) => setEventType(e.target.value)}>
                  {Object.entries(typeLabels).map(([k, v]) => (
                    <option key={k} value={k}>{v}</option>
                  ))}
                </select>
              </div>
              <div className='field'>
                <label>{t('Время', 'Time')}</label>
                <input
                  type='time'
                  value={eventTime}
                  onChange={(e) => setEventTime(e.target.value)}
                />
              </div>
              <div className='field'>
                <label>{t('Описание', 'Description')}</label>
                <input
                  value={eventDesc}
                  onChange={(e) => setEventDesc(e.target.value)}
                  placeholder={t('Описание события', 'Event description')}
                />
              </div>
              <button className='primary full' onClick={handleAddEvent}>
                {t('Добавить', 'Add')}
              </button>
            </div>
          )}
          {selectedDay && eventsForDay.length > 0 && (
            <div className='events-list'>
              {eventsForDay.map((ev, i) => (
                <div key={ev.id || i} className='list-row event-row'>
                  <div>
                    <div className='list-title'>{ev.title}</div>
                    <div className='muted'>
                      {typeLabels[ev.type] || ev.type} · {ev.time || ''} · {ev.description || ''}
                    </div>
                  </div>
                  <button
                    className='danger'
                    onClick={() => handleDeleteEvent(ev.id)}
                  >
                    {t('Удалить', 'Delete')}
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
