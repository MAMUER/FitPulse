import { useState, useEffect } from 'react';
import { useApp } from '../contexts/AppContext';

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
      body_fat_percentage: bodyFat ? parseFloat(bodyFat) : 0,
      muscle_mass_percentage: muscle ? parseFloat(muscle) : 0,
      water_percentage: water ? parseFloat(water) : 0,
      recorded_at: new Date().toISOString(),
      source: 'manual',
    };
    await saveBodyComposition(record);
    setSaving(false);
  };

  return (
    <section className='body'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Профиль тела', 'Body profile')}</div>
          <div className='panel-sub'>
            {height ? `${height} см` : '—'} ·{' '}
            {weight ? `${weight} кг` : '—'}
          </div>
        </div>
        <div className='panel-body'>
          <div className='field'>
            <label>{t('Рост, см', 'Height, cm')}</label>
            <input
              id='height'
              type='number'
              value={height}
              onChange={(e) => setHeight(+e.target.value)}
            />
          </div>
          <div className='field'>
            <label>{t('Вес, кг', 'Weight, kg')}</label>
            <input
              id='weight'
              type='number'
              value={weight}
              onChange={(e) => setWeight(+e.target.value)}
            />
          </div>
          <div className='field'>
            <label>{t('Жир, %', 'Body fat, %')}</label>
            <input
              id='bodyFat'
              type='number'
              value={bodyFat}
              onChange={(e) => setBodyFat(e.target.value)}
              placeholder='18.4'
            />
          </div>
          <div className='field'>
            <label>{t('Мышцы, %', 'Muscle, %')}</label>
            <input
              id='muscle'
              type='number'
              value={muscle}
              onChange={(e) => setMuscle(e.target.value)}
              placeholder='34.8'
            />
          </div>
          <div className='field'>
            <label>{t('Вода, %', 'Water, %')}</label>
            <input
              id='water'
              type='number'
              value={water}
              onChange={(e) => setWater(e.target.value)}
              placeholder='50'
            />
          </div>
          {bmi && (
            <div className='metric'>
              <div className='metric-val'>{bmi}</div>
              <div className='metric-label'>{t('ИМТ', 'BMI')}</div>
            </div>
          )}
          <button
            className='primary full'
            onClick={handleSave}
            disabled={saving}
          >
            {saving ? t('Сохранение...', 'Saving...') : t('Сохранить', 'Save')}
          </button>
          <button className='secondary full' onClick={() => go('home')}>
            {t('На главную', 'Go home')}
          </button>
        </div>
      </div>
      {records.length > 0 && (
        <div className='panel'>
          <div className='panel-head'>
            <div className='panel-title'>{t('История', 'History')}</div>
          </div>
          <div className='panel-body'>
            {records.map((r, i) => (
              <div key={i} className='list-row'>
                <div>
                  <div className='list-title'>
                    {r.weight_kg} кг · ИМТ {r.bmi}
                  </div>
                  <div className='muted'>
                    {r.recorded_at ? new Date(r.recorded_at).toLocaleDateString() : ''}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
