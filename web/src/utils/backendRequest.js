const ALLOWED_PROTOCOLS = new Set(['http:', 'https:']);

function resolveBaseUrl(rawUrl) {
  const trimmed = rawUrl.trim();
  if (!trimmed) {
    return '';
  }
  let parsed;
  try {
    parsed = new URL(trimmed);
  } catch {
    return '';
  }
  if (!ALLOWED_PROTOCOLS.has(parsed.protocol)) {
    return '';
  }
  let pathname = parsed.pathname;
  if (pathname === '/') {
    pathname = '';
  } else {
    while (pathname.endsWith('/')) {
      pathname = pathname.slice(0, -1);
    }
  }
  return `${parsed.origin}${pathname}`;
}

const BACKEND_CONFIG = {
  baseUrl: resolveBaseUrl(import.meta.env.VITE_API_BASE_URL || ''),
  production: import.meta.env.PROD === true,
};

// CAPTCHA token produced by the Turnstile widget on the auth screens. It is
// kept outside React state because the widget lives next to uncontrolled form
// fields; backendRequest attaches it to every call so the gateway can validate
// it when the auth rate limiter challenges the client.
let captchaToken = '';

export function setCaptchaToken(token) {
  captchaToken = typeof token === 'string' ? token : '';
}

function captchaHeaders() {
  return captchaToken ? { 'X-Captcha-Token': captchaToken } : {};
}

const UNSAFE_PATH = /\\|(^|\/)\.\.?($|\/)/;

function hasControlCharacters(value) {
  for (const char of value) {
    // codePointAt, а не charCodeAt: суррогатные пары (эмодзи и прочие
    // символы вне BMP) иначе разбираются как два невалидных code unit.
    const code = char.codePointAt(0);
    if (code < 0x20 || code === 0x7f) return true;
  }
  return false;
}

function buildUrl(path) {
  if (
    typeof path !== 'string' ||
    !path.startsWith('/') ||
    path.startsWith('//') ||
    hasControlCharacters(path) ||
    UNSAFE_PATH.test(path)
  ) {
    throw new Error('Invalid request path');
  }
  return `${BACKEND_CONFIG.baseUrl}${path}`;
}

function backendError(status) {
  const error = new Error(`Backend ${status}`);
  error.status = status;
  return error;
}

async function backendRequest(path, options = {}) {
  // Protected routes hide themselves behind 404 when the access token is
  // missing or expired, so callers that know the route exists can opt into
  // treating 404 like 401: refresh the session and retry once.
  const { retryOnNotFound = false, ...requestOptions } = options;
  const url = buildUrl(path);
  const buildRequestInit = () => ({
    ...requestOptions,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...captchaHeaders(),
      ...requestOptions.headers,
    },
  });

  const res = await fetch(url, buildRequestInit());
  const shouldRefresh =
    res.status === 401 || (retryOnNotFound && res.status === 404);
  if (shouldRefresh && path !== '/api/v1/auth/refresh') {
    const refreshed = await backendRequest('/api/v1/auth/refresh', {
      method: 'POST',
    });
    if (refreshed !== null) {
      const retry = await fetch(url, buildRequestInit());
      if (!retry.ok) {
        throw backendError(retry.status);
      }
      return retry.status === 204 ? null : retry.json();
    }
  }
  if (!res.ok) {
    throw backendError(res.status);
  }
  return res.status === 204 ? null : res.json();
}

export { BACKEND_CONFIG, backendRequest, resolveBaseUrl };
