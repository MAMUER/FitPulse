const KEY = 'fitpulse-merged-v9';

// Only JSON-safe, reasonably sized values are persisted. The state is assembled
// from the first-party API and local UI, but a compromised or buggy response must
// not be able to write arbitrary structures (functions, cycles, huge blobs) into
// localStorage, where they would be read back on the next start.
const MAX_DEPTH = 8;
const MAX_STRING_LENGTH = 100_000;

function isStorable(value, depth = 0) {
  if (value === null) return true;
  switch (typeof value) {
    case 'string':
      return value.length <= MAX_STRING_LENGTH;
    case 'number':
      return Number.isFinite(value);
    case 'boolean':
      return true;
    case 'object':
      break;
    default:
      return false;
  }
  if (depth >= MAX_DEPTH) return false;
  const items = Array.isArray(value) ? value : Object.values(value);
  return items.every((item) => isStorable(item, depth + 1));
}

export const loadState = () => {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return null;
    return JSON.parse(raw);
  } catch {
    return null;
  }
};

export const saveState = (state) => {
  try {
    if (!isStorable(state)) return;
    localStorage.setItem(KEY, JSON.stringify(state));
  } catch {
    // localStorage full or unavailable
  }
};
