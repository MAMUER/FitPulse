const BACKEND_CONFIG = {
  baseUrl: localStorage.getItem('fitpulse-api-base') || '',
  production: localStorage.getItem('fitpulse-production') === '1',
};

async function backendRequest(path, options = {}) {
  if (!BACKEND_CONFIG.baseUrl) {
    return null;
  }
  const url = `${BACKEND_CONFIG.baseUrl.replace(/\/$/, '')}${path}`;
  const res = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
  });
  if (res.status === 401) {
    const refreshed = await backendRequest('/api/v1/auth/refresh', {
      method: 'POST',
    });
    if (refreshed !== null) {
      const retry = await fetch(url, {
        ...options,
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          ...(options.headers || {}),
        },
      });
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

export { BACKEND_CONFIG };
