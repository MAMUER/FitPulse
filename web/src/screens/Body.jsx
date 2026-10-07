import { useEffect, useState } from 'react';
import { useApp } from '../contexts/AppContext';
import Panel from '../components/Panel';
import MetricCard from '../components/MetricCard';
import ListItem from '../components/ListItem';

export default function Body() {
  const { state, t, go, loadBodyComposition, saveBodyComposition } = useApp();
  const [height, setHeight] = useState(state.height || 170);
  const [weight, setWeight] = useState(state.weight || 70);
  const [bodyFat, setBodyFat] = useState('');
  const [muscle, setMuscle] = useState('');
  const [water, setWater] = useState('');
  const [saving, setSaving] = useState(false);
  const records = state.bodyComposition || [];

  useEffect(() => {
    loadBodyComposition();
  }, [loadBodyComposition]);

  const bmi =
    height > 0 && weight > 0
      ? +(weight / (height / 100) ** 2).toFixed(1)
      : null;

  const handleSave = async () => {
    setSaving(true);
    const record = {
      height_cm: height,
      weight_kg: weight,
      bmi: bmi || 0,
      body_fat_percentage: bodyFat ? Number.parseFloat(bodyFat) : 0,
      muscle_mass_percentage: muscle ? Number.parseFloat(muscle) : 0,
      water_percentage: water ? Number.parseFloat(water) : 0,
      recorded_at: new Date().toISOString(),
      source: 'manual',
    };
    await saveBodyComposition(record);
    setSaving(false);
  };

  return (
    <section className='body'>
      <Panel
        title={t('Профиль тела', 'Body profile')}
        subtitle={
          height && weight
            ? `${height} см · ${weight} кг`
            : '—'
        }
      >
        <div className='panel-body'>
          <p className='medical-disclaimer'>
            {t(
              'Это не медицинский совет. При заболеваниях или травмах consult врача.',
              'This is not medical advice. Consult a doctor if you have medical conditions or injuries.'
            )}
          </p>
          <div className='field'>
            <label htmlFor='height'>{t('Рост, см', 'Height, cm')}</label>
            <input
              id='height'
              type='number'
              value={height}
              onChange={(e) => setHeight(+e.target.value)}
            />
          </div>
          <div className='field'>
            <label htmlFor='weight'>{t('Вес, кг', 'Weight, kg')}</label>
            <input
              id='weight'
              type='number'
              value={weight}
              onChange={(e) => setWeight(+e.target.value)}
            />
          </div>
          <div className='field'>
            <label htmlFor='bodyFat'>{t('Жир, %', 'Body fat, %')}</label>
            <input
              id='bodyFat'
              type='number'
              value={bodyFat}
              onChange={(e) => setBodyFat(e.target.value)}
              placeholder='18.4'
            />
          </div>
          <div className='field'>
            <label htmlFor='muscle'>{t('Мышцы, %', 'Muscle, %')}</label>
            <input
              id='muscle'
              type='number'
              value={muscle}
              onChange={(e) => setMuscle(e.target.value)}
              placeholder='34.8'
            />
          </div>
          <div className='field'>
            <label htmlFor='water'>{t('Вода, %', 'Water, %')}</label>
            <input
              id='water'
              type='number'
              value={water}
              onChange={(e) => setWater(e.target.value)}
              placeholder='50'
            />
          </div>
          {bmi !== null && (
            <MetricCard value={bmi} label={t('ИМТ', 'BMI')} />
          )}
          <button
            type='button'
            className='primary full'
            onClick={handleSave}
            disabled={saving}
          >
            {saving ? t('Сохранение...', 'Saving...') : t('Сохранить', 'Save')}
          </button>
          <button
            type='button'
            className='secondary full'
            onClick={() => go('home')}
          >
            {t('На главную', 'Go home')}
          </button>
        </div>
      </Panel>
      {records.length > 0 && (
        <Panel title={t('История', 'History')}>
          <div className='panel-body'>
            {records.map((r, i) => (
              <ListItem
                key={r.recorded_at || r.id || i}
                title={`${r.weight_kg} кг · ИМТ ${r.bmi}`}
                subtitle={
                  r.recorded_at
                    ? new Date(r.recorded_at).toLocaleDateString()
                    : ''
                }
              />
            ))}
          </div>
        </Panel>
      )}
    </section>
  );
}
