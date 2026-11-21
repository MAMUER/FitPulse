import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import ErrorBoundary from './ErrorBoundary';

function Boom({ when }) {
  if (when) throw new Error('boom');
  return <div>safe content</div>;
}

describe('ErrorBoundary', () => {
  it('renders children when no error occurs', () => {
    render(
      <ErrorBoundary>
        <Boom when={false} />
      </ErrorBoundary>
    );
    expect(screen.getByText('safe content')).toBeInTheDocument();
  });

  it('renders fallback UI when a child throws', () => {
    // React logs the caught error; silence it for this assertion.
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
    render(
      <ErrorBoundary>
        <Boom when />
      </ErrorBoundary>
    );
    expect(screen.getByText(/что-то пошло не так/i)).toBeInTheDocument();
    spy.mockRestore();
  });
});
