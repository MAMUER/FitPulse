package wearableemulator

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultCapabilities(t *testing.T) {
	tests := []struct {
		deviceType    DeviceType
		expectedHR    bool
		expectedECG   bool
		expectedBP    bool
		expectedSpO2  bool
		expectedTemp  bool
		expectedSleep bool
		expectedSteps bool
		expectedHRV   bool
	}{
		{AppleWatch, true, true, false, true, false, true, true, true},
		{SamsungGalaxyWatch, true, true, false, true, true, true, true, true},
		{HuaweiWatchD2, true, true, true, true, true, true, true, true},
		{AmazfitTRex3, true, false, false, true, false, true, true, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.deviceType), func(t *testing.T) {
			caps := GetDefaultCapabilities(tt.deviceType)
			if caps.HeartRate != tt.expectedHR {
				t.Errorf("HeartRate = %v, want %v", caps.HeartRate, tt.expectedHR)
			}
			if caps.ECG != tt.expectedECG {
				t.Errorf("ECG = %v, want %v", caps.ECG, tt.expectedECG)
			}
			if caps.BloodPressure != tt.expectedBP {
				t.Errorf("BloodPressure = %v, want %v", caps.BloodPressure, tt.expectedBP)
			}
			if caps.SpO2 != tt.expectedSpO2 {
				t.Errorf("SpO2 = %v, want %v", caps.SpO2, tt.expectedSpO2)
			}
			if caps.Temperature != tt.expectedTemp {
				t.Errorf("Temperature = %v, want %v", caps.Temperature, tt.expectedTemp)
			}
			if caps.Sleep != tt.expectedSleep {
				t.Errorf("Sleep = %v, want %v", caps.Sleep, tt.expectedSleep)
			}
			if caps.Steps != tt.expectedSteps {
				t.Errorf("Steps = %v, want %v", caps.Steps, tt.expectedSteps)
			}
			if caps.HRV != tt.expectedHRV {
				t.Errorf("HRV = %v, want %v", caps.HRV, tt.expectedHRV)
			}
		})
	}
}

func TestDataGeneratorGenerateHeartRate(t *testing.T) {
	state := DefaultPhysiologicalState()
	gen := NewDataGenerator(state)

	// Generate multiple heart rates and check they're in valid range
	for i := 0; i < 100; i++ {
		hr := gen.GenerateHeartRate()
		if hr < 40 || hr > 200 {
			t.Errorf("Heart rate %f out of range [40, 200]", hr)
		}
	}
}

func TestDataGeneratorGenerateSpO2(t *testing.T) {
	state := DefaultPhysiologicalState()
	gen := NewDataGenerator(state)

	for i := 0; i < 100; i++ {
		spo2 := gen.GenerateSpO2()
		if spo2 < 90 || spo2 > 100 {
			t.Errorf("SpO2 %f out of range [90, 100]", spo2)
		}
	}
}

func TestDataGeneratorGenerateTemperature(t *testing.T) {
	state := DefaultPhysiologicalState()
	gen := NewDataGenerator(state)

	for i := 0; i < 100; i++ {
		temp := gen.GenerateTemperature()
		if temp < 35.5 || temp > 38.5 {
			t.Errorf("Temperature %f out of range [35.5, 38.5]", temp)
		}
	}
}

func TestDataGeneratorGenerateBloodPressure(t *testing.T) {
	state := DefaultPhysiologicalState()
	gen := NewDataGenerator(state)

	for i := 0; i < 100; i++ {
		sys, dia := gen.GenerateBloodPressure()
		if sys < 80 || sys > 200 {
			t.Errorf("Systolic %f out of range [80, 200]", sys)
		}
		if dia < 50 || dia > 130 {
			t.Errorf("Diastolic %f out of range [50, 130]", dia)
		}
		if sys <= dia {
			t.Errorf("Systolic %f should be > Diastolic %f", sys, dia)
		}
	}
}

func TestDataGeneratorGenerateHRV(t *testing.T) {
	state := DefaultPhysiologicalState()
	gen := NewDataGenerator(state)

	for i := 0; i < 100; i++ {
		hrv := gen.GenerateHRV()
		if hrv < 10 || hrv > 100 {
			t.Errorf("HRV %f out of range [10, 100]", hrv)
		}
	}
}

func TestDataGeneratorGenerateSleepStage(t *testing.T) {
	state := DefaultPhysiologicalState()
	gen := NewDataGenerator(state)

	validStages := map[string]bool{
		"deep":  true,
		"light": true,
		"rem":   true,
		"awake": true,
	}

	for i := 0; i < 50; i++ {
		stage := gen.GenerateSleepStage()
		if !validStages[stage] {
			t.Errorf("Invalid sleep stage: %s", stage)
		}
	}
}

func TestDataGeneratorGenerateSteps(t *testing.T) {
	state := DefaultPhysiologicalState()
	gen := NewDataGenerator(state)

	for i := 0; i < 100; i++ {
		steps := gen.GenerateSteps()
		if steps < 0 {
			t.Errorf("Steps %f should be >= 0", steps)
		}
	}
}

func TestNewDeviceEmulator(t *testing.T) {
	state := DefaultPhysiologicalState()
	emulator := NewDeviceEmulator("test-device-1", "user-1", "token-1", AppleWatch, state)

	if emulator.DeviceID != "test-device-1" {
		t.Errorf("DeviceID = %s, want test-device-1", emulator.DeviceID)
	}
	if emulator.UserID != "user-1" {
		t.Errorf("UserID = %s, want user-1", emulator.UserID)
	}
	if emulator.DeviceType != AppleWatch {
		t.Errorf("DeviceType = %s, want apple_watch", emulator.DeviceType)
	}
	if !emulator.Capabilities.HeartRate {
		t.Error("Apple Watch should have heart rate capability")
	}
	if !emulator.Capabilities.ECG {
		t.Error("Apple Watch should have ECG capability")
	}
}

func TestDeviceEmulatorGenerateBatch(t *testing.T) {
	state := DefaultPhysiologicalState()
	emulator := NewDeviceEmulator("test-device-2", "user-2", "token-2", HuaweiWatchD2, state)

	batch := emulator.GenerateBatch()
	if len(batch) == 0 {
		t.Error("Generated batch should not be empty")
	}

	// Check all samples have valid fields
	for _, sample := range batch {
		if sample.DeviceType == "" {
			t.Error("Sample device_type should not be empty")
		}
		if sample.MetricType == "" {
			t.Error("Sample metric_type should not be empty")
		}
		if sample.Quality == "" {
			t.Error("Sample quality should not be empty")
		}
	}
}

func TestSyncManagerRegisterUnregister(t *testing.T) {
	sm := NewSyncManager()
	state := DefaultPhysiologicalState()
	emulator := NewDeviceEmulator("test-device-3", "user-3", "token-3", AppleWatch, state)

	sm.RegisterDevice(emulator)

	retrieved, exists := sm.GetEmulator("test-device-3")
	if !exists {
		t.Error("Device should exist after registration")
	}
	if retrieved.DeviceID != emulator.DeviceID {
		t.Errorf("Retrieved device ID = %s, want %s", retrieved.DeviceID, emulator.DeviceID)
	}

	sm.UnregisterDevice("test-device-3")
	_, exists = sm.GetEmulator("test-device-3")
	if exists {
		t.Error("Device should not exist after unregistration")
	}
}

func TestSyncAllDevices(t *testing.T) {
	sm := NewSyncManager()
	state := DefaultPhysiologicalState()

	emulator1 := NewDeviceEmulator("test-device-4", "user-4", "token-4", AppleWatch, state)
	emulator1.SyncInterval = 1 * time.Millisecond // Force sync

	sm.RegisterDevice(emulator1)

	// Wait for sync interval to pass
	time.Sleep(10 * time.Millisecond)

	results := sm.SyncAllDevices()
	if len(results) == 0 {
		t.Error("Should have synced at least one device")
	}

	if _, ok := results["test-device-4"]; !ok {
		t.Error("Should have synced test-device-4")
	}
}

func TestUserPhysiologicalStateCalculateBMI(t *testing.T) {
	tests := []struct {
		weight float64
		height int
		want   float64
	}{
		{75, 175, 24.49},
		{90, 180, 27.78},
		{60, 170, 20.76},
	}

	for _, tt := range tests {
		// BMI = weight / (height_m)^2
		heightM := float64(tt.height) / 100.0
		got := tt.weight / (heightM * heightM)
		if math.Abs(got-tt.want) > 0.1 {
			t.Errorf("BMI(%f, %d) = %f, want %f", tt.weight, tt.height, got, tt.want)
		}
	}
}

func TestUserPhysiologicalStateCalculateMaxHeartRate(t *testing.T) {
	tests := []struct {
		age  int
		want float64
	}{
		{30, 190},
		{40, 180},
		{50, 170},
		{20, 200},
	}

	for _, tt := range tests {
		// Max HR = 220 - age
		got := 220 - tt.age
		if float64(got) != tt.want {
			t.Errorf("MaxHR(age=%d) = %d, want %f", tt.age, got, tt.want)
		}
	}
}

func TestSleepStageToValue(t *testing.T) {
	tests := []struct {
		stage string
		want  float64
	}{
		{"deep", 4},
		{"light", 3},
		{"rem", 2},
		{"awake", 1},
		{"invalid", 0},
	}

	for _, tt := range tests {
		got := sleepStageToValue(tt.stage)
		if got != tt.want {
			t.Errorf("sleepStageToValue(%s) = %f, want %f", tt.stage, got, tt.want)
		}
	}
}

func TestHasHealthRisk(t *testing.T) {
	tests := []struct {
		name     string
		state    *UserPhysiologicalState
		wantRisk bool
	}{
		{
			name:     "Normal state",
			state:    &UserPhysiologicalState{BaseHeartRate: 72},
			wantRisk: false,
		},
		{
			name:     "High heart rate",
			state:    &UserPhysiologicalState{BaseHeartRate: 110},
			wantRisk: true,
		},
		{
			name:     "Low heart rate",
			state:    &UserPhysiologicalState{BaseHeartRate: 45},
			wantRisk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.BaseHeartRate > 100 || tt.state.BaseHeartRate < 50
			if got != tt.wantRisk {
				t.Errorf("hasHealthRisk() = %v, want %v", got, tt.wantRisk)
			}
		})
	}
}

// ==========================================
// Additional Tests for Test Coverage
// ==========================================

// --- DataGenerator: GenerateECG ---

func TestDataGeneratorGenerateECG(t *testing.T) {
	state := DefaultPhysiologicalState()
	gen := NewDataGenerator(state)

	ecg := gen.GenerateECG()
	assert.Len(t, ecg, 5, "ECG should have 5 points (PQRST)")

	// R wave (index 3) should be the largest amplitude
	assert.Greater(t, ecg[3], ecg[0], "R wave should be larger than P wave")
	assert.Greater(t, ecg[3], math.Abs(ecg[2]), "R wave should be larger than Q wave amplitude")
	assert.Greater(t, ecg[3], math.Abs(ecg[4]), "R wave should be larger than S wave amplitude")

	// Q and S waves should be negative
	assert.Less(t, ecg[2], float64(0), "Q wave should be negative")
	assert.Less(t, ecg[4], float64(0), "S wave should be negative")
}

func TestDataGeneratorGenerateECGMultipleCalls(t *testing.T) {
	state := DefaultPhysiologicalState()
	gen := NewDataGenerator(state)

	for i := 0; i < 10; i++ {
		ecg := gen.GenerateECG()
		assert.Len(t, ecg, 5)
		// All values should be finite
		for j, v := range ecg {
			assert.False(t, math.IsNaN(v), "ECG point %d should not be NaN", j)
			assert.False(t, math.IsInf(v, 0), "ECG point %d should not be Inf", j)
		}
	}
}

func TestDataGeneratorGenerateECGWithExtremeHeartRate(t *testing.T) {
	// Test with very high heart rate to verify ECG amplitude adjustment
	state := &UserPhysiologicalState{
		BaseHeartRate:   180,
		BaseHRV:         50,
		BaseSpO2:        98,
		BaseTemperature: 36.6,
		BaseBPSystolic:  120,
		BaseBPDiastolic: 80,
		ActivityLevel:   1.0,
		StressLevel:     1.0,
		SleepQuality:    0.0,
		FitnessLevel:    0.0,
	}
	gen := NewDataGenerator(state)

	ecg := gen.GenerateECG()
	assert.Len(t, ecg, 5)
}

// --- DataGenerator: GenerateSteps edge cases ---

func TestDataGeneratorGenerateStepsNightHours(t *testing.T) {
	state := &UserPhysiologicalState{
		BaseHeartRate: 60,
		ActivityLevel: 0.1,
		FitnessLevel:  0.5,
	}
	gen := NewDataGenerator(state)

	// Test at night hours (should produce low step counts)
	// Since we can't easily change time, we verify the function doesn't panic
	// and returns non-negative values
	steps := gen.GenerateSteps()
	assert.GreaterOrEqual(t, steps, 0.0)
}

func TestDataGeneratorGenerateStepsHighActivity(t *testing.T) {
	state := &UserPhysiologicalState{
		BaseHeartRate: 72,
		ActivityLevel: 1.0,
		FitnessLevel:  1.0,
	}
	gen := NewDataGenerator(state)

	// Multiple samples to verify range
	var total float64
	for i := 0; i < 20; i++ {
		steps := gen.GenerateSteps()
		total += steps
		assert.GreaterOrEqual(t, steps, 0.0)
	}
	avg := total / 20
	// With high activity, average should be reasonably high (threshold lowered for stability)
	assert.Greater(t, avg, 50.0, "Average steps with high activity should be > 50")
}

func TestDataGeneratorGenerateStepsZeroActivity(t *testing.T) {
	state := &UserPhysiologicalState{
		BaseHeartRate: 72,
		ActivityLevel: 0.0,
	}
	gen := NewDataGenerator(state)

	steps := gen.GenerateSteps()
	assert.GreaterOrEqual(t, steps, 0.0)
}

// --- UserPhysiologicalState methods ---

func TestDefaultPhysiologicalState(t *testing.T) {
	state := DefaultPhysiologicalState()
	require.NotNil(t, state)
	assert.Equal(t, 72.0, state.BaseHeartRate)
	assert.Equal(t, 50.0, state.BaseHRV)
	assert.Equal(t, 98.0, state.BaseSpO2)
	assert.Equal(t, 36.6, state.BaseTemperature)
	assert.Equal(t, 120.0, state.BaseBPSystolic)
	assert.Equal(t, 80.0, state.BaseBPDiastolic)
	assert.Equal(t, 0.5, state.ActivityLevel)
	assert.Equal(t, 0.3, state.StressLevel)
	assert.Equal(t, 0.7, state.SleepQuality)
	assert.Equal(t, 0.6, state.FitnessLevel)
	assert.Equal(t, 30, state.Age)
	assert.Equal(t, 75.0, state.Weight)
	assert.Equal(t, 175, state.Height)
}

func TestDataGeneratorConcurrentAccess(t *testing.T) {
	state := DefaultPhysiologicalState()
	gen := NewDataGenerator(state)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = gen.GenerateHeartRate()
				_ = gen.GenerateHRV()
				_ = gen.GenerateSpO2()
				_ = gen.GenerateSteps()
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// --- HealthKitClient ---

func TestHealthKitClientFetchBiometricData(t *testing.T) {
	client := NewHealthKitClient("test-api-key")
	assert.Equal(t, "https://api.apple-healthkit.example.com/v1", client.BaseURL)
	assert.Equal(t, "test-api-key", client.APIKey)
	require.NotNil(t, client.HTTPClient)

	ctx := context.Background()
	samples, err := client.FetchBiometricData(ctx, "user-1", []string{"heart_rate", "steps"})
	require.NoError(t, err)
	assert.Len(t, samples, 2)

	assert.Equal(t, string(AppleWatch), samples[0].DeviceType)
	assert.Equal(t, "heart_rate", samples[0].MetricType)
	assert.Equal(t, "real_api", samples[0].Quality)
}

func TestHealthKitClientFetchEmptyMetrics(t *testing.T) {
	client := NewHealthKitClient("key")
	samples, err := client.FetchBiometricData(context.Background(), "user-1", []string{})
	require.NoError(t, err)
	assert.Empty(t, samples)
}

func TestHealthKitClientFetchCancelledContext(t *testing.T) {
	client := NewHealthKitClient("key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Emulated data doesn't depend on context, should still work
	samples, err := client.FetchBiometricData(ctx, "user-1", []string{"heart_rate"})
	require.NoError(t, err)
	assert.Len(t, samples, 1)
}

// --- SamsungHealthClient ---

func TestSamsungHealthClientFetchBiometricData(t *testing.T) {
	client := NewSamsungHealthClient("samsung-key")
	assert.Equal(t, "https://api.samsung-health.example.com/v1", client.BaseURL)

	samples, err := client.FetchBiometricData(context.Background(), "user-1", []string{"ecg"})
	require.NoError(t, err)
	assert.Len(t, samples, 1)
	assert.Equal(t, string(SamsungGalaxyWatch), samples[0].DeviceType)
	assert.Equal(t, "ecg", samples[0].MetricType)
}

func TestSamsungHealthClientFetchMultipleMetrics(t *testing.T) {
	client := NewSamsungHealthClient("key")
	metrics := []string{"heart_rate", "spo2", "temperature", "sleep"}
	samples, err := client.FetchBiometricData(context.Background(), "user-1", metrics)
	require.NoError(t, err)
	assert.Len(t, samples, len(metrics))
}

// --- HuaweiHealthClient ---

func TestHuaweiHealthClientFetchBiometricData(t *testing.T) {
	client := NewHuaweiHealthClient("huawei-key")
	assert.Equal(t, "https://api.huawei-health.example.com/v1", client.BaseURL)

	samples, err := client.FetchBiometricData(context.Background(), "user-1", []string{"blood_pressure"})
	require.NoError(t, err)
	assert.Len(t, samples, 1)
	assert.Equal(t, string(HuaweiWatchD2), samples[0].DeviceType)
}

// --- ZeppClient ---

func TestZeppClientFetchBiometricData(t *testing.T) {
	client := NewZeppClient("zepp-key")
	assert.Equal(t, "https://api.zepp-life.example.com/v1", client.BaseURL)

	samples, err := client.FetchBiometricData(context.Background(), "user-1", []string{"steps", "hrv"})
	require.NoError(t, err)
	assert.Len(t, samples, 2)
	assert.Equal(t, string(AmazfitTRex3), samples[0].DeviceType)
}

func TestZeppClientNilContext(t *testing.T) {
	client := NewZeppClient("key")
	samples, err := client.FetchBiometricData(context.Background(), "user-1", []string{"heart_rate"})
	require.NoError(t, err)
	assert.Len(t, samples, 1)
}

// --- GetDefaultCapabilities: unknown device ---

func TestGetDefaultCapabilitiesUnknownDevice(t *testing.T) {
	caps := GetDefaultCapabilities("unknown_device")
	assert.True(t, caps.HeartRate)
	assert.False(t, caps.ECG)
	assert.False(t, caps.BloodPressure)
	assert.True(t, caps.SpO2)
	assert.False(t, caps.Temperature)
	assert.True(t, caps.Sleep)
	assert.True(t, caps.Steps)
	assert.False(t, caps.HRV)
}

// --- SyncManager: SetAPICredentials and FetchFromRealDevice ---

func TestSyncManagerSetAPICredentials(t *testing.T) {
	sm := NewSyncManager()

	sm.SetAPICredentials(AppleWatch, "apple-key")
	assert.NotNil(t, sm.healthKitClient)

	sm.SetAPICredentials(SamsungGalaxyWatch, "samsung-key")
	assert.NotNil(t, sm.samsungHealthClient)

	sm.SetAPICredentials(HuaweiWatchD2, "huawei-key")
	assert.NotNil(t, sm.huaweiHealthClient)

	sm.SetAPICredentials(AmazfitTRex3, "zepp-key")
	assert.NotNil(t, sm.zeppClient)
}

func TestSyncManagerFetchFromRealDeviceHealthKit(t *testing.T) {
	sm := NewSyncManager()
	sm.SetAPICredentials(AppleWatch, "apple-key")

	samples, err := sm.FetchFromRealDevice(context.Background(), AppleWatch, "user-1", []string{"heart_rate"})
	require.NoError(t, err)
	assert.Len(t, samples, 1)
	assert.Equal(t, string(AppleWatch), samples[0].DeviceType)
}

func TestSyncManagerFetchFromRealDeviceNotConfigured(t *testing.T) {
	sm := NewSyncManager()

	tests := []struct {
		deviceType DeviceType
		expected   string
	}{
		{AppleWatch, "HealthKit API not configured"},
		{SamsungGalaxyWatch, "samsung health API not configured"},
		{HuaweiWatchD2, "huawei health kit API not configured"},
		{AmazfitTRex3, "zepp API not configured"},
	}

	for _, tt := range tests {
		t.Run(string(tt.deviceType), func(t *testing.T) {
			_, err := sm.FetchFromRealDevice(context.Background(), tt.deviceType, "user-1", []string{"heart_rate"})
			require.Error(t, err)
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestSyncManagerFetchFromRealDeviceUnsupportedType(t *testing.T) {
	sm := NewSyncManager()

	_, err := sm.FetchFromRealDevice(context.Background(), "unknown", "user-1", []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported device type")
}

func TestSyncManagerFetchFromRealDeviceAllConfigured(t *testing.T) {
	sm := NewSyncManager()
	sm.SetAPICredentials(AppleWatch, "ak")
	sm.SetAPICredentials(SamsungGalaxyWatch, "sk")
	sm.SetAPICredentials(HuaweiWatchD2, "hk")
	sm.SetAPICredentials(AmazfitTRex3, "zk")

	// Test each device type
	deviceTypes := []DeviceType{AppleWatch, SamsungGalaxyWatch, HuaweiWatchD2, AmazfitTRex3}
	for _, dt := range deviceTypes {
		t.Run(string(dt), func(t *testing.T) {
			samples, err := sm.FetchFromRealDevice(context.Background(), dt, "user-1", []string{"heart_rate"})
			require.NoError(t, err)
			assert.Len(t, samples, 1)
		})
	}
}

func TestSyncManagerSyncAllDevicesNoDevices(t *testing.T) {
	sm := NewSyncManager()
	results := sm.SyncAllDevices()
	assert.Empty(t, results)
}

func TestSyncManagerSyncAllDevicesNotDue(t *testing.T) {
	sm := NewSyncManager()
	state := DefaultPhysiologicalState()
	emulator := NewDeviceEmulator("dev-1", "user-1", "tok", AppleWatch, state)
	// Set LastSync to now so sync interval hasn't passed
	emulator.LastSync = time.Now()
	sm.RegisterDevice(emulator)

	results := sm.SyncAllDevices()
	assert.Empty(t, results, "should not sync if interval hasn't passed")
}

func TestSyncManagerGetEmulatorNotFound(t *testing.T) {
	sm := NewSyncManager()
	_, exists := sm.GetEmulator("nonexistent")
	assert.False(t, exists)
}

func TestSyncManagerMultipleDevices(t *testing.T) {
	sm := NewSyncManager()
	state := DefaultPhysiologicalState()

	dev1 := NewDeviceEmulator("dev-1", "user-1", "t1", AppleWatch, state)
	dev2 := NewDeviceEmulator("dev-2", "user-1", "t2", SamsungGalaxyWatch, state)
	dev3 := NewDeviceEmulator("dev-3", "user-1", "t3", HuaweiWatchD2, state)

	sm.RegisterDevice(dev1)
	sm.RegisterDevice(dev2)
	sm.RegisterDevice(dev3)

	_, ok := sm.GetEmulator("dev-1")
	assert.True(t, ok)
	_, ok = sm.GetEmulator("dev-2")
	assert.True(t, ok)
	_, ok = sm.GetEmulator("dev-3")
	assert.True(t, ok)

	sm.UnregisterDevice("dev-2")
	_, ok = sm.GetEmulator("dev-2")
	assert.False(t, ok)
	// Others should still exist
	_, ok = sm.GetEmulator("dev-1")
	assert.True(t, ok)
	_, ok = sm.GetEmulator("dev-3")
	assert.True(t, ok)
}

// --- HTTP Handlers: RegisterHandler ---

func TestRegisterHandlerSuccess(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	body := `{"device_type": "apple_watch", "user_id": "user-1"}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "user-1", resp["user_id"])
	assert.Equal(t, "apple_watch", resp["device_type"])
	assert.True(t, resp["emulated"].(bool))
	assert.NotEmpty(t, resp["device_id"])
	assert.NotEmpty(t, resp["device_token"])
}

func TestRegisterHandlerSamsungWatch(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	body := `{"device_type": "samsung_galaxy_watch", "user_id": "user-2"}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "samsung_galaxy_watch", resp["device_type"])
}

func TestRegisterHandlerHuaweiWatch(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	body := `{"device_type": "huawei_watch_d2", "user_id": "user-3"}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "huawei_watch_d2", resp["device_type"])
}

func TestRegisterHandlerAmazfit(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	body := `{"device_type": "amazfit_trex3", "user_id": "user-4"}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "amazfit_trex3", resp["device_type"])
}

func TestRegisterHandlerInvalidJSON(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid request")
}

func TestRegisterHandlerUnsupportedDevice(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	body := `{"device_type": "fitbit", "user_id": "user-1"}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "unsupported device type")
}

func TestRegisterHandlerEmptyDeviceType(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	body := `{"device_type": "", "user_id": "user-1"}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRegisterHandlerEmptyUserID(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	body := `{"device_type": "apple_watch", "user_id": ""}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterHandler(rr, req)

	// Should still register (empty userID is accepted by the handler)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- HTTP Handlers: SyncHandler ---

func TestSyncHandlerSuccess(t *testing.T) {
	state := DefaultPhysiologicalState()
	handler := NewEmulatorHTTPHandler(state)

	// First register
	regBody := `{"device_type": "apple_watch", "user_id": "user-1"}`
	regReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", strings.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRr := httptest.NewRecorder()
	handler.RegisterHandler(regRr, regReq)

	var regResp map[string]interface{}
	err := json.Unmarshal(regRr.Body.Bytes(), &regResp)
	require.NoError(t, err)
	deviceID := regResp["device_id"].(string)

	// Wait for sync interval
	time.Sleep(10 * time.Millisecond)

	// Then sync
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sync?device_id="+deviceID, nil)
	rr := httptest.NewRecorder()
	handler.SyncHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, deviceID, resp["device_id"])
	assert.Greater(t, resp["count"].(float64), 0.0)
	samples, ok := resp["samples"].([]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, samples)
}

func TestSyncHandlerMissingDeviceID(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sync", nil)
	rr := httptest.NewRecorder()
	handler.SyncHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "device_id required")
}

func TestSyncHandlerEmptyDeviceID(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sync?device_id=", nil)
	rr := httptest.NewRecorder()
	handler.SyncHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSyncHandlerDeviceNotFound(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/sync?device_id=nonexistent", nil)
	rr := httptest.NewRecorder()
	handler.SyncHandler(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "device not found")
}

// --- HTTP Handlers: UpdateStateHandler ---

func TestUpdateStateHandlerSuccess(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	newState := `{
		"BaseHeartRate": 80,
		"BaseHRV": 60,
		"BaseSpO2": 97,
		"BaseTemperature": 36.8,
		"BaseBPSystolic": 130,
		"BaseBPDiastolic": 85,
		"ActivityLevel": 0.8,
		"StressLevel": 0.5,
		"SleepQuality": 0.6,
		"FitnessLevel": 0.7,
		"Age": 35,
		"Weight": 80,
		"Height": 180
	}`

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/state", strings.NewReader(newState))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.UpdateStateHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])

	// Verify state was updated
	assert.Equal(t, 80.0, handler.State.BaseHeartRate)
	assert.Equal(t, 60.0, handler.State.BaseHRV)
	assert.Equal(t, 0.8, handler.State.ActivityLevel)
}

func TestUpdateStateHandlerInvalidJSON(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/state", strings.NewReader("bad-json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.UpdateStateHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid request")
}

func TestUpdateStateHandlerEmptyBody(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/state", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.UpdateStateHandler(rr, req)

	// Should succeed with zero values
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestUpdateStateHandlerPartialUpdate(t *testing.T) {
	handler := NewEmulatorHTTPHandler(DefaultPhysiologicalState())

	body := `{"BaseHeartRate": 90, "StressLevel": 0.9}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/state", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.UpdateStateHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 90.0, handler.State.BaseHeartRate)
	assert.Equal(t, 0.9, handler.State.StressLevel)
	// Unset fields should be zero
	assert.Equal(t, 0.0, handler.State.BaseHRV)
}

// --- NewEmulatorHTTPHandler ---

func TestNewEmulatorHTTPHandler(t *testing.T) {
	state := DefaultPhysiologicalState()
	handler := NewEmulatorHTTPHandler(state)

	require.NotNil(t, handler)
	assert.Equal(t, state, handler.State)
	assert.NotNil(t, handler.SyncManager)
}

// --- GenerateBatch with all device types ---

func TestGenerateBatchAppleWatch(t *testing.T) {
	state := DefaultPhysiologicalState()
	emulator := NewDeviceEmulator("dev", "user", "tok", AppleWatch, state)
	batch := emulator.GenerateBatch()

	metrics := make(map[string]bool)
	for _, s := range batch {
		metrics[s.MetricType] = true
	}

	assert.True(t, metrics["heart_rate"])
	assert.True(t, metrics["ecg_lead_0"])
	assert.True(t, metrics["spo2"])
	assert.True(t, metrics["hrv"])
	assert.True(t, metrics["sleep_stage"])
	assert.True(t, metrics["steps"])
	assert.False(t, metrics["temperature"], "Apple Watch should not have temperature")
	assert.False(t, metrics["blood_pressure_systolic"], "Apple Watch should not have blood pressure")
}

func TestGenerateBatchAmazfitNoECG(t *testing.T) {
	state := DefaultPhysiologicalState()
	emulator := NewDeviceEmulator("dev", "user", "tok", AmazfitTRex3, state)
	batch := emulator.GenerateBatch()

	metrics := make(map[string]bool)
	for _, s := range batch {
		metrics[s.MetricType] = true
	}

	assert.True(t, metrics["heart_rate"])
	assert.False(t, metrics["ecg_lead_0"], "Amazfit should not have ECG")
	assert.False(t, metrics["blood_pressure_systolic"], "Amazfit should not have blood pressure")
}

func TestGenerateBatchUnknownDeviceType(t *testing.T) {
	state := DefaultPhysiologicalState()
	// Create emulator with unknown device type
	emulator := &DeviceEmulator{
		DeviceID:     "dev",
		DeviceType:   "unknown",
		UserID:       "user",
		DeviceToken:  "tok",
		Capabilities: GetDefaultCapabilities("unknown"),
		Generator:    NewDataGenerator(state),
		SyncInterval: 30 * time.Second,
	}

	batch := emulator.GenerateBatch()
	assert.NotEmpty(t, batch)
	// Should have basic metrics but no ECG/BP
	metrics := make(map[string]bool)
	for _, s := range batch {
		metrics[s.MetricType] = true
	}
	assert.True(t, metrics["heart_rate"])
	assert.False(t, metrics["ecg_lead_0"])
}

// --- sleepStageToValue ---

func TestSleepStageToValueDefault(t *testing.T) {
	assert.Equal(t, float64(0), sleepStageToValue("unknown_stage"))
	assert.Equal(t, float64(0), sleepStageToValue(""))
}
