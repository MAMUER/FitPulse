import { useApp } from '../contexts/AppContext';

export default function Legal() {
  const { state, t, backFromLegal } = useApp();
  const kind =
    state.screen === 'privacy'
      ? 'privacy'
      : state.screen === 'consent'
        ? 'consent'
        : 'terms';
  const title =
    kind === 'terms'
      ? t('Пользовательское соглашение', 'Terms of Use')
      : kind === 'consent'
        ? t(
            'Соглашение об использовании персональных данных',
            'Personal Data Agreement'
          )
        : t('Политика конфиденциальности', 'Privacy Policy');
  const body =
    kind === 'terms'
      ? t('Условия использования сервиса FitPulse.', 'FitPulse Terms of Use.')
      : kind === 'consent'
        ? t(
            'Согласие на обработку персональных данных.',
            'Personal Data Processing Consent.'
          )
        : t(
            'Политика обработки и защиты персональных данных.',
            'Personal Data Privacy Policy.'
          );

  return (
    <section className='legal'>
      <div className='panel'>
        <div className='panel-head'>
          <div className='panel-title'>{title}</div>
        </div>
        <div className='panel-body'>
          <div className='muted'>{body}</div>
          <button className='secondary full' onClick={backFromLegal}>
            {t('Назад', 'Back')}
          </button>
        </div>
      </div>
    </section>
  );
}
