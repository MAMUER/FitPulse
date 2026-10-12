const BACKEND_CONFIG = {
  baseUrl: localStorage.getItem('fitpulse-api-base') || '',
  production: localStorage.getItem('fitpulse-production') === '1',
};

const ALLOWED_PROTOCOLS = new Set(['http:', 'https:']);

function resolveBaseUrl(rawUrl) {
  const trimmed = rawUrl.trim();
  if (!trimmed) {
    return null;
  }
  let parsed;
  try {
    parsed = new URL(trimmed);
  } catch {
    return null;
  }
  if (!ALLOWED_PROTOCOLS.has(parsed.protocol)) {
    return null;
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

async function backendRequest(path, options = {}) {
  const baseUrl = resolveBaseUrl(BACKEND_CONFIG.baseUrl);
  if (!baseUrl) {
    return null;
  }
  const url = `${baseUrl}${path}`;
  const buildRequestInit = () => ({
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  const res = await fetch(url, buildRequestInit());
  if (res.status === 401) {
    const refreshed = await backendRequest('/api/v1/auth/refresh', {
      method: 'POST',
    });
    if (refreshed !== null) {
      const retry = await fetch(url, buildRequestInit());
      if (!retry.ok) {
        throw new Error(`Backend ${retry.status}`);
      }
      return retry.status === 204 ? null : retry.json();
    }
  }
  if (!res.ok) {
    throw new Error(`Backend ${res.status}`);
  }
  return res.status === 204 ? null : res.json();
}

export { BACKEND_CONFIG, backendRequest, resolveBaseUrl };
