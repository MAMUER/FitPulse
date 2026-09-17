const BACKEND_CONFIG = {
  baseUrl: localStorage.getItem('fitpulse-api-base') || '',
  production: localStorage.getItem('fitpulse-production') === '1',
};

export async function backendRequest(path, options = {}) {
  if (!BACKEND_CONFIG.baseUrl) {
    return null;
  }
  const res = await fetch(BACKEND_CONFIG.baseUrl.replace(/\/$/, '') + path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
  });
  if (!res.ok) {
    throw new Error(`Backend ${res.status}`);
  }
  return res.status === 204 ? null : res.json();
}

export { BACKEND_CONFIG };
