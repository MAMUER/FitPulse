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
	input := "<script> & </script>"
	result := String(input)

	assert.NotContains(t, result, "&amp;lt;")
	assert.NotContains(t, result, "&amp;gt;")
	assert.Contains(t, result, "&lt;script&gt;")
	assert.Contains(t, result, "&amp;")
	assert.Contains(t, result, "&lt;/script&gt;")
}

func TestString_NonIdempotent(t *testing.T) {
	input := "hello & <world>"
	first := String(input)
	second := String(first)

	assert.NotEqual(t, first, second, "repeated sanitization should further escape & characters")
}

func TestLogString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "newline",
			input:    "hello\nworld",
			expected: "helloworld",
		},
		{
			name:     "carriage return",
			input:    "hello\rworld",
			expected: "helloworld",
		},
		{
			name:     "tab",
			input:    "hello\tworld",
			expected: "helloworld",
		},
		{
			name:     "mixed control chars",
			input:    "hello\n\r\tworld",
			expected: "helloworld",
		},
		{
			name:     "null byte",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "bell character",
			input:    "hello\x07world",
			expected: "helloworld",
		},
		{
			name:     "delete character",
			input:    "hello\x7Fworld",
			expected: "helloworld",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only control chars",
			input:    "\n\r\t\x00\x7F",
			expected: "",
		},
		{
			name:     "unicode preserved",
			input:    "Привет\nмир",
			expected: "Приветмир",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LogString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapStringString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "sanitize values",
			input: map[string]string{
				"key1": "<script>alert(1)</script>",
				"key2": "normal",
				"key3": "  spaced  ",
			},
			expected: map[string]string{
				"key1": "&lt;script&gt;alert(1)&lt;/script&gt;",
				"key2": "normal",
				"key3": "spaced",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapStringString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapStringInterface(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "nested sanitization",
			input: map[string]interface{}{
				"name":  "<b>test</b>",
				"count": 42,
				"tags":  []string{"<xss>", "safe"},
				"nested": map[string]interface{}{
					"value": "<script>alert(1)</script>",
				},
			},
			expected: map[string]interface{}{
				"name":  "&lt;b&gt;test&lt;/b&gt;",
				"count": 42,
				"tags":  []string{"&lt;xss&gt;", "safe"},
				"nested": map[string]interface{}{
					"value": "&lt;script&gt;alert(1)&lt;/script&gt;",
				},
			},
		},
		{
			name: "non-string primitives preserved",
			input: map[string]interface{}{
				"int":    42,
				"float":  3.14,
				"bool":   true,
				"string": "<xss>",
			},
			expected: map[string]interface{}{
				"int":    42,
				"float":  3.14,
				"bool":   true,
				"string": "&lt;xss&gt;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapStringInterface(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestString_WhitespaceOnly(t *testing.T) {
	result := String("   \t\n  ")
	assert.Equal(t, "", result)
}

func TestLogString_Empty(t *testing.T) {
	result := LogString("")
	assert.Equal(t, "", result)
}

func BenchmarkString(b *testing.B) {
	input := "<script>alert('xss')</script> & more content here"
	for b.Loop() {
		_ = String(input)
	}
}

func BenchmarkStrings(b *testing.B) {
	input := []string{"<b>bold</b>", "rock & roll", "normal text", "<script>x</script>"}
	for b.Loop() {
		_ = Strings(input)
	}
}

func BenchmarkLogString(b *testing.B) {
	input := "user input with\nnewlines\r\nand\ttabs"
	for b.Loop() {
		_ = LogString(input)
	}
}
