package profanity

import "testing"

func TestContains(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"Михаил", false},
		{"Alice", false},
		{"", false},
		{"хуй", true},
		{"Х*У*Й", true},   // obfuscated
		{"f.u.c.k", true}, // obfuscated EN
		{"fuck", true},
		{"bitch", true},
		{"nice guy", false},
	}
	for _, tt := range tests {
		if got := Contains(tt.in); got != tt.want {
			t.Errorf("Contains(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
