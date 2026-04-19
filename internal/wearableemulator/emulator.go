// Package wearableemulator эмулирует работу носимых устройств (Apple Watch, Samsung Galaxy Watch,
// Huawei Watch D2, Amazfit T-Rex 3) с реалистичной генерацией биометрических данных.
//
// Эмулятор имитирует реальные протоколы синхронизации:
// • Apple Watch — через HealthKit API (заглушка для реального подключения)
// • Samsung Galaxy Watch — через Samsung Health API (заглушка)
// • Huawei Watch D2 — через Huawei Health Kit (заглушка)
// • Amazfit T-Rex 3 — через Zepp API (заглушка)
//
// Когда реальное устройство недоступно, эмулятор генерирует данные,
// идентичные по формату и характеристикам реальным устройствам.
package wearableemulator

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

// ==========================================
// Device Type Definitions
// ==========================================

// DeviceType определяет тип носимого устройства
type DeviceType string

const (
	AppleWatch         DeviceType = "apple_watch"
	SamsungGalaxyWatch DeviceType = "samsung_galaxy_watch"
	HuaweiWatchD2      DeviceType = "huawei_watch_d2"
	AmazfitTRex3       DeviceType = "amazfit_trex3"
)

// DeviceCapabilities описывает возможности устройства (какие метрики оно поддерживает)
type DeviceCapabilities struct {
	HeartRate     bool `json:"heart_rate"`     // Пульс (BPM)
	ECG           bool `json:"ecg"`            // ЭКГ (только Apple Watch, Huawei Watch D2)
	BloodPressure bool `json:"blood_pressure"` // Давление (только Huawei Watch D2)
	SpO2          bool `json:"spo2"`           // Сатурация крови
	Temperature   bool `json:"temperature"`    // Температура тела
	Sleep         bool `json:"sleep"`          // Фазы сна
	Steps         bool `json:"steps"`          // Шаги
	HRV           bool `json:"hrv"`            // Вариабельность сердечного ритма
}

// GetDefaultCapabilities возвращает возможности для каждого типа устройства
func GetDefaultCapabilities(dt DeviceType) DeviceCapabilities {
	switch dt {
	case AppleWatch:
		return DeviceCapabilities{
			HeartRate: true, ECG: true, BloodPressure: false,
			SpO2: true, Temperature: false, Sleep: true,
			Steps: true, HRV: true,
		}
	case SamsungGalaxyWatch:
		return DeviceCapabilities{
			HeartRate: true, ECG: true, BloodPressure: false,
			SpO2: true, Temperature: true, Sleep: true,
			Steps: true, HRV: true,
		}
	case HuaweiWatchD2:
		return DeviceCapabilities{
			HeartRate: true, ECG: true, BloodPressure: true,
			SpO2: true, Temperature: true, Sleep: true,
			Steps: true, HRV: true,
		}
	case AmazfitTRex3:
		return DeviceCapabilities{
			HeartRate: true, ECG: false, BloodPressure: false,
			SpO2: true, Temperature: false, Sleep: true,
			Steps: true, HRV: true,
		}
	default:
		return DeviceCapabilities{
			HeartRate: true, ECG: false, BloodPressure: false,
			SpO2: true, Temperature: false, Sleep: true,
			Steps: true, HRV: false,
		}
	}
}

// ==========================================
// Biometric Data Models
// ==========================================

// BiometricSample представляет одно биометрическое измерение
type BiometricSample struct {
	DeviceType string    `json:"device_type"`
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	Quality    string    `json:"quality"` // good, fair, poor
}

// UserPhysiologicalState представляет физиологическое состояние пользователя
type UserPhysiologicalState struct {
	BaseHeartRate   float64 // Базовый пульс в покое
	BaseHRV         float64 // Базовая вариабельность
	BaseSpO2        float64 // Базовая сатурация
	BaseTemperature float64 // Базовая температура
	BaseBPSystolic  float64 // Базовое систолическое давление
	BaseBPDiastolic float64 // Базовое диастолическое давление
	ActivityLevel   float64 // Уровень активности (0-1)
	StressLevel     float64 // Уровень стресса (0-1)
	SleepQuality    float64 // Качество сна (0-1)
	FitnessLevel    float64 // Уровень фитнеса (0-1)
	Age             int     // Возраст
	Weight          float64 // Вес (кг)
	Height          int     // Рост (см)
}

// DefaultPhysiologicalState возвращает состояние пользователя по умолчанию
func DefaultPhysiologicalState() *UserPhysiologicalState {
	return &UserPhysiologicalState{
		BaseHeartRate:   72,
		BaseHRV:         50,
		BaseSpO2:        98,
		BaseTemperature: 36.6,
		BaseBPSystolic:  120,
		BaseBPDiastolic: 80,
		ActivityLevel:   0.5,
		StressLevel:     0.3,
		SleepQuality:    0.7,
		FitnessLevel:    0.6,
		Age:             30,
		Weight:          75,
		Height:          175,
	}
}

// ==========================================
// Realistic Data Generator
// ==========================================

// DataGenerator generates realistic biometric data using crypto/rand
type DataGenerator struct {
	state *UserPhysiologicalState
}

// cryptoFloat64 returns a cryptographically secure float64 in [0, 1)
func cryptoFloat64() float64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	n := binary.LittleEndian.Uint64(b[:])
	return float64(n) / float64(0x1_0000_0000_0000_0000)
}

// cryptoGauss returns Gaussian noise using Box-Muller transform with crypto/rand
func cryptoGauss(stdDev float64) float64 {
	u1 := cryptoFloat64()
	u2 := cryptoFloat64()
	if u1 == 0 {
		u1 = 1e-10
	}
	return stdDev * math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

// NewDataGenerator creates a data generator
func NewDataGenerator(state *UserPhysiologicalState) *DataGenerator {
	return &DataGenerator{
		state: state,
	}
}

// GenerateHeartRate generates realistic heart rate
func (g *DataGenerator) GenerateHeartRate() float64 {
	base := g.state.BaseHeartRate

	activityVariation := g.state.ActivityLevel * 40 * math.Sin(cryptoFloat64()*math.Pi*2)
	stressVariation := g.state.StressLevel * 15 * (cryptoFloat64() - 0.5)
	hour := time.Now().Hour()
	circadianVariation := -5 * math.Cos(float64(hour)/24*math.Pi*2)
	fitnessAdjustment := -g.state.FitnessLevel * 10

	value := base + activityVariation + stressVariation + circadianVariation + fitnessAdjustment
	value += cryptoGauss(2)

	return math.Max(40, math.Min(200, value))
}

// GenerateHRV генерирует вариабельность сердечного ритма
func (g *DataGenerator) GenerateHRV() float64 {
	base := g.state.BaseHRV
	stressAdjustment := -g.state.StressLevel * 20
	sleepAdjustment := g.state.SleepQuality * 10
	noise := cryptoGauss(3)

	value := base + stressAdjustment + sleepAdjustment + noise
	return math.Max(10, math.Min(100, value))
}

// GenerateSpO2 генерирует сатурацию крови
func (g *DataGenerator) GenerateSpO2() float64 {

	base := g.state.BaseSpO2
	noise := cryptoGauss(0.5)

	value := base + noise
	return math.Max(90, math.Min(100, value))
}

// GenerateTemperature генерирует температуру тела
func (g *DataGenerator) GenerateTemperature() float64 {

	base := g.state.BaseTemperature
	hour := time.Now().Hour()
	circadianVariation := 0.3 * math.Sin(float64(hour)/24*math.Pi*2)
	noise := cryptoGauss(0.1)

	value := base + circadianVariation + noise
	return math.Max(35.5, math.Min(38.5, value))
}

// GenerateBloodPressure генерирует артериальное давление
func (g *DataGenerator) GenerateBloodPressure() (systolic, diastolic float64) {

	baseSys := g.state.BaseBPSystolic
	baseDia := g.state.BaseBPDiastolic

	stressAdjSys := g.state.StressLevel * 15
	stressAdjDia := g.state.StressLevel * 10
	activityAdjSys := g.state.ActivityLevel * 20
	noiseSys := cryptoGauss(3)
	noiseDia := cryptoGauss(2)

	systolic = baseSys + stressAdjSys + activityAdjSys + noiseSys
	diastolic = baseDia + stressAdjDia + noiseDia

	systolic = math.Max(80, math.Min(200, systolic))
	diastolic = math.Max(50, math.Min(130, diastolic))

	// Ensure systolic > diastolic
	if systolic <= diastolic {
		systolic = diastolic + 10
		if systolic > 200 {
			systolic = 200
			diastolic = systolic - 10
		}
	}

	return systolic, diastolic
}

// GenerateSleepStage генерирует текущую фазу сна
func (g *DataGenerator) GenerateSleepStage() string {

	hour := time.Now().Hour()
	// Предполагаем, что сон между 23:00 и 7:00
	if hour >= 23 || hour < 7 {
		r := cryptoFloat64()
		quality := g.state.SleepQuality

		switch {
		case r < 0.1*quality:
			return "deep"
		case r < 0.5*quality:
			return "light"
		case r < 0.7*quality:
			return "rem"
		default:
			return sleepStageAwake
		}
	}
	return sleepStageAwake
}

// GenerateSteps генерирует количество шагов за интервал
func (g *DataGenerator) GenerateSteps() float64 {

	hour := time.Now().Hour()

	// Активность выше днём
	var base float64
	switch {
	case hour >= 8 && hour < 12:
		base = 500
	case hour >= 12 && hour < 18:
		base = 700
	case hour >= 18 && hour < 22:
		base = 300
	default:
		base = 50
	}

	activityMultiplier := 0.5 + g.state.ActivityLevel*1.5
	value := base * activityMultiplier * (0.5 + cryptoFloat64())

	return math.Max(0, value)
}

// GenerateECG генерирует упрощённые данные ЭКГ (симуляция PQRST волны)
func (g *DataGenerator) GenerateECG() []float64 {
	// GenerateHeartRate acquires its own lock, so call it BEFORE locking
	heartRate := g.GenerateHeartRate()

	// Симуляция PQRST комплекса (5 точек)
	amplitude := 1.0 + (heartRate-70)/100

	ecgPoints := []float64{
		0.1 * amplitude,  // P wave
		0.05 * amplitude, // PR segment
		-0.2 * amplitude, // Q wave
		1.2 * amplitude,  // R wave (пик)
		-0.3 * amplitude, // S wave
	}

	return ecgPoints
}

// ==========================================
// Device Emulator
// ==========================================

// DeviceEmulator эмулирует конкретное носимое устройство
type DeviceEmulator struct {
	DeviceID     string
	DeviceType   DeviceType
	UserID       string
	DeviceToken  string
	Capabilities DeviceCapabilities
	Generator    *DataGenerator
	SyncInterval time.Duration
	LastSync     time.Time
}

// NewDeviceEmulator создаёт эмулятор устройства
func NewDeviceEmulator(deviceID, userID, deviceToken string, deviceType DeviceType, state *UserPhysiologicalState) *DeviceEmulator {
	return &DeviceEmulator{
		DeviceID:     deviceID,
		DeviceType:   deviceType,
		UserID:       userID,
		DeviceToken:  deviceToken,
		Capabilities: GetDefaultCapabilities(deviceType),
		Generator:    NewDataGenerator(state),
		SyncInterval: 30 * time.Second,
		LastSync:     time.Time{},
	}
}

// GenerateBatch генерирует пакет биометрических данных для синхронизации
func (e *DeviceEmulator) GenerateBatch() []BiometricSample {
	now := time.Now()
	samples := make([]BiometricSample, 0)

	if e.Capabilities.HeartRate {
		samples = append(samples, BiometricSample{
			DeviceType: string(e.DeviceType),
			MetricType: "heart_rate",
			Value:      e.Generator.GenerateHeartRate(),
			Timestamp:  now,
			Quality:    "good",
		})
	}

	if e.Capabilities.HRV {
		samples = append(samples, BiometricSample{
			DeviceType: string(e.DeviceType),
			MetricType: "hrv",
			Value:      e.Generator.GenerateHRV(),
			Timestamp:  now,
			Quality:    "good",
		})
	}

	if e.Capabilities.SpO2 {
		samples = append(samples, BiometricSample{
			DeviceType: string(e.DeviceType),
			MetricType: "spo2",
			Value:      e.Generator.GenerateSpO2(),
			Timestamp:  now,
			Quality:    "good",
		})
	}

	if e.Capabilities.Temperature {
		samples = append(samples, BiometricSample{
			DeviceType: string(e.DeviceType),
			MetricType: "temperature",
			Value:      e.Generator.GenerateTemperature(),
			Timestamp:  now,
			Quality:    "good",
		})
	}

	if e.Capabilities.BloodPressure {
		sys, dia := e.Generator.GenerateBloodPressure()
		samples = append(samples, BiometricSample{
			DeviceType: string(e.DeviceType),
			MetricType: "blood_pressure_systolic",
			Value:      sys,
			Timestamp:  now,
			Quality:    "good",
		})
		samples = append(samples, BiometricSample{
			DeviceType: string(e.DeviceType),
			MetricType: "blood_pressure_diastolic",
			Value:      dia,
			Timestamp:  now,
			Quality:    "good",
		})
	}

	if e.Capabilities.ECG {
		ecgValues := e.Generator.GenerateECG()
		for i, val := range ecgValues {
			samples = append(samples, BiometricSample{
				DeviceType: string(e.DeviceType),
				MetricType: fmt.Sprintf("ecg_lead_%d", i),
				Value:      val,
				Timestamp:  now.Add(time.Duration(i) * time.Millisecond),
				Quality:    "good",
			})
		}
	}

	if e.Capabilities.Sleep {
		samples = append(samples, BiometricSample{
			DeviceType: string(e.DeviceType),
			MetricType: "sleep_stage",
			Value:      sleepStageToValue(e.Generator.GenerateSleepStage()),
			Timestamp:  now,
			Quality:    "good",
		})
	}

	if e.Capabilities.Steps {
		samples = append(samples, BiometricSample{
			DeviceType: string(e.DeviceType),
			MetricType: "steps",
			Value:      e.Generator.GenerateSteps(),
			Timestamp:  now,
			Quality:    "good",
		})
	}

	e.LastSync = now
	return samples
}

const sleepStageAwake = "awake"

// sleepStageToValue преобразует фазу сна в числовое значение
func sleepStageToValue(stage string) float64 {
	switch stage {
	case "deep":
		return 4
	case "light":
		return 3
	case "rem":
		return 2
	case sleepStageAwake:
		return 1
	default:
		return 0
	}
}

// ==========================================
// HealthKit API Stub (Apple Watch)
// ==========================================

// HealthKitClient заглушка для реального HealthKit API
// В будущем здесь будет реальное подключение через Apple HealthKit API
type HealthKitClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewHealthKitClient создаёт клиент HealthKit API
func NewHealthKitClient(apiKey string) *HealthKitClient {
	return &HealthKitClient{
		BaseURL:    "https://api.apple-healthkit.example.com/v1",
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// FetchBiometricData заглушка для получения данных из HealthKit
// В реальной реализации здесь будет HTTP запрос к Apple HealthKit API
func (c *HealthKitClient) FetchBiometricData(ctx context.Context, userID string, metricTypes []string) ([]BiometricSample, error) {
	// TODO: Реальная реализация при наличии доступа к Apple HealthKit API
	// Сейчас возвращаем эмулированные данные

	// Пример реального запроса (когда API будет доступен):
	/*
		req, _ := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/samples")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		q := req.URL.Query()
		q.Add("user_id", userID)
		q.Add("metrics", strings.Join(metricTypes, ","))
		req.URL.RawQuery = q.Encode()

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var data struct {
			Samples []BiometricSample `json:"samples"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}
		return data.Samples, nil
	*/

	// Эмуляция: возвращаем фиктивные данные
	samples := make([]BiometricSample, len(metricTypes))
	for i, metric := range metricTypes {
		samples[i] = BiometricSample{
			DeviceType: string(AppleWatch),
			MetricType: metric,
			Value:      0, // Заглушка
			Timestamp:  time.Now(),
			Quality:    "emulated",
		}
	}

	return samples, nil
}

// ==========================================
// Samsung Health API Stub
// ==========================================

// SamsungHealthClient заглушка для Samsung Health API
type SamsungHealthClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewSamsungHealthClient создаёт клиент Samsung Health API
func NewSamsungHealthClient(apiKey string) *SamsungHealthClient {
	return &SamsungHealthClient{
		BaseURL:    "https://api.samsung-health.example.com/v1",
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// FetchBiometricData заглушка для получения данных из Samsung Health
func (c *SamsungHealthClient) FetchBiometricData(ctx context.Context, userID string, metricTypes []string) ([]BiometricSample, error) {
	// TODO: Реальная реализация при наличии доступа к Samsung Health API
	samples := make([]BiometricSample, len(metricTypes))
	for i, metric := range metricTypes {
		samples[i] = BiometricSample{
			DeviceType: string(SamsungGalaxyWatch),
			MetricType: metric,
			Value:      0,
			Timestamp:  time.Now(),
			Quality:    "emulated",
		}
	}
	return samples, nil
}

// ==========================================
// Huawei Health Kit API Stub
// ==========================================

// HuaweiHealthClient заглушка для Huawei Health Kit API
type HuaweiHealthClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewHuaweiHealthClient создаёт клиент Huawei Health Kit API
func NewHuaweiHealthClient(apiKey string) *HuaweiHealthClient {
	return &HuaweiHealthClient{
		BaseURL:    "https://api.huawei-health.example.com/v1",
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// FetchBiometricData заглушка для получения данных из Huawei Health Kit
func (c *HuaweiHealthClient) FetchBiometricData(ctx context.Context, userID string, metricTypes []string) ([]BiometricSample, error) {
	// TODO: Реальная реализация при наличии доступа к Huawei Health Kit API
	samples := make([]BiometricSample, len(metricTypes))
	for i, metric := range metricTypes {
		samples[i] = BiometricSample{
			DeviceType: string(HuaweiWatchD2),
			MetricType: metric,
			Value:      0,
			Timestamp:  time.Now(),
			Quality:    "emulated",
		}
	}
	return samples, nil
}

// ==========================================
// Zepp API Stub (Amazfit)
// ==========================================

// ZeppClient заглушка для Zepp API (Amazfit)
type ZeppClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewZeppClient создаёт клиент Zepp API
func NewZeppClient(apiKey string) *ZeppClient {
	return &ZeppClient{
		BaseURL:    "https://api.zepp-life.example.com/v1",
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// FetchBiometricData заглушка для получения данных из Zepp API
func (c *ZeppClient) FetchBiometricData(ctx context.Context, userID string, metricTypes []string) ([]BiometricSample, error) {
	// TODO: Реальная реализация при наличии доступа к Zepp API
	samples := make([]BiometricSample, len(metricTypes))
	for i, metric := range metricTypes {
		samples[i] = BiometricSample{
			DeviceType: string(AmazfitTRex3),
			MetricType: metric,
			Value:      0,
			Timestamp:  time.Now(),
			Quality:    "emulated",
		}
	}
	return samples, nil
}

// ==========================================
// Sync Manager — управление синхронизацией
// ==========================================

// SyncManager управляет синхронизацией данных с устройств
type SyncManager struct {
	emulators map[string]*DeviceEmulator // deviceID -> emulator
	mu        sync.RWMutex

	// API клиенты для реальных подключений (заглушки)
	healthKitClient     *HealthKitClient
	samsungHealthClient *SamsungHealthClient
	huaweiHealthClient  *HuaweiHealthClient
	zeppClient          *ZeppClient
}

// NewSyncManager создаёт менеджер синхронизации
func NewSyncManager() *SyncManager {
	return &SyncManager{
		emulators: make(map[string]*DeviceEmulator),
	}
}

// RegisterDevice регистрирует устройство для эмуляции
func (sm *SyncManager) RegisterDevice(emulator *DeviceEmulator) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.emulators[emulator.DeviceID] = emulator
}

// UnregisterDevice удаляет устройство из эмуляции
func (sm *SyncManager) UnregisterDevice(deviceID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.emulators, deviceID)
}

// GetEmulator возвращает эмулятор устройства
func (sm *SyncManager) GetEmulator(deviceID string) (*DeviceEmulator, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	emu, exists := sm.emulators[deviceID]
	return emu, exists
}

// SyncAllDevices синхронизирует данные со всех устройств
func (sm *SyncManager) SyncAllDevices() map[string][]BiometricSample {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	results := make(map[string][]BiometricSample)
	for deviceID, emulator := range sm.emulators {
		if time.Since(emulator.LastSync) >= emulator.SyncInterval {
			results[deviceID] = emulator.GenerateBatch()
		}
	}
	return results
}

// SetAPICredentials устанавливает credentials для реальных API
func (sm *SyncManager) SetAPICredentials(deviceType DeviceType, apiKey string) {
	switch deviceType {
	case AppleWatch:
		sm.healthKitClient = NewHealthKitClient(apiKey)
	case SamsungGalaxyWatch:
		sm.samsungHealthClient = NewSamsungHealthClient(apiKey)
	case HuaweiWatchD2:
		sm.huaweiHealthClient = NewHuaweiHealthClient(apiKey)
	case AmazfitTRex3:
		sm.zeppClient = NewZeppClient(apiKey)
	}
}

// FetchFromRealDevice пытается получить данные с реального устройства через API
func (sm *SyncManager) FetchFromRealDevice(ctx context.Context, deviceType DeviceType, userID string, metricTypes []string) ([]BiometricSample, error) {
	var client interface {
		FetchBiometricData(ctx context.Context, userID string, metricTypes []string) ([]BiometricSample, error)
	}

	switch deviceType {
	case AppleWatch:
		if sm.healthKitClient == nil {
			return nil, fmt.Errorf("HealthKit API not configured")
		}
		client = sm.healthKitClient
	case SamsungGalaxyWatch:
		if sm.samsungHealthClient == nil {
			return nil, fmt.Errorf("samsung health API not configured")
		}
		client = sm.samsungHealthClient
	case HuaweiWatchD2:
		if sm.huaweiHealthClient == nil {
			return nil, fmt.Errorf("huawei health kit API not configured")
		}
		client = sm.huaweiHealthClient
	case AmazfitTRex3:
		if sm.zeppClient == nil {
			return nil, fmt.Errorf("zepp API not configured")
		}
		client = sm.zeppClient
	default:
		return nil, fmt.Errorf("unsupported device type: %s", deviceType)
	}

	return client.FetchBiometricData(ctx, userID, metricTypes)
}

// ==========================================
// HTTP Handlers for Device Emulator
// ==========================================

// EmulatorHTTPHandler HTTP обработчик для эмулятора устройств
type EmulatorHTTPHandler struct {
	SyncManager *SyncManager
	State       *UserPhysiologicalState
	mu          sync.RWMutex
}

// NewEmulatorHTTPHandler создаёт HTTP обработчик
func NewEmulatorHTTPHandler(state *UserPhysiologicalState) *EmulatorHTTPHandler {
	return &EmulatorHTTPHandler{
		SyncManager: NewSyncManager(),
		State:       state,
	}
}

// RegisterHandler регистрирует устройство
func (h *EmulatorHTTPHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceType string `json:"device_type"`
		UserID     string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	deviceType := DeviceType(req.DeviceType)
	if _, ok := map[DeviceType]bool{
		AppleWatch: true, SamsungGalaxyWatch: true,
		HuaweiWatchD2: true, AmazfitTRex3: true,
	}[deviceType]; !ok {
		http.Error(w, "unsupported device type", http.StatusBadRequest)
		return
	}

	deviceID := fmt.Sprintf("emu-%s-%d", deviceType, time.Now().UnixNano())
	token := fmt.Sprintf("token-%d", time.Now().UnixNano())

	emulator := NewDeviceEmulator(deviceID, req.UserID, token, deviceType, h.State)
	h.SyncManager.RegisterDevice(emulator)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"device_id":    deviceID,
		"device_type":  deviceType,
		"device_token": token,
		"user_id":      req.UserID,
		"emulated":     true,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// SyncHandler синхронизирует данные с устройства
func (h *EmulatorHTTPHandler) SyncHandler(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, "device_id required", http.StatusBadRequest)
		return
	}

	emulator, exists := h.SyncManager.GetEmulator(deviceID)
	if !exists {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	samples := emulator.GenerateBatch()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"device_id": deviceID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"samples":   samples,
		"count":     len(samples),
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// UpdateStateHandler обновляет физиологическое состояние
func (h *EmulatorHTTPHandler) UpdateStateHandler(w http.ResponseWriter, r *http.Request) {
	var newState UserPhysiologicalState
	if err := json.NewDecoder(r.Body).Decode(&newState); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	h.State = &newState
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
