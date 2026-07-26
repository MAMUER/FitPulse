// Package sanitize provides input sanitization utilities for security.
package sanitize

import (
	"strings"
)

// String очищает строку от потенциально опасных символов для защиты от XSS
// и других инъекционных атак.
//
// Применяемые трансформации:
// - Trim пробельных символов
// - Экранирование HTML-тегов (< >)
// - Экранирование кавычек
// - Экранирование обратных слешей
//
// Примечание: это базовая защита. Для полной защиты от XSS рекомендуется:
// 1. Использовать библиотеку bluemonday для HTML-контента
// 2. Применять Content-Security-Policy заголовки
// 3. Кодировать данные при выводе на frontend
func String(s string) string {
	if s == "" {
		return s
	}

	s = strings.TrimSpace(s)
	// ВАЖНО: замена & должна быть ПЕРВОЙ, чтобы избежать double-encoding
	// Если сначала заменить < на &lt;, а потом & на &amp;, получится &amp;lt;
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, `'`, "&#39;")
	s = strings.ReplaceAll(s, `\`, `\\`)

	return s
}

// LogString removes log-forgery control characters from user-provided strings.
// It strips all ASCII control characters (0x00-0x1F, 0x7F) to prevent log injection,
// including newlines, carriage returns, tabs, and other non-printable characters.
func LogString(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7F {
			return -1
		}
		return r
	}, s)
}

// Strings очищает слайс строк
func Strings(items []string) []string {
	if items == nil {
		return nil
	}

	result := make([]string, len(items))
	for i, item := range items {
		result[i] = String(item)
	}
	return result
}

// MapStringString sanitizes all string values in a map.
// Returns a new map with sanitized values; original map is not modified.
func MapStringString(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = String(v)
	}
	return result
}

// MapStringInterface sanitizes string values in a map[string]interface{}.
// It recursively sanitizes nested maps and []string values.
// Non-string primitive values are returned as-is.
func MapStringInterface(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}

	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = String(val)
		case []string:
			sanitized := make([]string, len(val))
			for i, s := range val {
				sanitized[i] = String(s)
			}
			result[k] = sanitized
		case map[string]interface{}:
			result[k] = MapStringInterface(val)
		case map[string]string:
			result[k] = MapStringString(val)
		default:
			result[k] = v
		}
	}
	return result
}
