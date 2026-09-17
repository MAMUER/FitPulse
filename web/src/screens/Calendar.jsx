import { useApp } from '../contexts/AppContext';
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

export default function Calendar() {
  const { state, t, toggleDay } = useApp();
  const monthStart = startOfMonth(state.currentMonth);
  const monthEnd = endOfMonth(state.currentMonth);
  const calStart = startOfWeek(monthStart);
  const calEnd = endOfWeek(monthEnd);
  const days = [];
  let day = calStart;
  while (day <= calEnd) {
    days.push(day);
    day = addDays(day, 1);
  }
  const weekDays = getWeekDays(state.language);
  const selectedCount = (state.selectedDays || []).length;

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
              const active =
                state.workoutStartDate && isSameDay(d, state.workoutStartDate);
              const selected = (state.selectedDays || []).some((sd) =>
                isSameDay(sd, d)
              );
              return (
                <div
                  key={d.getTime()}
                  className={
                    'cal-cell' +
                    (inMonth ? '' : ' muted') +
                    (active ? ' active' : '') +
                    (selected ? ' selected' : '')
                  }
                  onClick={() => toggleDay(d)}
                >
                  {format(d, 'd', state.language)}
                </div>
              );
            })}
          </div>
          <div className='cal-actions'>
            <div className='muted'>
              {t('Выбрано дней', 'Selected days')}: {selectedCount}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
