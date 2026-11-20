// Package profanity provides a basic RU/EN profanity check for user-supplied
// names (nickname, full_name). It normalizes the input (lowercase, strips all
// non-letter characters) so obfuscated forms like "х*й" or "f.u.c.k" are caught.
// This is a baseline heuristic; for strict filtering a maintained wordlist/API
// is recommended.
package profanity

import "strings"

var patterns = []string{
	// RU
	"хуй", "хуё", "хуя", "бля", "еба", "еб", "пизд", "мудак", "гандон",
	"сука", "залуп", "долбо", "уеб", "выеб", "нахуй", "нах",
	// EN
	"fuck", "shit", "bitch", "asshole", "bastard", "dick", "cunt", "whore", "slut",
}

func normalize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'а' && r <= 'я') || r == 'ё' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Contains reports whether s contains profanity (RU/EN, obfuscation-tolerant).
func Contains(s string) bool {
	norm := normalize(s)
	if norm == "" {
		return false
	}
	for _, p := range patterns {
		if strings.Contains(norm, p) {
			return true
		}
	}
	return false
}
