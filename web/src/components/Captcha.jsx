import { useEffect, useRef, useState } from 'react';

// Public site key, must match CLOUDFLARE_TURNSTILE_SITE_KEY in
// configs/k8s/base/captcha/configmap.yaml. The matching secret stays
// server-side in the `captcha-secrets` Secret. Override at build time with
// VITE_CAPTCHA_SITE_KEY when rotating keys.
export const CAPTCHA_SITE_KEY =
  import.meta.env.VITE_CAPTCHA_SITE_KEY || '0x4AAAAAAFAyPN_m3SG9U_kR';

const SCRIPT_ID = 'cf-turnstile-script';
const SCRIPT_SRC =
  'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';

function loadTurnstile() {
  if (window.turnstile) {
    return Promise.resolve(window.turnstile);
  }

  const existing = document.getElementById(SCRIPT_ID);
  if (existing) {
    return new Promise((resolve, reject) => {
      const onError = () => reject(new Error('turnstile failed to load'));
      existing.addEventListener('load', () => resolve(window.turnstile), {
        once: true,
      });
      existing.addEventListener('error', onError, { once: true });
    });
  }

  return new Promise((resolve, reject) => {
    const script = document.createElement('script');
    script.id = SCRIPT_ID;
    script.src = SCRIPT_SRC;
    script.async = true;
    script.onload = () => resolve(window.turnstile);
    script.onerror = () => reject(new Error('turnstile failed to load'));
    document.head.appendChild(script);
  });
}

/**
 * Renders the Cloudflare Turnstile widget and reports its token through
 * `onToken`, which the auth screens wire to the X-Captcha-Token header.
 *
 * Turnstile is bound to the production domain (fittpulse.duckdns.org). On a
 * local/LAN host (e.g. 172.27.x.x) the sitekey cannot be verified and the
 * widget would render an error box that blocks the form, so when Turnstile
 * fails to load or errors we hide the container instead of showing the error.
 * The backend still validates the captcha token only when its rate limiter
 * challenges the client.
 */
export default function Captcha({ onToken, className = 'captcha' }) {
  const containerRef = useRef(null);
  const onTokenRef = useRef(onToken);
  const [hidden, setHidden] = useState(false);
  onTokenRef.current = onToken;

  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return undefined;
    }
    let widgetId = null;
    let cancelled = false;
    const emit = (token) => onTokenRef.current?.(token || '');
    const hide = () => {
      if (!cancelled) setHidden(true);
    };

    loadTurnstile()
      .then((turnstile) => {
        if (cancelled) {
          return;
        }
        widgetId = turnstile.render(container, {
          sitekey: CAPTCHA_SITE_KEY,
          callback: emit,
          'expired-callback': () => emit(''),
          'error-callback': () => {
            emit('');
            hide();
          },
        });
      })
      .catch(() => {
        emit('');
        hide();
      });

    return () => {
      cancelled = true;
      if (widgetId && window.turnstile?.remove) {
        window.turnstile.remove(widgetId);
      }
    };
  }, []);

  return (
    <div
      ref={containerRef}
      className={className}
      style={hidden ? { display: 'none' } : undefined}
    />
  );
}
