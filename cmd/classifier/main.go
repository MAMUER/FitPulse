package main

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/metrics"
	"github.com/MAMUER/project/internal/middleware"
)

type server struct {
	log *logger.Logger
}

const trainingClassCount = 6

var trainingClasses = map[int]struct {
	Name            string   `json:"name"`
	NameRu          string   `json:"name_ru"`
	Description     string   `json:"description"`
	HrRange         string   `json:"hr_range"`
	Hrv             string   `json:"hrv"`
	Spo2            string   `json:"spo2"`
	Recommendations []string `json:"recommendations"`
}{
	0: {
		Name:        "recovery",
		NameRu:      "Восстановление",
		Description: "Низкая нагрузка + высокий HRV + хорошее восстановление",
		HrRange:     "50-65% HRmax",
		Hrv:         "Высокий",
		Spo2:        "96-99%",
		Recommendations: []string{
			"Лёгкая активность (ходьба, йога)",
			"Растяжка и мобилизация",
			"Плавание в лёгком темпе",
			"Велопрогулка без напряжения",
		},
	},
	1: {
		Name:        "endurance_basic",
		NameRu:      "Базовая выносливость E1-E2",
		Description: "Работа ниже лактатного порога, устойчивая кардиореспираторная система",
		HrRange:     "65-80% HRmax",
		Hrv:         "Умеренный",
		Spo2:        "95-98%",
		Recommendations: []string{
			"Бег в аэробной зоне",
			"Велосипед (средняя интенсивность)",
			"Плавание (дистанция)",
			"Лыжи/беговые лыжи",
		},
	},
	2: {
		Name:        "endurance_threshold",
		NameRu:      "Пороговая выносливость E3",
		Description: "Нагрузка вблизи анаэробного порога, баланс лактата",
		HrRange:     "80-90% HRmax",
		Hrv:         "Сниженный",
		Spo2:        "93-96%",
		Recommendations: []string{
			"Темповый бег",
			"Интервалы на пороге",
			"Fartlek тренировки",
			"Критическая мощность (велосипед)",
		},
	},
	3: {
		Name:        "power_hiit",
		NameRu:      "Силовая/HIIT",
		Description: "Высокая вариабельность пульса + постнагрузочная гипертензия + стресс-реакция",
		HrRange:     "90-100% HRmax",
		Hrv:         "Резкое падение",
		Spo2:        "90-94%",
		Recommendations: []string{
			"HIIT интервалы",
			"Силовые тренировки",
			"Спринты",
			"CrossFit/WOD",
		},
	},
	4: {
		Name:        "overtraining",
		NameRu:      "Перетренированность",
		Description: "Повышенный пульс в покое + низкий HRV + усталость + ухудшение сна",
		HrRange:     "Повышение в покое на 5-10% от нормы",
		Hrv:         "Значительно снижен",
		Spo2:        "95-98%",
		Recommendations: []string{
			"Полный отдых 1-3 дня",
			"Снижение объёма тренировок на 50%",
			"Консультация с тренером/спорт-медиком",
			"Акцент на сон и питание",
		},
	},
	5: {
		Name:        "illness",
		NameRu:      "Заболевание",
		Description: "Повышение температуры + общая слабость + отклонения в детоксикации",
		HrRange:     "Повышение на 10-20% от нормы",
		Hrv:         "Сниженный",
		Spo2:        "<95% при лихорадке",
		Recommendations: []string{
			"Прекратить все тренировки",
			"Обратиться к врачу при температуре >37.5°C",
			"Обильное питьё и постельный режим",
			"Возобновить тренировки только после выздоровления",
		},
	},
}

type physiologicalData struct {
	HeartRate              float64 `json:"heart_rate"`
	HeartRateVariability   float64 `json:"heart_rate_variability"`
	SpO2                   float64 `json:"spo2"`
	Temperature            float64 `json:"temperature"`
	BloodPressureSystolic  float64 `json:"blood_pressure_systolic"`
	BloodPressureDiastolic float64 `json:"blood_pressure_diastolic"`
	SleepHours             float64 `json:"sleep_hours"`
}

type userProfile struct {
	Gender           string   `json:"gender"`
	Age              int      `json:"age"`
	FitnessLevel     string   `json:"fitness_level"`
	Weight           *float64 `json:"weight,omitempty"`
	Height           *float64 `json:"height,omitempty"`
	HealthConditions []string `json:"health_conditions,omitempty"`
	Goals            []string `json:"goals,omitempty"`
}

type classifyRequest struct {
	PhysiologicalData physiologicalData `json:"physiological_data"`
	UserProfile       *userProfile      `json:"user_profile,omitempty"`
}

type classifyResponse struct {
	Status            string             `json:"status"`
	State             string             `json:"state"`
	Confidence        float64            `json:"confidence"`
	Recommendation    []string           `json:"recommendation"`
	FatigueLevel      *float64           `json:"fatigue_level,omitempty"`
	MotivationScore   *float64           `json:"motivation_score,omitempty"`
	RecoveryQuality   *float64           `json:"recovery_quality,omitempty"`
	PredictedClass    string             `json:"predicted_class,omitempty"`
	PredictedClassRu  string             `json:"predicted_class_ru,omitempty"`
	Probabilities     map[string]float64 `json:"probabilities,omitempty"`
	Description       string             `json:"description,omitempty"`
	HrRange           string             `json:"hr_range,omitempty"`
	PersonalizedNotes *string            `json:"personalized_notes,omitempty"`
}

type healthResponse struct {
	Status       string `json:"status"`
	ModelLoaded  bool   `json:"model_loaded"`
	ScalerLoaded bool   `json:"scaler_loaded"`
	AsyncEnabled bool   `json:"async_enabled"`
}

func (s *server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:       "healthy",
		ModelLoaded:  true,
		ScalerLoaded: true,
		AsyncEnabled: false,
	})
}

func (s *server) classesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(trainingClasses)
}

func (s *server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte("# Classifier metrics\n"))
}

func (s *server) modelInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"model_name":       "rule-based-classifier",
		"input_shape":      []int{1, 7},
		"output_shape":     []int{1, trainingClassCount},
		"total_params":     0,
		"training_classes": trainingClasses,
		"loaded_at":        time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *server) classifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req classifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Warn("Invalid classify request", zap.Error(err))
		metrics.ErrorTotal.WithLabelValues("classifier", "invalid_json").Inc()
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	if err := validateClassifyRequest(req); err != nil {
		s.log.Warn("Invalid classify payload", zap.Error(err))
		metrics.ErrorTotal.WithLabelValues("classifier", "validation_error").Inc()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data := req.PhysiologicalData
	data.HeartRateVariability = defaultIfZero(data.HeartRateVariability, 50.0)
	data.SpO2 = defaultIfZero(data.SpO2, 98.0)
	data.Temperature = defaultIfZero(data.Temperature, 37.0)
	data.BloodPressureSystolic = defaultIfZero(data.BloodPressureSystolic, 120.0)
	data.BloodPressureDiastolic = defaultIfZero(data.BloodPressureDiastolic, 80.0)
	data.SleepHours = defaultIfZero(data.SleepHours, 7.0)

	age := 30
	if req.UserProfile != nil && req.UserProfile.Age > 0 {
		age = req.UserProfile.Age
	}

	predictedClass, confidence, probs := classifyState(data, age)
	classInfo := trainingClasses[predictedClass]
	personalizedNotes := generatePersonalizedNotes(data, req.UserProfile, predictedClass)

	fatigueLevel, motivationScore, recoveryQuality := deriveScores(predictedClass)

	resp := classifyResponse{
		Status:            "success",
		State:             classInfo.Name,
		Confidence:        confidence,
		Recommendation:    classInfo.Recommendations,
		FatigueLevel:      &fatigueLevel,
		MotivationScore:   &motivationScore,
		RecoveryQuality:   &recoveryQuality,
		PredictedClass:    classInfo.Name,
		PredictedClassRu:  classInfo.NameRu,
		Probabilities:     probs,
		Description:       classInfo.Description,
		HrRange:           classInfo.HrRange,
		PersonalizedNotes: personalizedNotes,
	}

	metrics.ClassificationConfidence.WithLabelValues("rule-based", classInfo.Name).Set(confidence)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func validateClassifyRequest(req classifyRequest) error {
	data := req.PhysiologicalData
	if err := validateNonZeroRange("heart_rate", data.HeartRate, 20, 250); err != nil {
		return err
	}
	if err := validateNonZeroRange("heart_rate_variability", data.HeartRateVariability, 0, 300); err != nil {
		return err
	}
	if err := validateNonZeroRange("spo2", data.SpO2, 70, 100); err != nil {
		return err
	}
	if err := validateNonZeroRange("temperature", data.Temperature, 30, 45); err != nil {
		return err
	}
	if err := validateNonZeroRange("blood_pressure_systolic", data.BloodPressureSystolic, 60, 250); err != nil {
		return err
	}
	if err := validateNonZeroRange("blood_pressure_diastolic", data.BloodPressureDiastolic, 40, 150); err != nil {
		return err
	}
	if err := validateNonZeroRange("sleep_hours", data.SleepHours, 0, 24); err != nil {
		return err
	}
	return nil
}

func validateNonZeroRange(name string, val, min, max float64) error {
	if val == 0 {
		return nil
	}
	if val < min || val > max {
		return errors.New(name + " out of valid range (" + strconv.FormatFloat(min, 'f', -1, 64) + "-" + strconv.FormatFloat(max, 'f', -1, 64) + ")")
	}
	return nil
}

func deriveScores(predictedClass int) (fatigueLevel, motivationScore, recoveryQuality float64) {
	switch predictedClass {
	case 0:
		return 0.1, 0.8, 0.9
	case 1:
		return 0.3, 0.7, 0.7
	case 2:
		return 0.5, 0.6, 0.5
	case 3:
		return 0.7, 0.5, 0.3
	case 4:
		return 0.9, 0.2, 0.1
	case 5:
		return 1.0, 0.0, 0.0
	default:
		return 0.5, 0.5, 0.5
	}
}

func defaultIfZero(val, def float64) float64 {
	if val == 0 {
		return def
	}
	return val
}

func classifyState(data physiologicalData, age int) (int, float64, map[string]float64) {
	if data.Temperature > 37.5 {
		probs := map[string]float64{
			"recovery":            0.0,
			"endurance_basic":     0.0,
			"endurance_threshold": 0.0,
			"power_hiit":          0.02,
			"overtraining":        0.05,
			"illness":             0.93,
		}
		return 5, 0.93, probs
	}
	if data.Temperature > 37.3 {
		probs := map[string]float64{
			"recovery":            0.05,
			"endurance_basic":     0.05,
			"endurance_threshold": 0.02,
			"power_hiit":          0.03,
			"overtraining":        0.10,
			"illness":             0.75,
		}
		return 5, 0.75, probs
	}

	hrMax := 220.0 - float64(age)
	if hrMax <= 0 {
		hrMax = 200.0
	}
	hrPct := data.HeartRate / hrMax

	zone := 0
	switch {
	case hrPct < 0.65:
		zone = 0
	case hrPct < 0.80:
		zone = 1
	case hrPct < 0.90:
		zone = 2
	default:
		zone = 3
	}

	if data.HeartRateVariability < 30 && hrPct < 0.6 {
		topProbs := map[string]float64{
			"recovery":            0.03,
			"endurance_basic":     0.05,
			"endurance_threshold": 0.02,
			"power_hiit":          0.05,
			"overtraining":        0.85,
			"illness":             0.05,
		}
		return 4, 0.85, topProbs
	}

	boundaries := []float64{0.0, 0.65, 0.80, 0.90, 1.0}
	center := (boundaries[zone] + boundaries[zone+1]) / 2.0
	halfWidth := (boundaries[zone+1] - boundaries[zone]) / 2.0
	dist := math.Abs(hrPct - center)
	rawConf := 1.0 - (dist / halfWidth)
	if rawConf < 0.35 {
		rawConf = 0.35
	}
	confidence := math.Round(rawConf*10000) / 10000.0

	probs := make(map[string]float64)
	classNames := []string{"recovery", "endurance_basic", "endurance_threshold", "power_hiit", "overtraining", "illness"}
	remainder := (1.0 - confidence) / 5.0
	for i, name := range classNames {
		if i == zone {
			probs[name] = confidence
		} else {
			probs[name] = math.Round(remainder*10000) / 10000.0
		}
	}

	return zone, confidence, probs
}

func generatePersonalizedNotes(_ physiologicalData, profile *userProfile, predictedClass int) *string {
	if profile == nil {
		return nil
	}

	var notes []string

	if profile.Age > 50 {
		notes = append(notes, "Рекомендуется снизить интенсивность и увеличить время разминки")
	}

	if profile.FitnessLevel == "beginner" {
		notes = append(notes, "Начните с базовых упражнений и постепенно увеличивайте нагрузку")
	}

	if len(profile.HealthConditions) > 0 {
		notes = append(notes, "Проконсультируйтесь с врачом при: "+joinStrings(profile.HealthConditions, ", "))
	}

	for _, goal := range profile.Goals {
		switch goal {
		case "похудение":
			notes = append(notes, "Учитывая цель похудения, делайте акцент на аэробные тренировки в аэробной зоне")
		case "силовые":
			if predictedClass != 3 {
				notes = append(notes, "При силовых целях рекомендуется включать тренировки в силовой зоне")
			}
		}
	}

	if len(notes) == 0 {
		return nil
	}

	result := joinStrings(notes, ". ") + "."
	return &result
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(ss[0])
	for i := 1; i < len(ss); i++ {
		builder.WriteString(sep)
		builder.WriteString(ss[i])
	}
	return builder.String()
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func classifierLoggingMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			duration := time.Since(start)
			statusStr := strconv.Itoa(rw.statusCode)

			metrics.RequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration.Seconds())
			metrics.RequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()

			if rw.statusCode >= 400 {
				metrics.ErrorTotal.WithLabelValues("classifier", statusStr).Inc()
			}

			log.Info("HTTP_REQUEST",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rw.statusCode),
				zap.Int64("duration_ms", duration.Milliseconds()),
			)
		})
	}
}

func main() {
	log := logger.New("classifier")
	s := &server{log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/classes", s.classesHandler)
	mux.HandleFunc("/classify", s.classifyHandler)
	mux.HandleFunc("/metrics", s.metricsHandler)
	mux.HandleFunc("/model-info", s.modelInfoHandler)

	handler := middleware.RecoveryMiddleware(log.Logger)(mux)
	handler = middleware.RequestID(handler)
	handler = corsMiddleware(handler)
	handler = classifierLoggingMiddleware(log.Logger)(handler)

	port := config.GetEnv("CLASSIFIER_PORT", "8001")
	metricsPort := config.GetEnv("CLASSIFIER_METRICS_PORT", "9091")

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: metricsMux,
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("Starting metrics server", zap.String("port", metricsPort))
		if err := metricsSrv.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "Server closed") {
			log.Fatal("Metrics server failed", zap.Error(err))
		}
	}()

	go func() {
		log.Info("Starting classifier service", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "Server closed") {
			log.Fatal("Server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("Shutting down classifier service")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("HTTP server shutdown error", zap.Error(err))
		}
	}()
	go func() {
		defer wg.Done()
		if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
			log.Error("Metrics server shutdown error", zap.Error(err))
		}
	}()
	wg.Wait()
	log.Info("Classifier service stopped")
}
