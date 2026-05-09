// cmd/device-emulator/main.go — Сервис эмуляции носимых устройств
//
// Этот сервис эмулирует работу носимых устройств (Apple Watch, Samsung Galaxy Watch,
// Huawei Watch D2, Amazfit T-Rex 3) и отправляет биометрические данные в device-connector.
//
// Использует адаптер-паттерн через BiometricSource интерфейс, что позволяет:
//   - Легко добавлять новые источники данных
//   - Swap между реальными API и моками для разработки
//   - Настраивать реалистичность (шум, пропуски, задержки)
//
// Запуск:
//   go run ./cmd/device-emulator \
//     --user-id=<USER_ID> \
//     --device-type=apple_watch \
//     --connector-url=http://localhost:8082 \
//     --sync-interval=30s \
//     --noise=0.05 \
//     --gap-prob=0.02

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MAMUER/project/internal/biometric/domain"
	"github.com/MAMUER/project/internal/biometric/mocks"
)

// IngestRecord matches device-connector expected format.
type IngestRecord struct {
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	Quality    string    `json:"quality"`
}

// IngestRequest is the body sent to device-connector.
type IngestRequest struct {
	DeviceType     string         `json:"device_type"`
	DeviceToken    string         `json:"device_token"`
	SyncIntervalMs int64          `json:"sync_interval_ms"`
	Records        []IngestRecord `json:"records"`
}

func main() {
	userID := flag.String("user-id", "", "User ID (required)")
	deviceType := flag.String("device-type", "apple_watch", "Device type: apple_watch, samsung_galaxy_watch, huawei_watch_d2, amazfit_trex3")
	connectorURL := flag.String("connector-url", "http://localhost:8082", "Device connector URL")
	syncInterval := flag.Duration("sync-interval", 30*time.Second, "Sync interval")
	autoRegister := flag.Bool("auto-register", true, "Auto-register device with connector")
	noiseLevel := flag.Float64("noise", 0.05, "Gaussian noise std dev (0.0-1.0)")
	gapProb := flag.Float64("gap-prob", 0.02, "Probability of missing data point (0.0-1.0)")
	flag.Parse()

	if *userID == "" {
		log.Fatal("user-id is required")
	}

	normalizedType := normalizeDeviceType(*deviceType)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var deviceID, deviceToken string
	if envID := os.Getenv("DEVICE_ID"); envID != "" {
		deviceID = envID
	}
	if envToken := os.Getenv("DEVICE_TOKEN"); envToken != "" {
		deviceToken = envToken
	}

	config := mocks.MockConfig{
		DeviceType:     normalizedType,
		NoiseLevel:     *noiseLevel,
		GapProbability: *gapProb,
		DelayMin:       50 * time.Millisecond,
		DelayMax:       300 * time.Millisecond,
		FailureRate:    0.005,
	}
	source := mocks.NewCustomMockBiometricSource(*userID, deviceID, config)

	if deviceID == "" || deviceToken == "" {
		if *autoRegister {
			deviceID, deviceToken = registerDevice(ctx, *userID, *deviceType, *connectorURL)
			log.Printf("Device registered: %s (%s)", deviceID, *deviceType)
			source = mocks.NewCustomMockBiometricSource(*userID, deviceID, config)
		} else {
			log.Fatal("device_id and device_token required when auto-register=false")
		}
	}

	log.Printf("Device emulator started: type=%s, user=%s, device=%s",
		*deviceType, *userID, deviceID)
	log.Printf("Sync interval: %s, noise=%.3f, gap_prob=%.3f",
		*syncInterval, *noiseLevel, *gapProb)

	ticker := time.NewTicker(*syncInterval)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	if err := syncData(ctx, source, deviceID, deviceToken, *deviceType, *connectorURL); err != nil {
		log.Printf("Warning: initial sync failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		case <-sigChan:
			log.Println("Signal received, exiting")
			cancel()
			return
		case <-ticker.C:
			if err := syncData(ctx, source, deviceID, deviceToken, *deviceType, *connectorURL); err != nil {
				log.Printf("Sync error: %v", err)
			}
		}
	}
}

func normalizeDeviceType(dt string) string {
	switch dt {
	case "apple_watch":
		return "apple"
	case "samsung_galaxy_watch":
		return "samsung"
	case "huawei_watch_d2":
		return "huawei"
	case "amazfit_trex3":
		return "amazfit"
	default:
		return dt
	}
}

func registerDevice(ctx context.Context, userID, deviceType, connectorURL string) (string, string) {
	url := connectorURL + "/api/v1/devices/register"
	payload := map[string]string{"device_type": deviceType, "user_id": userID}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Create reg request failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Device registration failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("Registration HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Decode reg response failed: %v", err)
	}
	id, _ := result["device_id"].(string)
	token, _ := result["device_token"].(string)
	if id == "" || token == "" {
		log.Fatal("Invalid registration response: missing device_id or device_token")
	}
	return id, token
}

func syncData(ctx context.Context, source domain.BiometricSource, deviceID, deviceToken, deviceType, connectorURL string) error {
	// Validate connector URL to prevent SSRF attacks
	if err := validateConnectorURL(connectorURL); err != nil {
		return fmt.Errorf("invalid connector URL: %w", err)
	}

	metrics := make([]string, 0, len(domain.AllMetricTypes()))
	for _, mt := range domain.AllMetricTypes() {
		if source.Supports(string(mt)) {
			metrics = append(metrics, string(mt))
		}
	}
	if len(metrics) == 0 {
		return nil
	}

	samples, err := source.Fetch(ctx, deviceID, metrics)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}
	if len(samples) == 0 {
		return nil
	}

	records := make([]IngestRecord, 0, len(samples))
	for _, s := range samples {
		records = append(records, IngestRecord{
			MetricType: s.MetricType, Value: s.Value, Timestamp: s.Timestamp, Quality: s.Quality,
		})
	}

	reqBody := IngestRequest{
		DeviceType: deviceType, DeviceToken: deviceToken, SyncIntervalMs: 30000, Records: records,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	u, err := url.Parse(connectorURL)
	if err != nil {
		return fmt.Errorf("parse connector URL failed: %w", err)
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/api/v1/devices/" + url.PathEscape(deviceID) + "/ingest"
	urlStr := u.String()
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}
	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}
	log.Printf("Sync OK: %d samples, forwarded=%v duplicates=%v failed=%v",
		len(samples), stats["forwarded"], stats["duplicates"], stats["failed"])
	return nil
}

// validateConnectorURL ensures the connector URL is safe to prevent SSRF attacks
func validateConnectorURL(url string) error {
	allowedHosts := []string{"localhost", "127.0.0.1", "::1"}
	if strings.HasPrefix(url, "http://") {
		host := strings.TrimPrefix(url, "http://")
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		for _, allowed := range allowedHosts {
			if host == allowed {
				return nil
			}
		}
		return fmt.Errorf("connector URL host not allowed: %s", host)
	}
	if strings.HasPrefix(url, "https://") {
		host := strings.TrimPrefix(url, "https://")
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		// In production, validate against a proper allowlist
		// For development, allow localhost and common development hosts
		for _, allowed := range allowedHosts {
			if host == allowed {
				return nil
			}
		}
		// Allow development domains
		if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".dev") {
			return nil
		}
		return fmt.Errorf("connector URL host not allowed: %s", host)
	}
	return fmt.Errorf("connector URL must use http or https scheme")
}
