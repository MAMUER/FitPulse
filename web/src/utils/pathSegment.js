const PATH_SEGMENT = /^[A-Za-z0-9][A-Za-z0-9._~-]{0,127}$/;

// API path parameters come from persisted UI state and API responses, so they
// must be constrained to characters that cannot alter the path, query, or host.
// Leading alphanumeric also rejects "." / ".." and hidden dot-prefixed values.
export function isValidPathSegment(value) {
  return PATH_SEGMENT.test(String(value ?? ''));
}

export function pathSegment(value) {
  if (!isValidPathSegment(value)) {
    throw new Error('Invalid path parameter');
  }
  return encodeURIComponent(String(value));
}
