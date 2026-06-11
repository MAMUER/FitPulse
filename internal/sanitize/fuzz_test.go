package sanitize

import (
	"strings"
	"testing"
)

func FuzzString(f *testing.F) {
	f.Add("")
	f.Add("hello world")
	f.Add("Привет мир")
	f.Add("nl\n\r\t")
	f.Add("numbers 123456")
	f.Add("url https://example.com?q=test")

	f.Fuzz(func(t *testing.T, input string) {
		result := String(input)

		if strings.ContainsAny(result, `>"'\`) {
			t.Errorf("unsanitized characters in result: %q -> %q", input, result)
		}
	})
}
