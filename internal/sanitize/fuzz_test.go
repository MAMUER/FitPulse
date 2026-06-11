package sanitize

import (
	"testing"
)

func FuzzString(f *testing.F) {
	f.Add("")
	f.Add("hello world")
	f.Add("<script>alert('xss')</script>")
	f.Add("rock & roll")
	f.Add("'; DROP TABLE users; --")
	f.Add("  <script>alert(\"xss\")</script>  & more")
	f.Add("\x00\r\n")
	f.Add("日本語 <テスト>")

	f.Fuzz(func(t *testing.T, input string) {
		result := String(input)
		if result == "" {
			return
		}
		if containsRunes(result, []rune{'>', '"', '\'', '&', '\\'}) {
			t.Errorf("unsanitized runes in result: %q", result)
		}
	})
}

func containsRunes(s string, rs []rune) bool {
	rm := map[rune]bool{}
	for _, r := range rs {
		rm[r] = true
	}
	for _, r := range s {
		if rm[r] {
			return true
		}
	}
	return false
}
