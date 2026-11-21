import { useEffect, useMemo, useState } from 'react';
import { useApp } from '../contexts/AppContext';

const METRIC_DEFS = [
  {
    key: 'pulse',
    icon: 'fa-heart-pulse',
    unit: 'bpm',
    label: ['Пульс', 'Pulse'],
  },
  {
    key: 'sleep',
    icon: 'fa-moon',
    unit: ['часа', 'hours'],
    label: ['Сон', 'Sleep'],
  },
  { key: 'oxygen', icon: 'fa-lungs', unit: '%', label: ['Кислород', 'Oxygen'] },
  { key: 'hrv', icon: 'fa-wave-square', unit: 'мс', label: ['ВСР', 'HRV'] },
  {
    key: 'stress',
    icon: 'fa-brain',
    unit: ['уровень', 'level'],
    label: ['Стресс', 'Stress'],
  },
];

export default function Body() {
  const {
    state,
    t,
    canEdit,
    openMetric,
    loadBodyComposition,
    saveBodyComposition,
    addWeight,
  } = useApp();
  const [form, setForm] = useState({
    height: '',
    weight: '',
    fat: '',
    muscle: '',
    water: '',
  });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    loadBodyComposition();
  }, [loadBodyComposition]);

  const records = state.bodyComposition || [];
  const latest = useMemo(() => {
    if (!records.length) return null;
    return [...records].sort((a, b) =>
      String(b.recorded_at || b.recordedAt || '').localeCompare(
        String(a.recorded_at || a.recordedAt || '')
      )
    )[0];
  }, [records]);

  // Реальные значения из подключённого устройства, а не заглушки.
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
      case 'hrv':
        return fmt(latestOf('hrv'));
      case 'stress':
        return fmt(latestOf('stress'));
      default:
        return '—';
    }
  };

  // Composition метрики — только реальные, без выдуманных чисел.
  const weight = latest?.weight_kg ?? latest?.weightKg ?? null;
  const heightCm = latest?.height_cm ?? latest?.heightCm ?? null;
  const fat = latest?.body_fat_percent ?? latest?.bodyFatPercent ?? null;
  const muscle = latest?.muscle_mass_kg ?? latest?.muscleMassKg ?? null;
  const water = latest?.water_percent ?? latest?.waterPercent ?? null;
  const bmi =
    weight && heightCm ? +(weight / (heightCm / 100) ** 2).toFixed(1) : null;
  const goalWeight =
    state.profile?.goalWeight ?? state.profile?.goal_weight ?? null;
  const hasComposition = latest != null;
  const history = state.weightHistory || [];

  const handleChange = (field) => (event) =>
    setForm((current) => ({ ...current, [field]: event.target.value }));

  // Авто-масштаб силуэта по весу (базовый 75 кг): силуэт растёт/сжимается с весом.
  const silhouetteWeight = Number.parseFloat(form.weight) || weight || 0;
  const silhouetteScale =
    silhouetteWeight > 0
      ? Math.min(1.35, Math.max(0.75, 0.75 + (silhouetteWeight - 50) / 120))
      : 1;

  const handleSave = async () => {
    const heightValue = Number.parseFloat(form.height) || heightCm;
    const weightValue = Number.parseFloat(form.weight) || weight;
    if (!heightValue || !weightValue) {
      // Не сохраняем пустую запись без веса/роста.
      return;
    }
    setSaving(true);
    const ok = await saveBodyComposition({
      height_cm: heightValue,
      weight_kg: weightValue,
      body_fat_percent: Number.parseFloat(form.fat) || fat || 0,
      muscle_mass_kg: Number.parseFloat(form.muscle) || muscle || 0,
      water_percent: Number.parseFloat(form.water) || water || 0,
      bmi: +(weightValue / (heightValue / 100) ** 2).toFixed(1),
      recorded_at: new Date().toISOString(),
    });
    if (ok) setForm({ height: '', weight: '', fat: '', muscle: '', water: '' });
    setSaving(false);
  };

  return (
    <section className='body'>
      <div className='page-title'>
        <div className='eyebrow'>BODY COMPOSITION</div>
        <h2>{t('Аналитика тела', 'Body analytics')}</h2>
        <p>
          {t(
            'Вес, жир, мышцы, вода и восстановление — из ваших измерений и устройства.',
            'Weight, fat, muscle, water and recovery — from your measurements and device.'
          )}
        </p>
      </div>

      <section className='metrics'>
        <button
          type='button'
          className='card metric'
          onClick={() => openMetric('weight')}
        >
          <i className='fas fa-weight-scale' />
          <strong>{fmt(weight, 1)}</strong>
          <small>{t('кг', 'kg')}</small>
          <p>{t('Вес', 'Weight')}</p>
        </button>
        <button
          type='button'
          className='card metric'
          onClick={() => openMetric('fat')}
        >
          <i className='fas fa-fire' style={{ color: 'var(--orange)' }} />
          <strong style={{ color: 'var(--orange)' }}>{fmt(fat, 1)}</strong>
          <small>%</small>
          <p>{t('Жир', 'Fat')}</p>
        </button>
        <button
          type='button'
          className='card metric'
          onClick={() => openMetric('muscle')}
        >
          <i className='fas fa-person' style={{ color: 'var(--green)' }} />
          <strong style={{ color: 'var(--green)' }}>{fmt(muscle, 1)}</strong>
          <small>{t('кг', 'kg')}</small>
          <p>{t('Мышцы', 'Muscles')}</p>
        </button>
      </section>

      <div className='section'>
        <h3>{t('Вода и состав', 'Water and composition')}</h3>
      </div>
      <section className='grid-4'>
        <div className='circle-stat' style={{ '--stat': 'var(--blue)' }}>
          <i className='fas fa-droplet' />
          <b>{fmt(water, 1)}%</b>
          <small>{t('Вода', 'Water')}</small>
        </div>
        <div className='circle-stat' style={{ '--stat': 'var(--green)' }}>
          <i className='fas fa-person' />
          <b>{fmt(muscle, 1)}</b>
          <small>{t('Мышцы', 'Muscles')}</small>
        </div>
        <div className='circle-stat' style={{ '--stat': 'var(--orange)' }}>
          <i className='fas fa-fire' />
          <b>{fmt(fat, 1)}%</b>
          <small>{t('Жир', 'Fat')}</small>
        </div>
        <div className='circle-stat' style={{ '--stat': 'var(--purple)' }}>
          <i className='fas fa-weight-scale' />
          <b>{fmt(bmi, 1)}</b>
          <small>{t('ИМТ', 'BMI')}</small>
        </div>
      </section>

      <div className='section'>
        <h3>{t('Силуэт и целевой вес', 'Silhouette and target weight')}</h3>
      </div>
      <section className='body-visual'>
        <div className='weight-layer current'>
          {t('Сейчас', 'Current')} · {fmt(weight, 1)} {t('кг', 'kg')}
        </div>
        {goalWeight ? (
          <div className='weight-layer goal'>
            {t('Цель', 'Goal')} · {fmt(goalWeight, 1)} {t('кг', 'kg')}
          </div>
        ) : null}
        <div className='body-wave' />
        <div className='body-wave' />
        <div className='body-wave' />
        <div
          className='silhouette-container'
          style={{ transform: `scale(${silhouetteScale})` }}
        >
          <svg
            viewBox='0 0 200 280'
            fill='none'
            xmlns='http://www.w3.org/2000/svg'
            role='img'
            aria-label={t('Схема состава тела', 'Body composition diagram')}
          >
            <ellipse
              cx='100'
              cy='70'
              rx='40'
              ry='48'
              fill='#19d8da22'
              stroke='#19d8da66'
              strokeWidth='2'
            />
            <ellipse
              cx='100'
              cy='120'
              rx='48'
              ry='52'
              fill='#6fae2018'
              stroke='#6fae2044'
              strokeWidth='2'
            />
            <ellipse
              cx='100'
              cy='175'
              rx='52'
              ry='50'
              fill='#ff994718'
              stroke='#ff994744'
              strokeWidth='2'
            />
            <ellipse
              cx='100'
              cy='230'
              rx='44'
              ry='44'
              fill='#a78bfa18'
              stroke='#a78bfa44'
              strokeWidth='2'
            />
            <ellipse
              cx='58'
              cy='100'
              rx='18'
              ry='34'
              fill='#19d8da22'
              stroke='#19d8da55'
              strokeWidth='1.5'
            />
            <ellipse
              cx='142'
              cy='100'
              rx='18'
              ry='34'
              fill='#19d8da22'
              stroke='#19d8da55'
              strokeWidth='1.5'
            />
            <ellipse
              cx='58'
              cy='210'
              rx='18'
              ry='30'
              fill='#19d8da22'
              stroke='#19d8da55'
              strokeWidth='1.5'
            />
            <ellipse
              cx='142'
              cy='210'
              rx='18'
              ry='30'
              fill='#19d8da22'
              stroke='#19d8da55'
              strokeWidth='1.5'
            />
            <text
              x='100'
              y='48'
              textAnchor='middle'
              fill='#19d8da88'
              fontSize='8'
              fontWeight='700'
            >
              {t('Мышцы', 'Muscles')} {fmt(muscle, 1)} {t('кг', 'kg')}
            </text>
            <text
              x='100'
              y='262'
              textAnchor='middle'
              fill='#ff994788'
              fontSize='8'
              fontWeight='700'
            >
              {t('Жир', 'Fat')} {fmt(fat, 1)}%
            </text>
          </svg>
        </div>
      </section>

      <div className='section'>
        <h3>{t('История взвешиваний', 'Weight history')}</h3>
      </div>
      <section className='card history-list'>
        {history.length === 0 ? (
          <p className='muted' style={{ fontSize: 11 }}>
            {t('Пока нет записей', 'No entries yet')}
          </p>
        ) : (
          history.map((item, index) => (
            <div className='history-row' key={`${item[0]}-${index}`}>
              <span>{item[0]}</span>
              <b>{item[1]}</b>
            </div>
          ))
        )}
      </section>
      {canEdit && (
        <button
          type='button'
          className='primary full'
          style={{ marginTop: 10 }}
          onClick={addWeight}
        >
          <i className='fas fa-plus' />{' '}
          {t('Добавить взвешивание', 'Add weight')}
        </button>
      )}

      <div className='section'>
        <h3>{t('Показатели организма', 'Body metrics')}</h3>
      </div>
      <section className='metrics'>
        {METRIC_DEFS.map((metric) => (
          <button
            type='button'
            key={metric.key}
            className='card metric'
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

      <div className='section'>
        <h3>{t('График ИМТ', 'BMI chart')}</h3>
      </div>
      <section className='card'>
        <div className='fp-bmi-circle'>
          <div className='fp-bmi-circle-content'>
            <strong>{fmt(bmi, 1)}</strong>
            <small>{t('текущий ИМТ', 'current BMI')}</small>
          </div>
        </div>
        <div className='fp-bmi-legend'>
          <span>
            <i style={{ background: 'var(--blue)' }} />
            {t('Недовес', 'Underweight')}
          </span>
          <span>
            <i style={{ background: 'var(--green)' }} />
            {t('Норма', 'Normal')}
          </span>
          <span>
            <i style={{ background: 'var(--orange)' }} />
            {t('Избыток', 'Overweight')}
          </span>
          <span>
            <i style={{ background: 'var(--red)' }} />
            {t('Высокий', 'High')}
          </span>
        </div>
      </section>

      {hasComposition && (
        <>
          <div className='section'>
            <h3>{t('Другие параметры', 'Other parameters')}</h3>
          </div>
          <section className='card'>
            <div className='body-row'>
              <span>{t('ИМТ', 'BMI')}</span>
              <button
                type='button'
                className='secondary'
                onClick={() => openMetric('bmi')}
              >
                {fmt(bmi, 1)} · {t('график', 'chart')}
              </button>
            </div>
            <div className='body-row'>
              <span>{t('Безжировая масса', 'Lean mass')}</span>
              <b>
                {fmt(
                  weight != null && muscle != null
                    ? Math.max(0, weight - muscle)
                    : null,
                  1
                )}{' '}
                {t('кг', 'kg')}
              </b>
            </div>
          </section>
        </>
      )}

      <section className='card' style={{ marginTop: 10 }}>
        <strong>
          <i className='fas fa-lightbulb' style={{ color: 'var(--yellow)' }} />{' '}
          {t('Лайфхак', 'Tip')}
        </strong>
        <p style={{ marginTop: 7, color: 'var(--muted)', fontSize: 10 }}>
          {t(
            'Взвешивайся утром в одинаковых условиях и оценивай среднее значение за неделю.',
            'Weigh yourself in the morning under the same conditions and track your weekly average.'
          )}
        </p>
      </section>

      <div className='section'>
        <h3>{t('Внести показатели', 'Log measurements')}</h3>
      </div>
      <p className='muted' style={{ fontSize: 10, marginTop: -6 }}>
        {t(
          'Рост — в см, вес — в кг, жир/вода — в %, мышцы — в кг. Можно оставить только нужные поля.',
          'Height in cm, weight in kg, fat/water in %, muscle in kg. Fill only the fields you need.'
        )}
      </p>
      <section className='card form-card'>
        <div className='form-row'>
          <div className='field'>
            <label htmlFor='bodyHeight'>{t('Рост, см', 'Height, cm')}</label>
            <input
              id='bodyHeight'
              type='number'
              value={form.height}
              onChange={handleChange('height')}
              placeholder={heightCm != null ? String(heightCm) : '178'}
            />
          </div>
          <div className='field'>
            <label htmlFor='bodyWeight'>{t('Вес, кг', 'Weight, kg')}</label>
            <input
              id='bodyWeight'
              type='number'
              value={form.weight}
              onChange={handleChange('weight')}
              placeholder={weight != null ? String(weight) : '74.0'}
            />
          </div>
        </div>
        <div className='form-row'>
          <div className='field'>
            <label htmlFor='bodyFat'>{t('Жир, %', 'Fat, %')}</label>
            <input
              id='bodyFat'
              type='number'
              value={form.fat}
              onChange={handleChange('fat')}
              placeholder={fat != null ? String(fat) : '18.0'}
            />
          </div>
          <div className='field'>
            <label htmlFor='bodyMuscle'>{t('Мышцы, кг', 'Muscle, kg')}</label>
            <input
              id='bodyMuscle'
              type='number'
              value={form.muscle}
              onChange={handleChange('muscle')}
              placeholder={muscle != null ? String(muscle) : '34.0'}
            />
          </div>
        </div>
        <div className='field'>
          <label htmlFor='bodyWater'>{t('Вода, %', 'Water, %')}</label>
          <input
            id='bodyWater'
            type='number'
            value={form.water}
            onChange={handleChange('water')}
            placeholder={water != null ? String(water) : '57.0'}
          />
        </div>
        {canEdit && (
          <button
            type='button'
            className='primary full'
            style={{ marginTop: 10 }}
            disabled={saving}
            onClick={handleSave}
          >
            {saving ? t('Сохраняем…', 'Saving…') : t('Сохранить', 'Save')}
          </button>
        )}
      </section>
    </section>
  );
}
