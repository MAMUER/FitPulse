package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeIntToInt32(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int32
	}{
		{"positive value", 100, 100},
		{"zero", 0, 0},
		{"negative value", -100, -100},
		{"max int32", 2147483647, 2147483647},
		{"min int32", -2147483648, -2147483648},
		{"overflow positive", 2147483648, 2147483647},
		{"overflow negative", -2147483649, -2147483648},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeIntToInt32(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
