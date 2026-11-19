import { Component } from 'react';

// ErrorBoundary catches render/runtime errors in the screen tree so a single
// broken screen shows a friendly message instead of a blank black screen.
export default class ErrorBoundary extends Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  componentDidCatch(error, info) {
    // eslint-disable-next-line no-console
    console.error('FitPulse render error:', error, info);
  }

  render() {
    if (this.state.hasError) {
      return (
        <section className='page' style={{ padding: 24 }}>
          <div className='card' style={{ textAlign: 'center' }}>
            <div className='eyebrow'>FITPULSE</div>
            <h2 style={{ margin: '8px 0' }}>
              {this.props.t
                ? this.props.t('Что-то пошло не так', 'Something went wrong')
                : 'Что-то пошло не так'}
            </h2>
            <p className='muted' style={{ marginBottom: 16 }}>
              {this.props.t
                ? this.props.t(
                    'Обновите страницу или вернитесь на главную.',
                    'Reload the page or go back home.'
                  )
                : 'Обновите страницу или вернитесь на главную.'}
            </p>
            <button
              type='button'
              className='primary'
              onClick={() => {
                window.location.href = '/';
              }}
            >
              {this.props.t ? this.props.t('На главную', 'Home') : 'На главную'}
            </button>
          </div>
        </section>
      );
    }
    return this.props.children;
  }
}
