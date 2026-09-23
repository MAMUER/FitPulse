import { useNavigate } from 'react-router-dom';
import { useApp } from '../contexts/AppContext';

export default function Landing() {
  const { state, t } = useApp();
  const navigate = useNavigate();

  return (
    <section className='landing'>
      <div className='landing-hero'>
        <div className='landing-brand'>
          <div className='logo-icon'>FP</div>
          <h1>FitPulse</h1>
          <p>
            {t(
              'Open fitness & health platform',
              'Open fitness & health platform'
            )}
          </p>
        </div>
        <div className='landing-actions'>
          {state.registered ? (
            <button className='primary full' onClick={() => navigate('/home')}>
              {t('Open app', 'Open app')}
            </button>
          ) : (
            <>
              <button
                className='primary full'
                onClick={() => navigate('/login')}
              >
                {t('Sign in', 'Sign in')}
              </button>
              <button
                className='secondary full'
                onClick={() => navigate('/register')}
              >
                {t('Create account', 'Create account')}
              </button>
            </>
          )}
        </div>
      </div>
      <div className='landing-features'>
        <div className='feature'>
          <div className='feature-icon'>📊</div>
          <div className='feature-title'>
            {t('Health tracking', 'Health tracking')}
          </div>
          <div className='feature-desc'>
            {t(
              'Track heart rate, SpO2, steps, calories, sleep and more',
              'Track heart rate, SpO2, steps, calories, sleep and more'
            )}
          </div>
        </div>
        <div className='feature'>
          <div className='feature-icon'>🤖</div>
          <div className='feature-title'>AI</div>
          <div className='feature-desc'>
            {t(
              'Get personalized recommendations based on your data',
              'Get personalized recommendations based on your data'
            )}
          </div>
        </div>
        <div className='feature'>
          <div className='feature-icon'>⌚</div>
          <div className='feature-title'>
            {t('Device integrations', 'Device integrations')}
          </div>
          <div className='feature-desc'>
            {t(
              'Connect wearables and health services via Open Wearables',
              'Connect wearables and health services via Open Wearables'
            )}
          </div>
        </div>
      </div>
      <div className='landing-footer'>
        <a className='legal-link' onClick={() => navigate('/privacy')}>
          {t('Privacy Policy', 'Privacy Policy')}
        </a>
        <span> · </span>
        <a className='legal-link' onClick={() => navigate('/terms')}>
          {t('Terms of Service', 'Terms of Service')}
        </a>
      </div>
    </section>
  );
}
