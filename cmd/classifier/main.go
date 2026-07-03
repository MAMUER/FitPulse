package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

type server struct {
	log *zap.Logger
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
		Name:        "endurance_e1e2",
		NameRu:      "Базовая выносливость (E1-E2)",
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
		Name:        "threshold_e3",
		NameRu:      "Пороговая выносливость (E3)",
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
		Name:        "strength_hiit",
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
	PredictedClass    string             `json:"predicted_class"`
	PredictedClassRu  string             `json:"predicted_class_ru"`
	Confidence        float64            `json:"confidence"`
	Probabilities     map[string]float64 `json:"probabilities"`
	Description       string             `json:"description"`
	HrRange           string             `json:"hr_range"`
	Recommendations   []string           `json:"recommendations"`
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

func defaultIfZero(val, def float64) float64 {
	if val == 0 {
		return def
	}
	return val
}

func classifyState(data physiologicalData, age int) (int, float64, map[string]float64) {
	if data.Temperature > 37.5 {
		probs := map[string]float64{
			"recovery":       0.0,
			"endurance_e1e2": 0.0,
			"threshold_e3":   0.0,
			"strength_hiit":  0.02,
			"overtraining":   0.05,
			"illness":        0.93,
		}
		return 5, 0.93, probs
	}
	if data.Temperature > 37.3 {
		probs := map[string]float64{
			"recovery":       0.05,
			"endurance_e1e2": 0.05,
			"threshold_e3":   0.02,
			"strength_hiit":  0.03,
			"overtraining":   0.10,
			"illness":        0.75,
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
			"recovery":       0.03,
			"endurance_e1e2": 0.05,
			"threshold_e3":   0.02,
			"strength_hiit":  0.05,
			"overtraining":   0.85,
			"illness":        0.05,
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
	classNames := []string{"recovery", "endurance_e1e2", "threshold_e3", "strength_hiit", "overtraining", "illness"}
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

	notes := []string{}
	if profile.FitnessLevel == "beginner" {
		notes = append(notes, "Рекомендуется снизить интенсивность на 10-15%")
	}
	if profile.Age > 50 {
		notes = append(notes, "Учитывайте возраст при планировании восстановления")
	}
	if len(profile.HealthConditions) > 0 {
		notes = append(notes, fmt.Sprintf("Проконсультируйтесь с врачом при: %s", joinStrings(profile.HealthConditions, ", ")))
	}
	if len(profile.Goals) > 0 && predictedClass == 0 {
		for _, g := range profile.Goals {
			if g == "похудение" || g == "weight_loss" {
				notes = append(notes, "Для похудения добавьте кардио в зоне E1-E2")
				break
			}
		}
	}

	if len(notes) == 0 {
		return nil
	}
	result := joinStrings(notes, " | ")
	return &result
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += sep + ss[i]
	}
	return result
}

func (s *server) classifyHandler(w http.ResponseWriter, r *http.Request) {
	var req classifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Warn("Invalid classify request", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
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

	resp := classifyResponse{
		PredictedClass:    classInfo.Name,
		PredictedClassRu:  classInfo.NameRu,
		Confidence:        confidence,
		Probabilities:     probs,
		Description:       classInfo.Description,
		HrRange:           classInfo.HrRange,
		Recommendations:   classInfo.Recommendations,
		PersonalizedNotes: personalizedNotes,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	log, _ := zap.NewProduction()
	s := &server{log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/classes", s.classesHandler)
	mux.HandleFunc("/classify", s.classifyHandler)
	mux.HandleFunc("/metrics", s.metricsHandler)
	mux.HandleFunc("/model-info", s.modelInfoHandler)

	port := defaultEnv("CLASSIFIER_PORT", "8001")
	log.Info("Starting classifier service", zap.String("port", port))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      logMiddleware(log, mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed", zap.Error(err))
	}
}

func defaultEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func logMiddleware(log *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		safePath := strings.ReplaceAll(strings.ReplaceAll(r.URL.Path, "\n", ""), "\r", "")
		log.Info("request",
			zap.String("method", r.Method),
			zap.String("path", safePath),
			zap.Duration("duration", time.Since(start)),
		)
	})
}
