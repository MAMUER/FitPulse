// Package mocks provides mock biometric source for testing.
package mocks

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/MAMUER/project/internal/biometric/domain"
)

const stageStage = "stage"

type MockConfig struct {
	DeviceType       string
	NoiseLevel       float64
	GapProbability   float64
	DelayMin         time.Duration
	DelayMax         time.Duration
	FailureRate      float64
	SupportedMetrics map[domain.MetricType]bool
}

func DefaultMockConfig(deviceType string) MockConfig {
	caps := map[domain.MetricType]bool{
		domain.MetricHeartRate: true, domain.MetricHRV: true,
		domain.MetricSpO2: true, domain.MetricSteps: true,
	}
	switch deviceType {
	case "apple":
		caps[domain.MetricECG] = true
		caps[domain.MetricSleepStage] = true
	case "samsung":
		caps[domain.MetricECG] = true
		caps[domain.MetricSleepStage] = true
		caps[domain.MetricTemperature] = true
	case "huawei":
		caps[domain.MetricECG] = true
		caps[domain.MetricSleepStage] = true
		caps[domain.MetricTemperature] = true
		caps[domain.MetricBloodPressureSys] = true
		caps[domain.MetricBloodPressureDia] = true
	case "amazfit":
		caps[domain.MetricSleepStage] = true
	}
	return MockConfig{
		DeviceType: deviceType, NoiseLevel: 0.05,
		GapProbability: 0.02, DelayMin: 50 * time.Millisecond,
		DelayMax: 300 * time.Millisecond, FailureRate: 0.005,
		SupportedMetrics: caps,
	}
}

type userPhysioState struct {
	baseHR, baseHRV, baseSpO2, baseTemp float64
	baseBPSys, baseBPDia                float64
	activityLevel, fitnessLevel         float64
	stressLevel, sleepQuality           float64
}

type MockBiometricSource struct {
	config    MockConfig
	userID    string
	deviceID  string
	baseState *userPhysioState
}

func newMockBiometricSource(userID, deviceID string, config MockConfig) *MockBiometricSource {
	if config.SupportedMetrics == nil {
		config.SupportedMetrics = DefaultMockConfig(config.DeviceType).SupportedMetrics
	}
	return &MockBiometricSource{
		config: config,
		userID: userID, deviceID: deviceID,
		baseState: &userPhysioState{
			baseHR: 72, baseHRV: 50, baseSpO2: 98, baseTemp: 36.6,
			baseBPSys: 120, baseBPDia: 80,
			activityLevel: 0.5, fitnessLevel: 0.6,
			stressLevel: 0.3, sleepQuality: 0.7,
		},
	}
}

func NewMockBiometricSource(userID, deviceID, deviceType string) domain.BiometricSource {
	cfg := DefaultMockConfig(deviceType)
	cfg.DeviceType = deviceType
	return newMockBiometricSource(userID, deviceID, cfg)
}

func NewCustomMockBiometricSource(userID, deviceID string, config MockConfig) domain.BiometricSource {
	return newMockBiometricSource(userID, deviceID, config)
}

// secureFloat64 generates a cryptographically secure float64 in [0, 1)
func secureFloat64() float64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		// Fallback to a deterministic value if crypto/rand fails
		return 0.5
	}
	return float64(binary.LittleEndian.Uint64(b[:])) / float64(0x1_0000_0000_0000_0000)
}

func (m *MockBiometricSource) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	delay := time.Duration(int64(secureFloat64()*float64(m.config.DelayMax-m.config.DelayMin))) + m.config.DelayMin
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if secureFloat64() < m.config.FailureRate {
		return nil, fmt.Errorf("device unavailable")
	}
	now := time.Now()
	var samples []domain.BiometricSample
	for _, metric := range metricTypes {
		if !m.config.SupportedMetrics[domain.MetricType(metric)] {
			continue
		}
		if secureFloat64() < m.config.GapProbability {
			continue
		}
		value, unit := m.generateMetric(domain.MetricType(metric))
		value = m.addNoise(value)
		quality := "good"
		confidence := 0.9
		if secureFloat64() < 0.1 {
			quality = "fair"
			confidence = 0.7
		}
		samples = append(samples, domain.BiometricSample{
			UserID: m.userID, DeviceID: m.deviceID,
			DeviceType: m.config.DeviceType, MetricType: metric,
			Value: value, Unit: unit, Timestamp: now,
			Quality: quality, Confidence: confidence,
			SourceID: fmt.Sprintf("%s-%d", metric, now.UnixNano()),
			Metadata: map[string]interface{}{"simulated": true},
		})
	}
	return samples, nil
}

func (m *MockBiometricSource) generateMetric(metric domain.MetricType) (float64, string) {
	switch metric {
	case domain.MetricHeartRate:
		hr := m.baseState.baseHR + m.baseState.activityLevel*40 - m.baseState.fitnessLevel*10 + m.baseState.stressLevel*15
		return clamp(hr, 40, 180), "bpm"
	case domain.MetricHRV:
		hrv := m.baseState.baseHRV - m.baseState.stressLevel*20 + m.baseState.sleepQuality*10
		return clamp(hrv, 10, 100), "ms"
	case domain.MetricSpO2:
		spo2 := m.baseState.baseSpO2 + (secureFloat64()-0.5)*2
		return clamp(spo2, 90, 100), "%"
	case domain.MetricTemperature:
		temp := m.baseState.baseTemp + 0.3*math.Sin(2*math.Pi*float64(time.Now().Hour())/24) + (secureFloat64()-0.5)*0.2
		return clamp(temp, 35.5, 38.5), "C"
	case domain.MetricBloodPressureSys:
		sys := m.baseState.baseBPSys + m.baseState.stressLevel*15 + m.baseState.activityLevel*20
		return clamp(sys, 80, 200), "mmHg"
	case domain.MetricBloodPressureDia:
		dia := m.baseState.baseBPDia + m.baseState.stressLevel*10
		return clamp(dia, 50, 130), "mmHg"
	case domain.MetricECG:
		hr := clamp(m.baseState.baseHR+m.baseState.activityLevel*40, 40, 180)
		return 1.0 + (hr-70)/100, "mV"
	case domain.MetricSleepStage:
		h := time.Now().Hour()
		if h >= 23 || h < 7 {
			r := secureFloat64()
			if r < 0.1*m.baseState.sleepQuality {
				return 4, stageStage
			}
			if r < 0.5*m.baseState.sleepQuality {
				return 3, stageStage
			}
			if r < 0.7*m.baseState.sleepQuality {
				return 2, stageStage
			}
			return 1, stageStage
		}
		return 1, stageStage
	case domain.MetricSteps:
		h := time.Now().Hour()
		base := 50.0
		switch {
		case h >= 8 && h < 12:
			base = 500
		case h >= 12 && h < 18:
			base = 700
		case h >= 18 && h < 22:
			base = 300
		}
		return math.Floor(base * (0.5 + m.baseState.activityLevel) * (0.5 + secureFloat64())), "count"
	case domain.MetricDistance:
		return (m.baseState.activityLevel + 0.5) * 700, "m"
	case domain.MetricCalories:
		return 60.0 + m.baseState.activityLevel*400, "kcal"
	case domain.MetricRespiratoryRate:
		return math.Floor(12 + m.baseState.activityLevel*8 + (secureFloat64()-0.5)*4), "breaths/min"
	case domain.MetricBloodGlucose:
		return math.Round((90+secureFloat64()*20)*10) / 10, "mg/dL"
	case domain.MetricOxygenSaturation:
		spo2 := m.baseState.baseSpO2 + (secureFloat64()-0.5)*2
		return clamp(spo2, 90, 100), "%"
	default:
		return 0, ""
	}
}

func (m *MockBiometricSource) addNoise(v float64) float64 {
	u1, u2 := secureFloat64(), secureFloat64()
	if u1 == 0 {
		u1 = 1e-10
	}
	return v + m.config.NoiseLevel*math.Sqrt(-2*math.Log(u1))*math.Cos(2*math.Pi*u2)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m *MockBiometricSource) Supports(metricType string) bool {
	return m.config.SupportedMetrics[domain.MetricType(metricType)]
}

func (m *MockBiometricSource) DeviceType() string {
	return m.config.DeviceType
}

func (m *MockBiometricSource) HealthCheck(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

type Scenario int

const (
	ScenarioNormal Scenario = iota
	ScenarioHighStress
	ScenarioPoorSleep
	ScenarioLowFitness
	ScenarioRecovery
	ScenarioIllness
)

type ScenarioMock struct {
	*MockBiometricSource
	scenario Scenario
}

func NewScenarioMock(uid, did, dtype string, s Scenario) domain.BiometricSource {
	base := newMockBiometricSource(uid, did, DefaultMockConfig(dtype))
	switch s {
	case ScenarioNormal:
	case ScenarioHighStress:
		base.baseState.stressLevel = 0.8
		base.baseState.baseHR += 15
		base.baseState.baseHRV -= 15
	case ScenarioPoorSleep:
		base.baseState.sleepQuality = 0.3
		base.baseState.stressLevel = 0.6
		base.baseState.baseHR += 10
	case ScenarioLowFitness:
		base.baseState.fitnessLevel = 0.2
		base.baseState.baseHR += 20
		base.baseState.baseHRV -= 10
	case ScenarioRecovery:
		base.baseState.sleepQuality = 0.9
		base.baseState.stressLevel = 0.2
		base.baseState.baseBPSys = 110
		base.baseState.baseHRV = 65
	case ScenarioIllness:
		base.baseState.baseTemp = 38.2
		base.baseState.baseHR += 25
		base.baseState.baseSpO2 = 95
	}
	base.config.NoiseLevel = 0.1
	return &ScenarioMock{base, s}
}

func (s *ScenarioMock) Fetch(ctx context.Context, uid string, mt []string) ([]domain.BiometricSample, error) {
	samples, err := s.MockBiometricSource.Fetch(ctx, uid, mt)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range samples {
		samples[i].Timestamp = now
		if samples[i].Metadata == nil {
			samples[i].Metadata = map[string]interface{}{}
		}
		samples[i].Metadata["scenario"] = s.scenario
	}
	return samples, nil
}
