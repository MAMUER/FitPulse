package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/metrics"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/sanitize"
	"github.com/MAMUER/project/internal/server"
)

type ganService struct {
	log          *logger.Logger
	pythonPath   string
	ganScript    string
	ganModelPath string
}

type generatePlanRequest struct {
	ClassIndex int     `json:"class_index"`
	Age        float64 `json:"age"`
	Gender     float64 `json:"gender"`
	Weight     float64 `json:"weight"`
	Height     float64 `json:"height"`
	Goal       float64 `json:"goal"`
	NumSamples int     `json:"num_samples,omitempty"`
}

type generatePlanResponse struct {
	Status     string          `json:"status"`
	ClassIndex int             `json:"class_index"`
	Plans      []generatedPlan `json:"plans"`
	Model      string          `json:"model"`
}

type generatedPlan struct {
	Sample int       `json:"sample"`
	Plan   []float64 `json:"plan"`
}

func (s *ganService) generatePlanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, errMethodNotSupported, http.StatusMethodNotAllowed)
		return
	}

	var req generatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Warn("Некорректный запрос генерации плана", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	if req.ClassIndex < 0 || req.ClassIndex > 6 {
		http.Error(w, "class_index должен быть в диапазоне 0-6", http.StatusBadRequest)
		return
	}

	if req.Age < 0 || req.Weight < 0 || req.Height < 0 {
		http.Error(w, "возраст, вес и рост не могут быть отрицательными", http.StatusBadRequest)
		return
	}

	if req.NumSamples <= 0 {
		req.NumSamples = 1
	}

	// Вызываем локальный Python GAN скрипт
	plans, err := s.callPythonGAN(req)
	if err != nil {
		s.log.Error("Ошибка вызова локального GAN", zap.Error(err), zap.String("user_id", sanitize.LogString(r.URL.Query().Get("user_id"))))
		http.Error(w, "Ошибка генерации плана", http.StatusInternalServerError)
		return
	}

	resp := generatePlanResponse{
		Status:     "ok",
		ClassIndex: req.ClassIndex,
		Plans:      plans,
		Model:      "conditional-gan-local",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.log.Error("Failed to encode generate plan response", zap.Error(err))
	}

	metrics.ClassificationConfidence.WithLabelValues("gan", "generated").Inc()
}

func (s *ganService) callPythonGAN(req generatePlanRequest) ([]generatedPlan, error) {
	// #nosec G204 -- exec arguments are validated and sourced from trusted config
	cmd := exec.Command(
		s.pythonPath,
		s.ganScript,
		"generate",
		"--model", s.ganModelPath,
		"--class-index", strconv.Itoa(req.ClassIndex),
		"--age", strconv.FormatFloat(req.Age, 'f', -1, 64),
		"--gender", strconv.FormatFloat(req.Gender, 'f', -1, 64),
		"--weight", strconv.FormatFloat(req.Weight, 'f', -1, 64),
		"--height", strconv.FormatFloat(req.Height, 'f', -1, 64),
		"--goal", strconv.FormatFloat(req.Goal, 'f', -1, 64),
		"--num-samples", strconv.Itoa(req.NumSamples),
		"--output", "-",
	)

	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("python GAN failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("python GAN error: %w", err)
	}

	var result []generatedPlan
	if err := json.Unmarshal(stdout, &result); err != nil {
		return nil, fmt.Errorf("failed to parse GAN output: %w", err)
	}

	return result, nil
}

func (s *ganService) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "model": "conditional-gan-local"})
}

func main() {
	log := logger.New("gan-service")

	config.InitViper("gan-service")

	s := &ganService{
		log:          log,
		pythonPath:   config.GetEnv("PYTHON_PATH", "python"),
		ganScript:    config.GetEnv("GAN_SCRIPT_PATH", "cmd/ml_generator/gan_train.py"),
		ganModelPath: config.GetEnv("GAN_MODEL_PATH", "models/gan_generator.pt"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/gan/generate", s.generatePlanHandler)

	handler := middleware.RecoveryMiddleware(log.Logger)(mux)
	handler = middleware.RequestID(handler)

	port := config.GetEnv("GAN_SERVICE_PORT", "8004")
	metricsPort := config.GetEnv("GAN_SERVICE_METRICS_PORT", "9094")

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	cfg := server.DefaultConfig(port, metricsPort)
	server.Serve(log, cfg, metricsMux, mux, handler)
}

var errMethodNotSupported = "Метод не поддерживается"
