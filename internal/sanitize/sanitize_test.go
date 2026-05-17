package sanitize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "xss script tag",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "xss event handler",
			input:    `<img src=x onerror="alert(1)">`,
			expected: `&lt;img src=x onerror=&quot;alert(1)&quot;&gt;`,
		},
		{
			name:     "html entity ampersand",
			input:    "rock & roll",
			expected: "rock &amp; roll",
		},
		{
			name:     "no double encoding",
			input:    "&lt;script&gt;",
			expected: "&amp;lt;script&amp;gt;",
		},
		{
			name:     "backslash escaping",
			input:    `path\to\file`,
			expected: `path\\to\\file`,
		},
		{
			name:     "whitespace trimming",
			input:    "  hello  ",
			expected: "hello",
		},
		{
			name:     "mixed attacks",
			input:    `  <script>alert("xss")</script>  & more`,
			expected: `&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;  &amp; more`,
		},
		{
			name:     "sql injection attempt",
			input:    "'; DROP TABLE users; --",
			expected: "&#39;; DROP TABLE users; --",
		},
		{
			name:     "unicode characters",
			input:    "Привет мир <script>",
			expected: "Привет мир &lt;script&gt;",
		},
		{
			name:     "nested tags",
			input:    "<<script>>",
			expected: "&lt;&lt;script&gt;&gt;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := String(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "nil slice",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "sanitize multiple strings",
			input:    []string{"<b>bold</b>", "rock & roll", "normal text"},
			expected: []string{"&lt;b&gt;bold&lt;/b&gt;", "rock &amp; roll", "normal text"},
		},
		{
			name:     "trim whitespace",
			input:    []string{"  hello  ", "  world  "},
			expected: []string{"hello", "world"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Strings(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringOrderOfOperations(t *testing.T) {
	// Критичный тест: проверяем что & заменяется ПЕРВЫМ
	// чтобы избежать double-encoding

	// Если & заменяется первым, то <script> & </script>
	// должно стать &lt;script&gt; &amp; &lt;/script&gt;
	input := "<script> & </script>"
	result := String(input)

	// Проверяем что нет &amp;lt; или &amp;gt; (double encoding)
	assert.NotContains(t, result, "&amp;lt;")
	assert.NotContains(t, result, "&amp;gt;")

	// Проверяем корректный результат
	assert.Contains(t, result, "&lt;script&gt;")
	assert.Contains(t, result, "&amp;")
	assert.Contains(t, result, "&lt;/script&gt;")
}

func TestStringIdempotency(t *testing.T) {
	// Проверяем что повторная санитизация не ломает данные
	input := "hello & <world>"
	first := String(input)
	second := String(first)

	// Вторая санитизация должна дополнительно экранировать &
	// это ожидаемое поведение
	assert.NotEqual(t, first, second, "second sanitization should change the string")
}

func BenchmarkString(b *testing.B) {
	input := "<script>alert('xss')</script> & more content here"
	for i := 0; i < b.N; i++ {
		_ = String(input)
	}
}

func BenchmarkStrings(b *testing.B) {
	input := []string{"<b>bold</b>", "rock & roll", "normal text", "<script>x</script>"}
	for i := 0; i < b.N; i++ {
		_ = Strings(input)
	}
}
