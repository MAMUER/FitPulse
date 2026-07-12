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
	f.Add(`<script>alert("xss")</script>`)
	f.Add(`'; DROP TABLE users; --`)
	f.Add(`path\to\file`)
	f.Add(`rock & roll`)
	f.Add(`<<nested>>`)
	f.Add(`  spaces  `)

	f.Fuzz(func(t *testing.T, input string) {
		result := String(input)
		if strings.ContainsAny(result, `<>"'`) {
			t.Errorf("raw dangerous character in result: %q -> %q", input, result)
		}

		// Result should not exceed a reasonable expansion bound.
		// Each input char expands to at most 6 chars (&quot; = 6 chars).
		// Plus trimming, so result <= len(input)*6 is a safe upper bound.
		if len(input) > 0 && len(result) > len(input)*6+10 {
			t.Errorf("result unexpectedly long: input=%d, result=%d", len(input), len(result))
		}

		// Empty input should produce empty output
		if input == "" && result != "" {
			t.Errorf("empty input produced non-empty result: %q", result)
		}

		// Result should not start or end with whitespace (trimmed)
		if result != strings.TrimSpace(result) {
			t.Errorf("result has leading/trailing whitespace: %q", result)
		}
	})
}
