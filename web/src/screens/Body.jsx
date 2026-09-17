import { useApp } from '../contexts/AppContext';

export default function Body() {
  const { state, t, logWeight, go } = useApp();
  const bmi =
    state.height > 0 && state.weight > 0
      ? +(state.weight / (state.height / 100) ** 2).toFixed(1)
      : null;

  return (
    <section className='body'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{t('Профиль тела', 'Body profile')}</div>
          <div className='panel-sub'>
            {state.height ? `${state.height} см` : '—'} ·{' '}
            {state.weight ? `${state.weight} кг` : '—'}
          </div>
        </div>
        <div className='panel-body'>
          <div className='field'>
            <label>{t('Рост, см', 'Height, cm')}</label>
            <input
              id='height'
              type='number'
              defaultValue={state.height || 170}
              onBlur={(e) => logWeight(+e.target.value, state.weight)}
            />
          </div>
          <div className='field'>
            <label>{t('Вес, кг', 'Weight, kg')}</label>
            <input
              id='weight'
              type='number'
              defaultValue={state.weight || 70}
              onBlur={(e) => logWeight(state.height, +e.target.value)}
            />
          </div>
          {bmi && (
            <div className='metric'>
              <div className='metric-val'>{bmi}</div>
              <div className='metric-label'>{t('ИМТ', 'BMI')}</div>
            </div>
          )}
          <button className='secondary full' onClick={() => go('home')}>
            {t('На главную', 'Go home')}
          </button>
        </div>
      </div>
    </section>
  );
}
