import { useCallback, useEffect, useMemo, useState } from 'react';
import EmptyState from '../components/EmptyState';
import ListItem from '../components/ListItem';
import Panel from '../components/Panel';
import { useApp } from '../contexts/AppContext';

const DAY_MS = 86400000;

function dateKey(date) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

function startOfDay(date) {
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);
  return d;
}

export default function Menstrual() {
  const {
    state,
    t,
    canEdit,
    notify,
    loadMenstrualCycles,
    createMenstrualCycle,
    updateMenstrualCycle,
    deleteMenstrualCycle,
  } = useApp();
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({
    start_date: '',
    end_date: '',
    flow_intensity: 'moderate',
    symptoms: '',
    moods: '',
    notes: '',
  });
  const [editingId, setEditingId] = useState(null);
  const [saving, setSaving] = useState(false);
  const [monthOffset, setMonthOffset] = useState(0);

  const loadCycles = useCallback(async () => {
    setLoading(true);
    try {
      await loadMenstrualCycles?.();
    } catch {
      notify('Ошибка загрузки цикла');
    } finally {
      setLoading(false);
    }
  }, [loadMenstrualCycles, notify]);

  useEffect(() => {
    void loadCycles();
  }, [loadCycles]);

  const cycles = state.menstrualCycles || [];

  // ---- Прогнозы (в рамках фитнес-приложения) ----
  const sortedCycles = useMemo(
    () =>
      cycles
        .filter((c) => c.start_date)
        .slice()
        .sort((a, b) => new Date(b.start_date) - new Date(a.start_date)),
    [cycles]
  );

  // Полный список для отображения (включая записи без даты начала — их тоже
  // можно редактировать/удалить).
  const displayCycles = useMemo(
    () =>
      cycles.slice().sort((a, b) => {
        const at = a.start_date ? new Date(a.start_date).getTime() : 0;
        const bt = b.start_date ? new Date(b.start_date).getTime() : 0;
        return bt - at;
      }),
    [cycles]
  );

  const avgCycle = useMemo(() => {
    if (sortedCycles.length < 2) return null;
    const diffs = [];
    for (let i = 0; i < sortedCycles.length - 1; i += 1) {
      const diff =
        (new Date(sortedCycles[i].start_date) -
          new Date(sortedCycles[i + 1].start_date)) /
        DAY_MS;
      if (diff > 15 && diff < 60) diffs.push(Math.round(diff));
    }
    if (!diffs.length) return null;
    return Math.round(diffs.reduce((a, b) => a + b, 0) / diffs.length);
  }, [sortedCycles]);

  const avgPeriod = useMemo(() => {
    const lengths = sortedCycles
      .filter((c) => c.start_date && c.end_date)
      .map(
        (c) =>
          Math.round((new Date(c.end_date) - new Date(c.start_date)) / DAY_MS) +
          1
      )
      .filter((n) => n > 0 && n <= 14);
    if (!lengths.length) return null;
    return Math.round(lengths.reduce((a, b) => a + b, 0) / lengths.length);
  }, [sortedCycles]);

  const nextPeriod = useMemo(() => {
    if (!sortedCycles.length) return null;
    const last = startOfDay(new Date(sortedCycles[0].start_date));
    const cycleLen = avgCycle || 28;
    return new Date(last.getTime() + cycleLen * DAY_MS);
  }, [sortedCycles, avgCycle]);

  // Дни менструации (реальные) и прогноз.
  const periodDays = useMemo(() => {
    const days = new Set();
    for (const c of cycles) {
      if (!c.start_date) continue;
      const start = startOfDay(new Date(c.start_date));
      const end = c.end_date
        ? startOfDay(new Date(c.end_date))
        : new Date(start.getTime() + 4 * DAY_MS);
      const totalDays =
        Math.round((end.getTime() - start.getTime()) / DAY_MS) + 1;
      for (let i = 0; i < totalDays; i += 1) {
        days.add(dateKey(new Date(start.getTime() + i * DAY_MS)));
      }
    }
    return days;
  }, [cycles]);

  const predictedDays = useMemo(() => {
    const days = new Set();
    if (!nextPeriod) return days;
    const start = startOfDay(nextPeriod);
    const len = avgPeriod || 5;
    for (let i = 0; i < len; i += 1) {
      const d = new Date(start.getTime() + i * DAY_MS);
      days.add(dateKey(d));
    }
    return days;
  }, [nextPeriod, avgPeriod]);

  const todayKey = dateKey(new Date());

  // ---- Календарь текущего месяца ----
  const viewDate = new Date(
    new Date().getFullYear(),
    new Date().getMonth() + monthOffset,
    1
  );
  const year = viewDate.getFullYear();
  const monthIndex = viewDate.getMonth();
  const daysInMonth = new Date(year, monthIndex + 1, 0).getDate();
  const startOffset = (new Date(year, monthIndex, 1).getDay() + 6) % 7;
  const monthLabel = viewDate.toLocaleDateString(
    state.language === 'en' ? 'en-US' : 'ru-RU',
    { month: 'long', year: 'numeric' }
  );
  const weekdays = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];

  const handleSubmit = async () => {
    if (!form.start_date) {
      notify('Укажите дату начала цикла');
      return;
    }
    setSaving(true);
    const payload = {
      start_date: form.start_date,
      end_date: form.end_date || undefined,
      flow_intensity: form.flow_intensity,
      symptoms: form.symptoms
        ? form.symptoms
            .split(',')
            .map((s) => s.trim())
            .filter(Boolean)
        : [],
      moods: form.moods
        ? form.moods
            .split(',')
            .map((s) => s.trim())
            .filter(Boolean)
        : [],
      notes: form.notes || undefined,
    };
    try {
      if (editingId) {
        await updateMenstrualCycle(editingId, payload);
        notify('Цикл обновлён');
      } else {
        await createMenstrualCycle(payload);
        notify('Цикл добавлен');
      }
      setForm({
        start_date: '',
        end_date: '',
        flow_intensity: 'moderate',
        symptoms: '',
        moods: '',
        notes: '',
      });
      setEditingId(null);
    } catch {
      notify('Ошибка сохранения цикла');
    } finally {
      setSaving(false);
    }
  };

  const handleEdit = (cycle) => {
    setEditingId(cycle.id);
    setForm({
      start_date: cycle.start_date || '',
      end_date: cycle.end_date || '',
      flow_intensity: cycle.flow_intensity || 'moderate',
      symptoms: Array.isArray(cycle.symptoms) ? cycle.symptoms.join(', ') : '',
      moods: Array.isArray(cycle.moods) ? cycle.moods.join(', ') : '',
      notes: cycle.notes || '',
    });
  };

  const handleDelete = async (cycleId) => {
    try {
      await deleteMenstrualCycle(cycleId);
      notify('Цикл удалён');
    } catch {
      notify('Ошибка удаления цикла');
    }
  };

  const renderMenstrualList = () => {
    if (loading) {
      return <EmptyState text={t('Загрузка...', 'Loading...')} />;
    }
    if (cycles.length === 0) {
      return <EmptyState text={t('Нет записей', 'No records')} />;
    }
    return displayCycles.map((cycle) => {
      const endDatePart = cycle.end_date ? `→ ${cycle.end_date}` : '';
      const notesPart = cycle.notes ? ` · ${cycle.notes}` : '';
      return (
        <ListItem
          key={cycle.id}
          title={`${cycle.start_date || '—'} ${endDatePart}`}
          subtitle={`${t('Интенсивность', 'Flow')}: ${cycle.flow_intensity || '—'}${notesPart}`}
          right={
            <div className='row-buttons'>
              <button
                type='button'
                className='secondary'
                onClick={() => handleEdit(cycle)}
              >
                {t('Изменить', 'Edit')}
              </button>
              <button
                type='button'
                className='danger'
                onClick={() => handleDelete(cycle.id)}
              >
                {t('Удалить', 'Delete')}
              </button>
            </div>
          }
        />
      );
    });
  };

  return (
    <section className='menstrual'>
      <Panel
        title={t('Менструальный календарь', 'Menstrual calendar')}
        subtitle={t(
          'Календарь, прогнозы и дневник',
          'Calendar, predictions and diary'
        )}
      >
        <div className='panel-body'>
          <p className='medical-disclaimer'>
            {t(
              'Это не медицинский совет. Данные носят ознакомительный характер.',
              'This is not medical advice. Data is for informational purposes only.'
            )}
          </p>

          <div className='metric-row'>
            <div className='metric-card'>
              <div className='metric-label'>
                {t('Средний цикл', 'Avg cycle')}
              </div>
              <div className='metric-value'>
                {avgCycle ? `${avgCycle} ${t('дн.', 'd')}` : '—'}
              </div>
            </div>
            <div className='metric-card'>
              <div className='metric-label'>
                {t('Средние месячные', 'Avg period')}
              </div>
              <div className='metric-value'>
                {avgPeriod ? `${avgPeriod} ${t('дн.', 'd')}` : '—'}
              </div>
            </div>
            <div className='metric-card'>
              <div className='metric-label'>{t('Следующие', 'Next')}</div>
              <div className='metric-value'>
                {nextPeriod ? dateKey(nextPeriod) : '—'}
              </div>
            </div>
          </div>

          <div className='section'>
            <div className='between'>
              <h3>{monthLabel}</h3>
              <div className='row-buttons'>
                <button
                  type='button'
                  className='secondary'
                  aria-label={t('Предыдущий месяц', 'Previous month')}
                  onClick={() => setMonthOffset((o) => o - 1)}
                >
                  ‹
                </button>
                <button
                  type='button'
                  className='secondary'
                  aria-label={t('Следующий месяц', 'Next month')}
                  onClick={() => setMonthOffset((o) => o + 1)}
                >
                  ›
                </button>
              </div>
            </div>
          </div>
          <section className='card'>
            <div className='menstrual-grid weekdays'>
              {weekdays.map((w) => (
                <span className='mweekday' key={w}>
                  {w}
                </span>
              ))}
            </div>
            <div className='menstrual-grid'>
              {Array.from({ length: startOffset }, (_, i) => `empty-${i}`).map(
                (blankKey) => (
                  <span className='mday empty' key={blankKey} />
                )
              )}
              {Array.from({ length: daysInMonth }).map((_, i) => {
                const day = i + 1;
                const key = dateKey(new Date(year, monthIndex, day));
                const isPeriod = periodDays.has(key);
                const isPredicted = predictedDays.has(key);
                const isToday = key === todayKey;
                const cls = [
                  'mday',
                  isPeriod ? 'period' : '',
                  isPredicted ? 'predicted' : '',
                  isToday ? 'today' : '',
                ]
                  .filter(Boolean)
                  .join(' ');
                return (
                  <span className={cls} key={key}>
                    {day}
                  </span>
                );
              })}
            </div>
            <div className='menstrual-legend'>
              <span>
                <i className='mdot period' /> {t('Месячные', 'Period')}
              </span>
              <span>
                <i className='mdot predicted' /> {t('Прогноз', 'Predicted')}
              </span>
            </div>
          </section>

          {canEdit && (
            <div className='form'>
              <div className='field'>
                <label htmlFor='start_date'>
                  {t('Дата начала', 'Start date')}
                </label>
                <input
                  id='start_date'
                  type='date'
                  value={form.start_date}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, start_date: e.target.value }))
                  }
                />
              </div>
              <div className='field'>
                <label htmlFor='end_date'>
                  {t('Дата окончания', 'End date')}
                </label>
                <input
                  id='end_date'
                  type='date'
                  value={form.end_date}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, end_date: e.target.value }))
                  }
                />
              </div>
              <div className='field'>
                <label htmlFor='flow_intensity'>
                  {t('Интенсивность', 'Flow intensity')}
                </label>
                <select
                  id='flow_intensity'
                  value={form.flow_intensity}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, flow_intensity: e.target.value }))
                  }
                >
                  <option value='light'>{t('Легкая', 'Light')}</option>
                  <option value='moderate'>{t('Средняя', 'Moderate')}</option>
                  <option value='heavy'>{t('Сильная', 'Heavy')}</option>
                  <option value='spotting'>
                    {t('Кровянистые выделения', 'Spotting')}
                  </option>
                </select>
              </div>
              <div className='field'>
                <label htmlFor='symptoms'>
                  {t('Симптомы (через запятую)', 'Symptoms (comma separated)')}
                </label>
                <input
                  id='symptoms'
                  type='text'
                  value={form.symptoms}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, symptoms: e.target.value }))
                  }
                />
              </div>
              <div className='field'>
                <label htmlFor='moods'>
                  {t('Настроение (через запятую)', 'Moods (comma separated)')}
                </label>
                <input
                  id='moods'
                  type='text'
                  value={form.moods}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, moods: e.target.value }))
                  }
                />
              </div>
              <div className='field'>
                <label htmlFor='notes'>{t('Заметки', 'Notes')}</label>
                <textarea
                  id='notes'
                  value={form.notes}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, notes: e.target.value }))
                  }
                />
              </div>
              <button
                type='button'
                className='primary'
                onClick={handleSubmit}
                disabled={saving}
              >
                {editingId ? t('Сохранить', 'Save') : t('Добавить', 'Add')}
              </button>
              {editingId && (
                <button
                  type='button'
                  className='secondary'
                  onClick={() => {
                    setEditingId(null);
                    setForm({
                      start_date: '',
                      end_date: '',
                      flow_intensity: 'moderate',
                      symptoms: '',
                      moods: '',
                      notes: '',
                    });
                  }}
                >
                  {t('Отмена', 'Cancel')}
                </button>
              )}
            </div>
          )}

          <div className='list'>{renderMenstrualList()}</div>
        </div>
      </Panel>
    </section>
  );
}
