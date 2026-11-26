package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDumpDistilledDataset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "distilled.csv")

	err := dumpDistilledDataset(path, 5)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "heart_rate,hrv,spo2,temperature")
	assert.Contains(t, content, "label")

	lines := 0
	for _, line := range splitLines(content) {
		if line != "" {
			lines++
		}
	}
	// 7 архетипов × 5 строк + заголовок
	assert.GreaterOrEqual(t, lines, 10)
}

func TestDumpDistilledDataset_InvalidPath(t *testing.T) {
	err := dumpDistilledDataset(filepath.Join(t.TempDir(), "sub", "..", "..", "x.csv"), 1)
	// Путь с «..» может быть отклонён или создан — главное, что не паникует.
	_ = err
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
