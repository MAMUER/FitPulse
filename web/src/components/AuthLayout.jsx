import { useApp } from '../contexts/AppContext';

export default function AuthLayout({ children }) {
  const { state } = useApp();

  return (
    <section className='auth'>
      <div className='auth-brand'>
        <img
          className='splash-logo'
          src='https://uploads.onecompiler.io/44temhxkf/1785230799105/fit%20pulse.jpg'
          alt='FitPulse'
        />
        <h1>FitPulse</h1>
      </div>
      {children}
      <div className='auth-link' style={{ marginTop: 8 }}>
        <button
          type='button'
          className='secondary auth-control'
          onClick={() => state.toggleHighContrast()}
          style={{ fontSize: 9 }}
        >
          <i className='fas fa-eye'></i>{' '}
          {state.t('Режим высокой контрастности', 'High contrast mode')}
        </button>
        <button
          type='button'
          className='secondary auth-control lang-control'
          onClick={() => state.toggleLanguage()}
          style={{ fontSize: 9, marginLeft: 6 }}
        >
          <i className='fas fa-globe'></i>{' '}
          {state.language === 'ru' ? 'EN' : 'RU'}
        </button>
      </div>
    </section>
  );
}
