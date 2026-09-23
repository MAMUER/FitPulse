const BACKEND_CONFIG = {
  baseUrl: localStorage.getItem('fitpulse-api-base') || '',
  production: localStorage.getItem('fitpulse-production') === '1',
};

function getAuthHeaders() {
  const token = localStorage.getItem('fitpulse-access-token');
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}

function getRefreshHeaders() {
  const token = localStorage.getItem('fitpulse-refresh-token');
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}

async function refreshAccessToken() {
  const refreshToken = localStorage.getItem('fitpulse-refresh-token');
  if (!refreshToken) return false;
  try {
    const res = await fetch(
      `${BACKEND_CONFIG.baseUrl.replace(/\/$/, '')}/api/v1/auth/refresh`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...getRefreshHeaders(),
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
      }
    );
    if (!res.ok) {
      localStorage.removeItem('fitpulse-refresh-token');
      return false;
    }
    const data = await res.json();
    if (data?.access_token) {
      localStorage.setItem('fitpulse-access-token', data.access_token);
      if (data?.refresh_token) {
        localStorage.setItem('fitpulse-refresh-token', data.refresh_token);
      }
      return true;
    }
    localStorage.removeItem('fitpulse-refresh-token');
    return false;
  } catch {
    localStorage.removeItem('fitpulse-refresh-token');
    return false;
  }
}

export async function backendRequest(path, options = {}) {
  if (!BACKEND_CONFIG.baseUrl) {
    return null;
  }
  const url = `${BACKEND_CONFIG.baseUrl.replace(/\/$/, '')}${path}`;
  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
      ...(options.headers || {}),
    },
  });
  if (res.status === 401) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      const retry = await fetch(url, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders(),
          ...(options.headers || {}),
        },
      });
      if (!retry.ok) {
        throw new Error(`Backend ${retry.status}`);
      }
      return retry.status === 204 ? null : retry.json();
    }
    localStorage.removeItem('fitpulse-access-token');
    localStorage.removeItem('fitpulse-refresh-token');
  }
  if (!res.ok) {
    throw new Error(`Backend ${res.status}`);
  }
  return res.status === 204 ? null : res.json();
}

export { BACKEND_CONFIG };
