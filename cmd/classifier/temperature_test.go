package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemperatureThresholds(t *testing.T) {
	t.Run("wrist without baseline", func(t *testing.T) {
		d := physiologicalData{TemperatureSource: "wrist"}
		sub, fever, high := d.temperatureThresholds()
		assert.InDelta(t, 35.8, sub, 1e-9)
		assert.InDelta(t, 36.0, fever, 1e-9)
		assert.InDelta(t, 36.5, high, 1e-9)
	})

	t.Run("core", func(t *testing.T) {
		d := physiologicalData{TemperatureSource: "core"}
		sub, fever, high := d.temperatureThresholds()
		assert.InDelta(t, 37.3, sub, 1e-9)
		assert.InDelta(t, 37.5, fever, 1e-9)
		assert.InDelta(t, 38.0, high, 1e-9)
	})

	t.Run("wrist with baseline", func(t *testing.T) {
		d := physiologicalData{TemperatureSource: "wrist", TemperatureBaseline: 33.5}
		sub, fever, high := d.temperatureThresholds()
		assert.InDelta(t, 33.9, sub, 1e-9)
		assert.InDelta(t, 34.2, fever, 1e-9)
		assert.InDelta(t, 34.7, high, 1e-9)
	})
}

func TestClassifyState_TemperatureSemantics(t *testing.T) {
	base := physiologicalData{
		HeartRate:            70,
		HeartRateVariability: 60,
		SpO2:                 98,
		SleepHours:           7,
	}

	t.Run("wrist 35.9 is subfebrile", func(t *testing.T) {
		data := base
		data.Temperature = 35.9
		data.TemperatureSource = "wrist"
		class, _, _ := classifyState(data, 30)
		assert.Equal(t, unknownClass, class)
	})

	t.Run("wrist 36.8 is illness", func(t *testing.T) {
		data := base
		data.Temperature = 36.8
		data.TemperatureSource = "wrist"
		class, _, _ := classifyState(data, 30)
		assert.Equal(t, 5, class)
	})

	t.Run("core 36.8 is normal", func(t *testing.T) {
		data := base
		data.Temperature = 36.8
		data.TemperatureSource = "core"
		class, _, _ := classifyState(data, 30)
		assert.NotEqual(t, unknownClass, class)
	})

	t.Run("wrist baseline-relative", func(t *testing.T) {
		data := base
		data.TemperatureSource = "wrist"
		data.TemperatureBaseline = 33.0
		data.Temperature = 34.4 // baseline + 1.4 -> выше high-порога
		class, _, _ := classifyState(data, 30)
		assert.Equal(t, 5, class, "болезнь при отклонении от базлайна")
	})
}

func TestResolveTemperatureSource(t *testing.T) {
	t.Run("empty source defaults to wrist", func(t *testing.T) {
		assert.Equal(t, "wrist", resolveTemperatureSource(physiologicalData{Temperature: 36.6}))
		assert.Equal(t, "wrist", resolveTemperatureSource(physiologicalData{}))
	})

	t.Run("explicit source wins", func(t *testing.T) {
		assert.Equal(t, "core", resolveTemperatureSource(physiologicalData{Temperature: 36.6, TemperatureSource: "core"}))
		assert.Equal(t, "wrist", resolveTemperatureSource(physiologicalData{Temperature: 33.5, TemperatureSource: "wrist"}))
	})
}

func TestDefaultTemperature(t *testing.T) {
	assert.InDelta(t, 33.5, defaultTemperature("wrist"), 1e-9)
	assert.InDelta(t, 33.5, defaultTemperature(""), 1e-9)
	assert.InDelta(t, 37.0, defaultTemperature("core"), 1e-9)
}

func TestClassifyState_MissingTemperatureIsNotFever(t *testing.T) {
	data := physiologicalData{
		HeartRate:              62,
		HeartRateVariability:   55,
		SpO2:                   98,
		SleepHours:             7.5,
		BloodPressureSystolic:  118,
		BloodPressureDiastolic: 76,
	}
	data.TemperatureSource = resolveTemperatureSource(data)
	data = applyPhysiologicalDefaults(data)

	class, _, _ := classifyState(data, 30)
	assert.NotEqual(t, 5, class, "отсутствие температуры не должно давать illness")
	assert.NotEqual(t, unknownClass, class, "отсутствие температуры не должно давать unknown")
}

func TestMissingMetricsAndQuality(t *testing.T) {
	t.Run("nothing missing", func(t *testing.T) {
		d := physiologicalData{
			HeartRate:              70,
			HeartRateVariability:   50,
			SpO2:                   98,
			Temperature:            36.6,
			SleepHours:             7,
			BloodPressureSystolic:  120,
			BloodPressureDiastolic: 80,
		}
		assert.Empty(t, missingMetrics(d))
		assert.Equal(t, "high", dataQuality(nil))
	})

	t.Run("only heart rate", func(t *testing.T) {
		d := physiologicalData{HeartRate: 120}
		missing := missingMetrics(d)
		assert.Contains(t, missing, "hrv")
		assert.Contains(t, missing, "spo2")
		assert.Contains(t, missing, "temperature")
		assert.Contains(t, missing, "sleep_hours")
		assert.Equal(t, "low", dataQuality(missing))
	})

	t.Run("penalty lowers confidence", func(t *testing.T) {
		assert.InDelta(t, 0.5, applyQualityPenalty(0.5, "high"), 1e-9)
		assert.InDelta(t, 0.45, applyQualityPenalty(0.5, "medium"), 1e-9)
		assert.InDelta(t, 0.4, applyQualityPenalty(0.5, "low"), 1e-9)
	})
}
