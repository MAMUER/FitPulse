import { useState, useEffect } from 'react';
import {
  listHealthConditions,
  upsertHealthCondition,
  deleteHealthCondition,
  listBodyComposition,
  createBodyComposition,
  listMenstrualCycles,
  createMenstrualCycle,
  deleteMenstrualCycle,
  syncFlo,
  syncOKOK,
} from '../../utils/api';
import './Health.css';

const TYPE_LABELS = {
  allergy: 'Аллергия',
  disease: 'Заболевание',
  disability: 'Инвалидность',
  other: 'Другое',
};

export default function Health() {
  const [conditions, setConditions] = useState([]);
  const [bodyComposition, setBodyComposition] = useState([]);
  const [cycles, setCycles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [toast, setToast] = useState('');

  useEffect(() => {
    loadAll();
  }, []);

  const loadAll = async () => {
    try {
      const [c, b, m] = await Promise.allSettled([
        listHealthConditions(),
        listBodyComposition(),
        listMenstrualCycles(),
      ]);
      if (c.status === 'fulfilled') setConditions(c.value || []);
      if (b.status === 'fulfilled') setBodyComposition(b.value || []);
      if (m.status === 'fulfilled') setCycles(m.value || []);
    } catch (e) {
      console.error('Failed to load health data:', e);
    } finally {
      setLoading(false);
    }
  };

  const showToast = (msg) => {
    setToast(msg);
    setTimeout(() => setToast(''), 3000);
  };

  const handleAddCondition = async () => {
    const name = prompt('Название состояния:');
    if (!name) return;
    const type = prompt('Тип (allergy/disease/disability/other):') || 'other';
    const severity = prompt('Серьёзность:') || '';
    const notes = prompt('Заметки:') || '';
    try {
      await upsertHealthCondition({ condition_type: type, condition_name: name, severity, notes, is_active: true });
      showToast('Состояние добавлено');
      loadAll();
    } catch (e) {
      showToast('Ошибка: ' + e.message);
    }
  };

  const handleAddBodyComposition = async () => {
    const weight_kg = prompt('Вес, кг:');
    if (!weight_kg) return;
    const height_cm = prompt('Рост, см:');
    const body_fat_percentage = prompt('Процент жира:');
    const muscle_mass_percentage = prompt('Процент мышц:');
    try {
      await createBodyComposition({ weight_kg: parseFloat(weight_kg), height_cm: height_cm ? parseFloat(height_cm) : null, body_fat_percentage: body_fat_percentage ? parseFloat(body_fat_percentage) : null, muscle_mass_percentage: muscle_mass_percentage ? parseFloat(muscle_mass_percentage) : null });
      showToast('Запись добавлена');
      loadAll();
    } catch (e) {
      showToast('Ошибка: ' + e.message);
    }
  };

  const handleAddMenstrualCycle = async () => {
    const cycle_start_date = prompt('Дата начала цикла (YYYY-MM-DD):');
    if (!cycle_start_date) return;
    const cycle_end_date = prompt('Дата окончания цикла (YYYY-MM-DD, необязательно):') || '';
    const flow_intensity = prompt('Интенсивность (light/medium/heavy):') || 'medium';
    const symptoms = prompt('Симптомы (через запятую):') || '';
    const moods = prompt('Настроения (через запятую):') || '';
    const notes = prompt('Заметки:') || '';
    try {
      await createMenstrualCycle({
        cycle_start_date,
        cycle_end_date: cycle_end_date || null,
        flow_intensity,
        symptoms: symptoms.split(',').map(s => s.trim()).filter(Boolean),
        moods: moods.split(',').map(s => s.trim()).filter(Boolean),
        notes,
      });
      showToast('Цикл добавлен');
      loadAll();
    } catch (e) {
      showToast('Ошибка: ' + e.message);
    }
  };

  const handleSync = async (fn, name) => {
    const access_token = prompt(`Токен доступа для ${name}:`);
    if (!access_token) return;
    const refresh_token = prompt(`Refresh токен для ${name}:`);
    if (!refresh_token) return;
    try {
      await fn(access_token, refresh_token);
      showToast(`Синхронизация с ${name} выполнена`);
    } catch (e) {
      showToast('Ошибка: ' + e.message);
    }
  };

  const handleDeleteCondition = async (id) => {
    if (!confirm('Удалить это состояние?')) return;
    try {
      await deleteHealthCondition(id);
      showToast('Удалено');
      loadAll();
    } catch (e) {
      showToast('Ошибка: ' + e.message);
    }
  };

  const handleDeleteCycle = async (id) => {
    if (!confirm('Удалить эту запись цикла?')) return;
    try {
      await deleteMenstrualCycle(id);
      showToast('Удалено');
      loadAll();
    } catch (e) {
      showToast('Ошибка: ' + e.message);
    }
  };

  if (loading) return <div className="loading">Загрузка данных здоровья...</div>;

  return (
    <div className="view active">
      {toast && <div className="toast success">{toast}</div>}

      <section className="health-section">
        <h3>Заболевания и состояния</h3>
        <div className="health-actions">
          <button className="btn-secondary" onClick={handleAddCondition}>Добавить</button>
        </div>
        <div id="conditionsList" className="health-list">
          {conditions.length === 0 ? (
            <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>Нет добавленных состояний</p>
          ) : (
            conditions.map(c => (
              <div key={c.condition_id} className="health-card">
                <div className="health-card-header">
                  <div className="card-value" style={{ fontSize: 16 }}>{c.condition_name || 'Без названия'}</div>
                  <span className="badge">{TYPE_LABELS[c.condition_type] || c.condition_type || 'Другое'}</span>
                </div>
                <div className="health-card-meta">
                  {c.severity && <span>Серьёзность: {c.severity}</span>}
                  {c.is_active !== undefined && <span>{c.is_active ? 'Активно' : 'Неактивно'}</span>}
                </div>
                {c.notes && <div className="health-card-notes">{c.notes}</div>}
                <div className="health-card-actions">
                  <button className="btn-danger-ghost" onClick={() => handleDeleteCondition(c.condition_id)}>Удалить</button>
                </div>
              </div>
            ))
          )}
        </div>
      </section>

      <section className="health-section">
        <h3>Состав тела</h3>
        <div className="health-actions">
          <button className="btn-secondary" onClick={handleAddBodyComposition}>Добавить</button>
        </div>
        <div id="bodyCompositionList" className="health-list">
          {bodyComposition.length === 0 ? (
            <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>Нет записей</p>
          ) : (
            bodyComposition.map(b => (
              <div key={b.body_composition_id || b.id || JSON.stringify(b)} className="health-card">
                <div className="health-card-header">
                  <div className="card-value" style={{ fontSize: 16 }}>Состав тела</div>
                  <span className="badge">{new Date(b.recorded_at || b.created_at).toLocaleDateString('ru-RU')}</span>
                </div>
                <div className="health-card-meta">
                  {b.weight_kg && <span>Вес: {b.weight_kg} кг</span>}
                  {b.height_cm && <span>Рост: {b.height_cm} см</span>}
                  {b.body_fat_percentage && <span>Жир: {b.body_fat_percentage}%</span>}
                  {b.muscle_mass_percentage && <span>Мышцы: {b.muscle_mass_percentage}%</span>}
                </div>
              </div>
            ))
          )}
        </div>
      </section>

      <section className="health-section">
        <h3>Менструальный цикл</h3>
        <div className="health-actions">
          <button className="btn-secondary" onClick={handleAddMenstrualCycle}>Добавить</button>
        </div>
        <div className="sync-grid">
          <button onClick={() => handleSync(syncFlo, 'Flo')}>Синхронизировать Flo</button>
          <button onClick={() => handleSync(syncOKOK, 'OKOK')}>Синхронизировать OKOK</button>
        </div>
        <div id="menstrualCyclesList" className="health-list">
          {cycles.length === 0 ? (
            <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>Нет записей</p>
          ) : (
            cycles.map(c => (
              <div key={c.menstrual_cycle_id || c.id || JSON.stringify(c)} className="health-card">
                <div className="health-card-header">
                  <div className="card-value" style={{ fontSize: 16 }}>Цикл</div>
                  <span className="badge">{c.flow_intensity || '—'}</span>
                </div>
                <div className="health-card-meta">
                  {c.cycle_start_date && <span>Начало: {c.cycle_start_date}</span>}
                  {c.cycle_end_date && <span>Окончание: {c.cycle_end_date}</span>}
                </div>
                {c.symptoms && c.symptoms.length > 0 && (
                  <div className="health-card-notes">Симптомы: {c.symptoms.join(', ')}</div>
                )}
                {c.moods && c.moods.length > 0 && (
                  <div className="health-card-notes">Настроения: {c.moods.join(', ')}</div>
                )}
                {c.notes && <div className="health-card-notes">{c.notes}</div>}
                <div className="health-card-actions">
                  <button className="btn-danger-ghost" onClick={() => handleDeleteCycle(c.menstrual_cycle_id || c.id)}>Удалить</button>
                </div>
              </div>
            ))
          )}
        </div>
      </section>
    </div>
  );
}
