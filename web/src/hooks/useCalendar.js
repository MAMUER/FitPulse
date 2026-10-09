import { useCallback } from 'react';
import * as api from '../services/api';
import { getDateKey, isDateAllowed } from '../utils/helpers';
import { t } from '../utils/i18n';

export function useCalendar({
  state,
  update,
  notify,
  getMonthName: getMonthNameFn,
  language,
}) {
  const changeCalendar = useCallback(
    (direction) => {
      let m = state.calendarMonthIndex + Number(direction || 0);
      let y = state.calendarYear;
      while (m < 0) {
        m += 12;
        y--;
      }
      while (m > 11) {
        m -= 12;
        y++;
      }
      if (!isDateAllowed(y, m)) {
        notify('Нельзя выбрать дату раньше августа 2025 года');
        return;
      }
      update({
        calendarMonthIndex: m,
        calendarYear: y,
        selectedCalendarDay: null,
      });
    },
    [state.calendarMonthIndex, state.calendarYear, update, notify]
  );

  const quickAddEvent = useCallback(
    (type) => {
      const d = new Date();
      const colors = {
        gym: '#6fae20',
        pool: '#19d8da',
        recovery: '#ff6375',
        work: '#a78bfa',
        study: '#f5d45d',
        note: '#a78bfa',
      };
      const labels = {
        gym: 'Тренировка в зале',
        pool: 'Бассейн',
        recovery: 'Восстановление',
        work: 'Работа',
        study: 'Учёба',
        note: 'Заметка',
      };
      const key = getDateKey(d.getFullYear(), d.getMonth(), d.getDate());
      const newEvents = { ...state.calendarEvents };
      if (!newEvents[key]) newEvents[key] = [];
      newEvents[key] = [
        ...newEvents[key],
        {
          title: labels[type] || type,
          type,
          date: `${d.getDate()} ${getMonthNameFn(d.getMonth(), language)}`,
          time: '19:00',
          color: colors[type] || '#19d8da',
        },
      ];
      update({ calendarEvents: newEvents });
    },
    [state.calendarEvents, language, update, getMonthNameFn]
  );

  const openCalendarPage = useCallback(
    (resetToToday = true) => {
      if (resetToToday) {
        const now = new Date();
        update({
          calendarMonthIndex: now.getMonth(),
          calendarYear: now.getFullYear(),
        });
      }
    },
    [update]
  );

  const selectCalendarDay = useCallback(
    (day) => {
      update({
        selectedCalendarDay: day,
        dateModalDate: day,
        showDateModal: true,
      });
    },
    [update]
  );

  const closeDateModal = useCallback(() => {
    update({ showDateModal: false, dateModalDate: null });
  }, [update]);

  const addEventToDay = useCallback(
    (day) => {
      update({
        selectedCalendarDay: day,
        dateModalDate: day,
        showDateModal: true,
      });
    },
    [update]
  );

  const addEvent = useCallback(
    async (eventData = {}) => {
      const day = state.selectedCalendarDay || new Date().getDate();
      const type =
        eventData.type ||
        document.getElementById('eventType')?.value ||
        'training';
      const title =
        eventData.title ||
        document.getElementById('eventTitle')?.value.trim() ||
        t('Новое событие', 'New event');
      const time =
        eventData.time ||
        document.getElementById('eventTime')?.value ||
        '19:00';
      const description =
        eventData.description ||
        document.getElementById('eventDescription')?.value.trim() ||
        '';
      const dateStr = `${state.calendarYear}-${String(state.calendarMonthIndex + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
      try {
        const data = await api.createEvent({
          title,
          date: dateStr,
          type,
          description,
        });
        if (data?.event) {
          const newEvents = { ...state.calendarEvents };
          const dateLabel = `${day} ${getMonthNameFn(state.calendarMonthIndex, language)}`;
          if (!newEvents[dateLabel]) newEvents[dateLabel] = [];
          newEvents[dateLabel] = [
            ...newEvents[dateLabel],
            {
              id: data.event.id,
              title: data.event.title,
              type: data.event.type,
              info: description || type,
              time,
              date: dateLabel,
              color:
                {
                  training: '#6fae20',
                  food: '#f5d45d',
                  recovery: '#ff6375',
                  meeting: '#a78bfa',
                  note: '#19d8da',
                }[type] || '#19d8da',
            },
          ];
          const newList = [
            ...(state.calendarEventsList || []),
            { ...data.event, dateLabel },
          ];
          update({
            calendarEvents: newEvents,
            calendarEventsList: newList,
            showDateModal: false,
            dateModalDate: null,
          });
        }
      } catch {
        notify('Ошибка добавления события');
      }
    },
    [
      state.selectedCalendarDay,
      state.calendarMonthIndex,
      state.calendarYear,
      state.calendarEvents,
      language,
      update,
      notify,
      getMonthNameFn,
      state.calendarEventsList,
    ]
  );

  const updateEvent = useCallback(
    async (eventId, title, date, type, description) => {
      try {
        const data = await api.updateEvent(
          eventId,
          title,
          date,
          type,
          description
        );
        if (data?.event) {
          const newList = (state.calendarEventsList || []).map((e) =>
            e.id === eventId ? { ...e, ...data.event } : e
          );
          update({ calendarEventsList: newList });
          notify('Событие обновлено');
        }
      } catch {
        notify('Ошибка обновления события');
      }
    },
    [state.calendarEventsList, update, notify]
  );

  const deleteCalendarEvent = useCallback(
    async (keyOrEventId, index) => {
      let eventId = keyOrEventId;

      if (typeof index === 'number' && typeof keyOrEventId === 'string') {
        const list = state.calendarEvents[keyOrEventId];
        if (!list || index < 0 || index >= list.length) return;
        eventId = list[index].id;
      }

      try {
        await api.deleteCalendarEvent(eventId);
        const newList = (state.calendarEventsList || []).filter(
          (e) => e.id !== eventId
        );
        const newEvents = { ...state.calendarEvents };
        for (const key of Object.keys(newEvents)) {
          newEvents[key] = newEvents[key].filter((e) => e.id !== eventId);
          if (!newEvents[key].length) delete newEvents[key];
        }
        update({
          calendarEvents: newEvents,
          calendarEventsList: newList,
          showDateModal: false,
          dateModalDate: null,
        });
      } catch {
        notify('Ошибка удаления события');
      }
    },
    [state.calendarEvents, state.calendarEventsList, update, notify]
  );

  return {
    changeCalendar,
    quickAddEvent,
    openCalendarPage,
    selectCalendarDay,
    closeDateModal,
    addEventToDay,
    addEvent,
    updateEvent,
    deleteCalendarEvent,
  };
}
