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
