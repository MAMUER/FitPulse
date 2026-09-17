const MONTHS_RU = [
  'Январь',
  'Февраль',
  'Март',
  'Апрель',
  'Май',
  'Июнь',
  'Июль',
  'Август',
  'Сентябрь',
  'Октябрь',
  'Ноябрь',
  'Декабрь',
];
const MONTHS_EN = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
];
const MIN_YEAR = 2025;
const MIN_MONTH = 7;
const WEEK_RU = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
const WEEK_EN = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

export function getMonthName(index, language = 'ru') {
  const months = language === 'en' ? MONTHS_EN : MONTHS_RU;
  return months[index] || '';
}

export function getWeekDays(language = 'ru') {
  return language === 'en' ? WEEK_EN : WEEK_RU;
}

export function getDateKey(year, monthIndex, day) {
  return `${year}-${String(monthIndex + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
}

export function getEventsForDate(calendarEvents, year, monthIndex, day) {
  const key = getDateKey(year, monthIndex, day);
  return calendarEvents[key] || [];
}

export function getEventsForMonth(calendarEvents, year, monthIndex) {
  const result = {};
  for (let day = 1; day <= 31; day++) {
    const key = getDateKey(year, monthIndex, day);
    if (calendarEvents[key]) {
      result[day] = calendarEvents[key];
    }
  }
  return result;
}

export function addEventToDate(
  calendarEvents,
  year,
  monthIndex,
  day,
  eventData
) {
  const key = getDateKey(year, monthIndex, day);
  if (!calendarEvents[key]) calendarEvents[key] = [];
  calendarEvents[key].push(eventData);
}

export function isDateAllowed(year, monthIndex) {
  if (year < MIN_YEAR) return false;
  if (year === MIN_YEAR && monthIndex < MIN_MONTH) return false;
  return true;
}

export function waterDateKey() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

export function safeText(v) {
  return String(v ?? '').replace(
    /[&<>"']/g,
    (ch) =>
      ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[
        ch
      ]
  );
}

export function formatRelativeDate(offsetDays, language = 'ru') {
  const d = new Date();
  d.setDate(d.getDate() + offsetDays);
  const weekday = d.toLocaleDateString(language === 'en' ? 'en-US' : 'ru-RU', {
    weekday: 'long',
  });
  const month = getMonthName(d.getMonth(), language);
  const day = d.getDate();
  const today = new Date();
  const isToday = offsetDays === 0 && d.getDate() === today.getDate();
  const tomorrow = new Date(today);
  tomorrow.setDate(tomorrow.getDate() + 1);
  const isTomorrow =
    offsetDays === 1 &&
    d.getDate() === tomorrow.getDate() &&
    d.getMonth() === tomorrow.getMonth();

  if (isToday) return language === 'en' ? 'Today' : 'Сегодня';
  if (isTomorrow) return language === 'en' ? 'Tomorrow' : 'Завтра';
  return `${weekday} · ${day} ${month}`;
}

export function startOfMonth(date) {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

export function endOfMonth(date) {
  return new Date(date.getFullYear(), date.getMonth() + 1, 0);
}

export function startOfWeek(date) {
  const d = new Date(date);
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1);
  return new Date(d.setDate(diff));
}

export function endOfWeek(date) {
  const d = new Date(date);
  const day = d.getDay();
  const diff = d.getDate() + (day === 0 ? 0 : 7 - day);
  return new Date(d.setDate(diff));
}

export function addDays(date, days) {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

export function isSameDay(a, b) {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

export function isSameMonth(a, b) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth();
}

export function format(date, fmt, locale = 'ru') {
  const map = {
    d: String(date.getDate()),
    M: String(date.getMonth() + 1),
    yyyy: String(date.getFullYear()),
    MMMM: getMonthName(date.getMonth(), locale),
  };
  let out = fmt;
  for (const [k, v] of Object.entries(map)) {
    out = out.replace(k, v);
  }
  return out;
}

export { MIN_MONTH, MIN_YEAR, MONTHS_EN, MONTHS_RU };
