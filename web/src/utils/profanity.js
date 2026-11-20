// Базовая проверка на нецензурные/оскорбительные слова в имени.
// Нормализация (нижний регистр + удаление всех не-буквенных символов) ловит
// обфускацию вида «х*й», «f.u.c.k», «h-i-t». Для строгой проверки нужен
// бэкенд-словарь (см. docs), здесь — клиентская эвристика.

const PROFANITY_PATTERNS = [
  // RU
  'хуй',
  'хуё',
  'хуя',
  'бля',
  'еба',
  'еб',
  'пизд',
  'мудак',
  'гандон',
  'сука',
  'залуп',
  'долбо',
  'уеб',
  'выеб',
  'нахуй',
  'нах',
  // EN
  'fuck',
  'shit',
  'bitch',
  'asshole',
  'bastard',
  'dick',
  'cunt',
  'whore',
  'slut',
];

function normalize(s) {
  return String(s || '')
    .toLowerCase()
    .replace(/[^a-zа-яё]/gi, '');
}

export function containsProfanity(text) {
  const norm = normalize(text);
  if (!norm) return false;
  return PROFANITY_PATTERNS.some((w) => norm.includes(w));
}
