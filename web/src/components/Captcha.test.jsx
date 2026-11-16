import { render, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import Captcha, { CAPTCHA_SITE_KEY } from './Captcha';

const SCRIPT_ID = 'cf-turnstile-script';

/**
 * Turnstile is loaded from a third-party script that jsdom never executes, so
 * the tests drive the two ways loadTurnstile() can resolve: an already-present
 * window.turnstile, and a script element whose load/error event we fire by hand.
 */
function makeTurnstile(overrides = {}) {
  return {
    render: vi.fn(() => 'widget-1'),
    remove: vi.fn(),
    ...overrides,
  };
}

function removeExistingScript() {
  document.getElementById(SCRIPT_ID)?.remove();
}

describe('Captcha', () => {
  beforeEach(() => {
    removeExistingScript();
    delete window.turnstile;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    removeExistingScript();
    delete window.turnstile;
  });

  it('renders a container even though the widget cannot load in jsdom', () => {
    render(<Captcha onToken={vi.fn()} />);
    expect(document.querySelector('.captcha')).toBeInTheDocument();
  });

  it('exports the public site key', () => {
    expect(CAPTCHA_SITE_KEY).toMatch(/^0x/);
  });

  it('renders immediately when turnstile is already on window', async () => {
    window.turnstile = makeTurnstile();
    render(<Captcha onToken={vi.fn()} />);

    await waitFor(() => expect(window.turnstile.render).toHaveBeenCalled());
    expect(window.turnstile.render.mock.calls[0][1]).toMatchObject({
      sitekey: CAPTCHA_SITE_KEY,
    });
  });

  it('forwards the token through the callback', async () => {
    let options;
    window.turnstile = makeTurnstile({
      render: vi.fn((_el, opts) => {
        options = opts;
        return 'widget-1';
      }),
    });
    const onToken = vi.fn();
    render(<Captcha onToken={onToken} />);

    await waitFor(() => expect(options).toBeDefined());
    options.callback('token-value');
    expect(onToken).toHaveBeenCalledWith('token-value');

    options['expired-callback']();
    expect(onToken).toHaveBeenLastCalledWith('');

    options['error-callback']();
    expect(onToken).toHaveBeenLastCalledWith('');
  });

  it('removes the widget on unmount', async () => {
    window.turnstile = makeTurnstile();
    const { unmount } = render(<Captcha onToken={vi.fn()} />);

    await waitFor(() => expect(window.turnstile.render).toHaveBeenCalled());
    unmount();
    expect(window.turnstile.remove).toHaveBeenCalledWith('widget-1');
  });

  it('does not render the widget after unmounting mid-load', async () => {
    window.turnstile = makeTurnstile();
    const { unmount } = render(<Captcha onToken={vi.fn()} />);

    unmount();
    await Promise.resolve();
    expect(window.turnstile.render).not.toHaveBeenCalled();
  });

  it('appends the script tag once and resolves when it loads', async () => {
    const api = makeTurnstile();
    const appended = [];
    vi.spyOn(document.head, 'appendChild').mockImplementation((node) => {
      appended.push(node);
      setTimeout(() => {
        window.turnstile = api;
        node.onload?.();
      }, 0);
      return node;
    });

    render(<Captcha onToken={vi.fn()} />);

    await waitFor(() => expect(api.render).toHaveBeenCalled());
    expect(appended).toHaveLength(1);
    const script = appended[0];
    expect(script.id).toBe(SCRIPT_ID);
    expect(script.src).toContain('challenges.cloudflare.com');
    expect(script.async).toBe(true);
  });

  it('reuses an existing script tag and waits for its load event', async () => {
    const existing = document.createElement('script');
    existing.id = SCRIPT_ID;
    document.head.appendChild(existing);

    const api = makeTurnstile();
    render(<Captcha onToken={vi.fn()} />);

    // turnstile is not on window yet, so nothing can render before the load event
    expect(api.render).not.toHaveBeenCalled();

    window.turnstile = api;
    existing.dispatchEvent(new Event('load'));

    await waitFor(() => expect(api.render).toHaveBeenCalled());
  });

  it('emits an empty token when the script fails to load', async () => {
    vi.spyOn(document.head, 'appendChild').mockImplementation((node) => {
      setTimeout(() => node.onerror?.(), 0);
      return node;
    });

    const onToken = vi.fn();
    render(<Captcha onToken={onToken} />);

    await waitFor(() => expect(onToken).toHaveBeenCalledWith(''));
  });

  it('applies a custom class name', () => {
    const { container } = render(
      <Captcha onToken={vi.fn()} className='custom-captcha' />
    );
    expect(container.querySelector('.custom-captcha')).toBeInTheDocument();
    expect(container.querySelector('.captcha')).toBeNull();
  });
});
