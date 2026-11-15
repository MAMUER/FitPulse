import { useEffect, useMemo, useState } from 'react';
import { useApp } from '../contexts/AppContext';

const METRICS = [
  {
    key: 'pulse',
    icon: 'fa-heart-pulse',
    value: '72',
    unit: 'bpm',
    label: ['Пульс', 'Pulse'],
  },
  {
    key: 'sleep',
    icon: 'fa-moon',
    value: '7.2',
    unit: ['часа', 'hours'],
    label: ['Сон', 'Sleep'],
  },
  {
    key: 'oxygen',
    icon: 'fa-lungs',
    value: '98',
    unit: '%',
    label: ['Кислород', 'Oxygen'],
  },
  {
    key: 'hrv',
    icon: 'fa-wave-square',
    value: '64',
    unit: 'мс',
    label: ['ВСР', 'HRV'],
  },
  {
    key: 'stress',
    icon: 'fa-brain',
    value: '32',
    unit: ['уровень', 'level'],
    label: ['Стресс', 'Stress'],
  },
];

export default function Body() {
  const {
    state,
    t,
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

  const weight = latest?.weight_kg ?? latest?.weightKg ?? 74.2;
  const heightCm = latest?.height_cm ?? latest?.heightCm ?? 178;
  const fat = latest?.body_fat_percent ?? latest?.bodyFatPercent ?? 18.4;
  const muscle = latest?.muscle_mass_kg ?? latest?.muscleMassKg ?? 34.8;
  const water = latest?.water_percent ?? latest?.waterPercent ?? 57.2;
  const bmi = heightCm > 0 ? +(weight / (heightCm / 100) ** 2).toFixed(1) : 0;
  const history = state.weightHistory || [];

  const handleChange = (field) => (event) =>
    setForm((current) => ({ ...current, [field]: event.target.value }));

  const handleSave = async () => {
    const heightValue = Number.parseFloat(form.height) || heightCm;
    const weightValue = Number.parseFloat(form.weight) || weight;
    setSaving(true);
    const ok = await saveBodyComposition({
      height_cm: heightValue,
      weight_kg: weightValue,
      body_fat_percent: Number.parseFloat(form.fat) || fat,
      muscle_mass_kg: Number.parseFloat(form.muscle) || muscle,
      water_percent: Number.parseFloat(form.water) || water,
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
            'Вес, жир, мышцы, белок, вода, минералы и восстановление.',
            'Weight, fat, muscle, protein, water, minerals and recovery.'
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
          <strong>{weight}</strong>
          <small>{t('кг', 'kg')}</small>
          <p>{t('Вес', 'Weight')}</p>
        </button>
        <button
          type='button'
          className='card metric'
          onClick={() => openMetric('fat')}
        >
          <i className='fas fa-fire' style={{ color: 'var(--orange)' }} />
          <strong style={{ color: 'var(--orange)' }}>{fat}</strong>
          <small>%</small>
          <p>
            {t('Жир', 'Fat')} · {t('ниже нормы', 'below normal')}
          </p>
        </button>
        <button
          type='button'
          className='card metric'
          onClick={() => openMetric('muscle')}
        >
          <i className='fas fa-person' style={{ color: 'var(--green)' }} />
          <strong style={{ color: 'var(--green)' }}>{muscle}</strong>
          <small>{t('кг', 'kg')}</small>
          <p>
            {t('Мышцы', 'Muscles')} · {t('выше нормы', 'above normal')}
          </p>
        </button>
        <button
          type='button'
          className='card metric'
          onClick={() => openMetric('protein')}
        >
          <i className='fas fa-dumbbell' style={{ color: 'var(--purple)' }} />
          <strong style={{ color: 'var(--purple)' }}>16.2</strong>
          <small>%</small>
          <p>
            {t('Белок', 'Protein')} · {t('норма', 'normal')}
          </p>
        </button>
      </section>

      <div className='section'>
        <h3>{t('Вода и минералы', 'Water and minerals')}</h3>
      </div>
      <section className='grid-4'>
        <div className='circle-stat' style={{ '--stat': 'var(--blue)' }}>
          <i className='fas fa-droplet' />
          <b>{water}%</b>
          <small>{t('Вода', 'Water')}</small>
        </div>
        <div className='circle-stat' style={{ '--stat': 'var(--yellow)' }}>
          <i className='fas fa-cubes' />
          <b>4.1%</b>
          <small>{t('Минералы', 'Minerals')}</small>
        </div>
        <div className='circle-stat' style={{ '--stat': 'var(--green)' }}>
          <i className='fas fa-person' />
          <b>{muscle}</b>
          <small>{t('Мышцы', 'Muscles')}</small>
        </div>
        <div className='circle-stat' style={{ '--stat': 'var(--orange)' }}>
          <i className='fas fa-fire' />
          <b>{fat}%</b>
          <small>{t('Жир', 'Fat')}</small>
        </div>
      </section>

      <div className='section'>
        <h3>{t('Силуэт и целевой вес', 'Silhouette and target weight')}</h3>
      </div>
      <section className='body-visual'>
        <div className='weight-layer current'>
          {t('Сейчас', 'Current')} · {weight} {t('кг', 'kg')}
        </div>
        <div className='weight-layer goal'>
          {t('Цель', 'Goal')} · 72 {t('кг', 'kg')}
        </div>
        <div className='body-wave' />
        <div className='body-wave' />
        <div className='body-wave' />
        <div className='silhouette-container'>
          <svg
            viewBox='0 0 200 280'
            fill='none'
            xmlns='http://www.w3.org/2000/svg'
            role='img'
            aria-label={t('Схема состава тела', 'Body composition diagram')}
          >
            <ellipse cx='100' cy='70' rx='40' ry='48' fill='#19d8da22' stroke='#19d8da66' strokeWidth='2' />
            <ellipse cx='100' cy='120' rx='48' ry='52' fill='#6fae2018' stroke='#6fae2044' strokeWidth='2' />
            <ellipse cx='100' cy='175' rx='52' ry='50' fill='#ff994718' stroke='#ff994744' strokeWidth='2' />
            <ellipse cx='100' cy='230' rx='44' ry='44' fill='#a78bfa18' stroke='#a78bfa44' strokeWidth='2' />
            <ellipse cx='58' cy='100' rx='18' ry='34' fill='#19d8da22' stroke='#19d8da55' strokeWidth='1.5' />
            <ellipse cx='142' cy='100' rx='18' ry='34' fill='#19d8da22' stroke='#19d8da55' strokeWidth='1.5' />
            <ellipse cx='58' cy='210' rx='18' ry='30' fill='#19d8da22' stroke='#19d8da55' strokeWidth='1.5' />
            <ellipse cx='142' cy='210' rx='18' ry='30' fill='#19d8da22' stroke='#19d8da55' strokeWidth='1.5' />
            <text x='100' y='48' textAnchor='middle' fill='#19d8da88' fontSize='8' fontWeight='700'>
              {t('Мышцы', 'Muscles')} {muscle} {t('кг', 'kg')}
            </text>
            <text x='100' y='262' textAnchor='middle' fill='#ff994788' fontSize='8' fontWeight='700'>
              {t('Жир', 'Fat')} {fat}%
            </text>
          </svg>
        </div>
      </section>

      <div className='section'>
        <h3>{t('История взвешиваний', 'Weight history')}</h3>
      </div>
      <section className='card history-list'>
        {history.map((item, index) => (
          <div className='history-row' key={`${item[0]}-${index}`}>
            <span>{item[0]}</span>
            <b>{item[1]}</b>
          </div>
        ))}
      </section>
      <button
        type='button'
        className='primary full'
        style={{ marginTop: 10 }}
        onClick={addWeight}
      >
        <i className='fas fa-plus' /> {t('Добавить взвешивание', 'Add weight')}
      </button>

      <div className='section'>
        <h3>{t('Показатели организма', 'Body metrics')}</h3>
      </div>
      <section className='metrics'>
        {METRICS.map((metric) => (
          <button
            type='button'
            key={metric.key}
            className='card metric'
            onClick={() => openMetric(metric.key)}
          >
            <i className={`fas ${metric.icon}`} />
            <strong>{metric.value}</strong>
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
            <strong>{bmi}</strong>
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

      <div className='section'>
        <h3>{t('Другие параметры', 'Other parameters')}</h3>
      </div>
      <section className='card'>
        <div className='body-row'>
          <span>{t('Висцеральный жир', 'Visceral fat')}</span>
          <b>
            6 <em className='status good'>{t('норма', 'normal')}</em>
          </b>
        </div>
        <div className='body-row'>
          <span>{t('ИМТ', 'BMI')}</span>
          <button
            type='button'
            className='secondary'
            onClick={() => openMetric('bmi')}
          >
            {bmi} · {t('график', 'chart')}
          </button>
        </div>
        <div className='body-row'>
          <span>{t('Безжировая масса', 'Lean mass')}</span>
          <b>{Math.max(0, +(weight - muscle).toFixed(1))} кг</b>
        </div>
        <div className='body-row'>
          <span>{t('Основной обмен', 'Basal metabolism')}</span>
          <b>1 720 {t('ккал', 'kcal')}</b>
        </div>
      </section>

      <div className='section'>
        <h3>{t('Анализ по сегментам', 'Segment analysis')}</h3>
      </div>
      <section className='card segments'>
        <div className='segment'>
          <i className='fas fa-hand' />
          <b>92%</b>
          <small>{t('Левая рука', 'Left arm')}</small>
        </div>
        <div className='segment'>
          <i className='fas fa-hand' />
          <b>94%</b>
          <small>{t('Правая рука', 'Right arm')}</small>
        </div>
        <div className='segment'>
          <i className='fas fa-person' />
          <b>88%</b>
          <small>{t('Торс', 'Torso')}</small>
        </div>
        <div className='segment'>
          <i className='fas fa-shoe-prints' />
          <b>91%</b>
          <small>{t('Левая нога', 'Left leg')}</small>
        </div>
        <div className='segment'>
          <i className='fas fa-shoe-prints' />
          <b>93%</b>
          <small>{t('Правая нога', 'Right leg')}</small>
        </div>
      </section>

      <div className='section'>
        <h3>{t('Рекомендация', 'Recommendation')}</h3>
      </div>
      <section className='card'>
        <div className='body-row'>
          <span>{t('Оптимальный вес', 'Optimal weight')}</span>
          <b style={{ color: 'var(--green)' }}>72–76 кг</b>
        </div>
        <div className='body-row'>
          <span>{t('Ежедневный приём', 'Daily intake')}</span>
          <b style={{ color: 'var(--blue)' }}>
            2 350–2 450 {t('ккал', 'kcal')}
          </b>
        </div>
        <div className='body-row'>
          <span>{t('Белок', 'Protein')}</span>
          <b>120–145 г</b>
        </div>
        <p style={{ marginTop: 12, color: 'var(--muted)', fontSize: 10, lineHeight: 1.5 }}>
          {t(
            'Сохраняй силовые тренировки 3 раза в неделю, добавь белок после занятия и держи стабильный сон.',
            'Keep strength training 3 times a week, add protein after workouts and maintain stable sleep.'
          )}
        </p>
      </section>

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
      <section className='card form-card'>
        <div className='form-row'>
          <div className='field'>
            <label htmlFor='bodyHeight'>{t('Рост, см', 'Height, cm')}</label>
            <input
              id='bodyHeight'
              type='number'
              value={form.height}
              onChange={handleChange('height')}
              placeholder={String(heightCm)}
            />
          </div>
          <div className='field'>
            <label htmlFor='bodyWeight'>{t('Вес, кг', 'Weight, kg')}</label>
            <input
              id='bodyWeight'
              type='number'
              value={form.weight}
              onChange={handleChange('weight')}
              placeholder={String(weight)}
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
              placeholder={String(fat)}
            />
          </div>
          <div className='field'>
            <label htmlFor='bodyMuscle'>{t('Мышцы, кг', 'Muscle, kg')}</label>
            <input
              id='bodyMuscle'
              type='number'
              value={form.muscle}
              onChange={handleChange('muscle')}
              placeholder={String(muscle)}
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
            placeholder={String(water)}
          />
        </div>
        <button
          type='button'
          className='primary full'
          style={{ marginTop: 10 }}
          disabled={saving}
          onClick={handleSave}
        >
          {saving ? t('Сохраняем…', 'Saving…') : t('Сохранить', 'Save')}
        </button>
      </section>
    </section>
  );
}
