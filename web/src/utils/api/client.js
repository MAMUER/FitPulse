const API_BASE = '/api/v1';

export function getAuthToken() {
  return localStorage.getItem('authToken');
}

export function setAuthToken(token) {
  if (token) {
    localStorage.setItem('authToken', token);
  } else {
    localStorage.removeItem('authToken');
  }
}

export async function apiRequest(endpoint, options = {}) {
  const token = getAuthToken();
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options.headers,
  };

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    setAuthToken(null);
    window.location.reload();
    throw new Error('Сессия истекла. Войдите заново');
  }

  if (response.status === 429) {
    const retryAfter = response.headers.get('Retry-After');
    throw new Error(
      retryAfter
        ? `Слишком много запросов. Повторите через ${retryAfter} сек.`
        : 'Слишком много запросов. Попробуйте через минуту.'
    );
  }

  const contentType = response.headers.get('content-type') || '';
  let data;
  if (contentType.includes('application/json')) {
    data = await response.json();
  } else {
    data = await response.text();
  }

  if (!response.ok) {
    const msg =
      typeof data === 'string'
        ? data
        : data?.message || data?.error || `Ошибка сервера (${response.status})`;
    throw new Error(msg);
  }

  return data;
}
