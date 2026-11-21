package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// canonicalHeartRate — часть контракта датасета: ЧСС пользователя
// пересчитывается к опорному возрасту distillAge, чтобы rule-based и ML
// классификатор видели одну и ту же шкалу.
func TestCanonicalHeartRate(t *testing.T) {
	assert.InDelta(t, 100, canonicalHeartRate(100, 0), 1e-9)
	assert.InDelta(t, 100, canonicalHeartRate(100, 30), 1e-9)

	// 40 лет: HRmax = 180, опорный 190 → ЧСС 180 соответствует 190.
	assert.InDelta(t, 190, canonicalHeartRate(180, 40), 1e-9)

	// Некорректный возраст не должен ломать расчёт.
	assert.InDelta(t, 90, canonicalHeartRate(90, 500), 1e-9)
}

func TestDumpDatasetFlag(t *testing.T) {
	assert.Equal(t, "out.csv", dumpDatasetFlag([]string{"-dump-dataset", "out.csv"}))
	assert.Equal(t, "out.csv", dumpDatasetFlag([]string{"-dump-dataset=out.csv"}))
	assert.Empty(t, dumpDatasetFlag([]string{"--serve"}))
	assert.Empty(t, dumpDatasetFlag(nil))
}
