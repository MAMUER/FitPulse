import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';
import Panel from '../components/Panel';
import ListItem from '../components/ListItem';
import EmptyState from '../components/EmptyState';

export default function Menstrual() {
  const { state, t, notify, loadMenstrualCycles, createMenstrualCycle, updateMenstrualCycle, deleteMenstrualCycle } = useApp();
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({ start_date: '', end_date: '', flow_intensity: 'moderate', symptoms: '', moods: '', notes: '' });
  const [editingId, setEditingId] = useState(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void loadCycles();
  }, [loadMenstrualCycles]);

  const loadCycles = async () => {
    setLoading(true);
    try {
      await loadMenstrualCycles();
    } catch {
      notify('Ошибка загрузки цикла');
    } finally {
      setLoading(false);
    }
  };

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
      symptoms: form.symptoms ? form.symptoms.split(',').map((s) => s.trim()).filter(Boolean) : [],
      moods: form.moods ? form.moods.split(',').map((s) => s.trim()).filter(Boolean) : [],
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
      setForm({ start_date: '', end_date: '', flow_intensity: 'moderate', symptoms: '', moods: '', notes: '' });
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

  const cycles = state.menstrualCycles || [];

  const nextPeriodEstimate = () => {
    if (cycles.length < 2) return null;
    const sorted = cycles
      .filter((c) => c.start_date)
      .sort((a, b) => new Date(b.start_date) - new Date(a.start_date));
    if (sorted.length < 2) return null;
    const last = new Date(sorted[0].start_date);
    const prev = new Date(sorted[1].start_date);
    const diff = last - prev;
    if (diff <= 0) return null;
    const next = new Date(last.getTime() + diff);
    return next.toISOString().slice(0, 10);
  };

  const renderMenstrualList = () => {
    if (loading) {
      return <EmptyState text={t('Загрузка...', 'Loading...')} />;
    }
    if (cycles.length === 0) {
      return <EmptyState text={t('Нет записей', 'No records')} />;
    }
    return cycles
      .slice()
      .sort((a, b) => new Date(b.start_date) - new Date(a.start_date))
      .map((cycle) => (
        <ListItem
          key={cycle.id}
          title={`${cycle.start_date || '—'} ${cycle.end_date ? '→ ' + cycle.end_date : ''}`}
          subtitle={`${t('Интенсивность', 'Flow')}: ${cycle.flow_intensity || '—'}${cycle.notes ? ' · ' + cycle.notes : ''}`}
          right={
            <div className='row-buttons'>
              <button type='button' className='secondary' onClick={() => handleEdit(cycle)}>
                {t('Изменить', 'Edit')}
              </button>
              <button type='button' className='danger' onClick={() => handleDelete(cycle.id)}>
                {t('Удалить', 'Delete')}
              </button>
            </div>
          }
        />
      ));
  };

  return (
    <section className='menstrual'>
      <Panel
        title={t('Менструальный календарь', 'Menstrual calendar')}
        subtitle={t('Ручной ввод и прогнозы', 'Manual entry and predictions')}
      >
        <div className='panel-body'>
          <p className='medical-disclaimer'>
            {t(
              'Это не медицинский совет. Данные носят ознакомительный характер.',
              'This is not medical advice. Data is for informational purposes only.'
            )}
          </p>

          {nextPeriodEstimate && (
            <div className='metric-row'>
              <div className='metric-card'>
                <div className='metric-label'>{t('Следующий период примерно', 'Next period approx')}</div>
                <div className='metric-value'>{nextPeriodEstimate}</div>
              </div>
            </div>
          )}

          <div className='form'>
            <div className='field'>
              <label htmlFor='start_date'>{t('Дата начала', 'Start date')}</label>
              <input
                id='start_date'
                type='date'
                value={form.start_date}
                onChange={(e) => setForm((f) => ({ ...f, start_date: e.target.value }))}
              />
            </div>
            <div className='field'>
              <label htmlFor='end_date'>{t('Дата окончания', 'End date')}</label>
              <input
                id='end_date'
                type='date'
                value={form.end_date}
                onChange={(e) => setForm((e) => setForm((f) => ({ ...f, end_date: e.target.value })))}
              />
            </div>
            <div className='field'>
              <label htmlFor='flow_intensity'>{t('Интенсивность', 'Flow intensity')}</label>
              <select
                id='flow_intensity'
                value={form.flow_intensity}
                onChange={(e) => setForm((f) => ({ ...f, flow_intensity: e.target.value }))}
              >
                <option value='light'>{t('Легкая', 'Light')}</option>
                <option value='moderate'>{t('Средняя', 'Moderate')}</option>
                <option value='heavy'>{t('Сильная', 'Heavy')}</option>
                <option value='spotting'>{t('Кровянистые выделения', 'Spotting')}</option>
              </select>
            </div>
            <div className='field'>
              <label htmlFor='symptoms'>{t('Симптомы (через запятую)', 'Symptoms (comma separated)')}</label>
              <input
                id='symptoms'
                type='text'
                value={form.symptoms}
                onChange={(e) => setForm((f) => ({ ...f, symptoms: e.target.value }))}
              />
            </div>
            <div className='field'>
              <label htmlFor='moods'>{t('Настроение (через запятую)', 'Moods (comma separated)')}</label>
              <input
                id='moods'
                type='text'
                value={form.moods}
                onChange={(e) => setForm((f) => ({ ...f, moods: e.target.value }))}
              />
            </div>
            <div className='field'>
              <label htmlFor='notes'>{t('Заметки', 'Notes')}</label>
              <textarea
                id='notes'
                value={form.notes}
                onChange={(e) => setForm((f) => ({ ...f, notes: e.target.value }))}
              />
            </div>
            <button type='button' className='primary' onClick={handleSubmit} disabled={saving}>
              {editingId ? t('Сохранить', 'Save') : t('Добавить', 'Add')}
            </button>
            {editingId && (
              <button type='button' className='secondary' onClick={() => { setEditingId(null); setForm({ start_date: '', end_date: '', flow_intensity: 'moderate', symptoms: '', moods: '', notes: '' }); }}>
                {t('Отмена', 'Cancel')}
              </button>
            )}
          </div>

          <div className='list'>
            {renderMenstrualList()}
          </div>
        </div>
      </Panel>
    </section>
  );
}
