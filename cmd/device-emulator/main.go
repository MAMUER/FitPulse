// cmd/device-emulator/main.go — Сервис эмуляции носимых устройств
//
// Этот сервис эмулирует работу носимых устройств (Apple Watch, Samsung Galaxy Watch,
// Huawei Watch D2, Amazfit T-Rex 3) и отправляет биометрические данные в device-connector.
//
// Запуск:
//   go run ./cmd/device-emulator \
//     --user-id=<USER_ID> \
//     --device-type=apple_watch \
//     --connector-url=http://localhost:8082 \
//     --sync-interval=30s

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MAMUER/Project/internal/wearableemulator"
)

func main() {
	// Parse command-line flags
	userID := flag.String("user-id", "", "User ID (required)")
	deviceType := flag.String("device-type", "apple_watch", "Device type: apple_watch, samsung_galaxy_watch, huawei_watch_d2, amazfit_trex3")
	connectorURL := flag.String("connector-url", "http://localhost:8082", "Device connector URL")
	syncInterval := flag.Duration("sync-interval", 30*time.Second, "Sync interval")
	autoRegister := flag.Bool("auto-register", true, "Auto-register device with connector")
	flag.Parse()

	if *userID == "" {
		log.Fatal("user-id is required")
	}

	validTypes := map[string]bool{
		"apple_watch": true, "samsung_galaxy_watch": true,
		"huawei_watch_d2": true, "amazfit_trex3": true,
	}
	if !validTypes[*deviceType] {
		log.Fatalf("invalid device type: %s (must be one of: apple_watch, samsung_galaxy_watch, huawei_watch_d2, amazfit_trex3)", *deviceType)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create physiological state
	state := wearableemulator.DefaultPhysiologicalState()

	// Create emulator
	emulator := wearableemulator.NewDeviceEmulator(
		"", // Will be set after registration
		*userID,
		"", // Will be set after registration
		wearableemulator.DeviceType(*deviceType),
		state,
	)

	// Auto-register device
	var deviceID, deviceToken string
	if deviceIDEnv := os.Getenv("DEVICE_ID"); deviceIDEnv != "" {
		if deviceTokenEnv := os.Getenv("DEVICE_TOKEN"); deviceTokenEnv != "" {
			deviceID = deviceIDEnv
			deviceToken = deviceTokenEnv
			emulator.DeviceID = deviceID
			emulator.DeviceToken = deviceToken
			log.Printf("Using device from environment: %s (%s)", deviceID, *deviceType)
		}
	} else if *autoRegister {
		regURL := *connectorURL + "/api/v1/devices/register"
		regReq := map[string]string{
			"device_type": *deviceType,
			"user_id":     *userID,
		}

		body, _ := json.Marshal(regReq)
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequestWithContext(ctx, "POST", regURL, bytes.NewReader(body))
		if err != nil {
			log.Fatalf("Failed to create registration request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Failed to register device: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var regResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
			log.Fatalf("Failed to decode registration response: %v", err)
		}

		deviceID = regResp["device_id"].(string)
		deviceToken = regResp["device_token"].(string)

		emulator.DeviceID = deviceID
		emulator.DeviceToken = deviceToken

		log.Printf("Device registered: %s (%s)", deviceID, *deviceType)
	} else {
		log.Fatal("auto-register is disabled, please register device manually")
	}

	// Set sync interval
	emulator.SyncInterval = *syncInterval

	// Start sync loop
	log.Printf("Starting device emulator: %s", *deviceType)
	log.Printf("Sync interval: %s", *syncInterval)
	log.Printf("Connector URL: %s", *connectorURL)

	ticker := time.NewTicker(*syncInterval)
	defer ticker.Stop()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run first sync immediately
	if err := syncData(ctx, emulator, *connectorURL); err != nil {
		log.Printf("Warning: first sync failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down device emulator...")
			return
		case <-sigChan:
			log.Println("Received shutdown signal")
			cancel()
		case <-ticker.C:
			if err := syncData(ctx, emulator, *connectorURL); err != nil {
				log.Printf("Sync failed: %v", err)
			}
		}
	}
}

// syncData sends biometric data to device connector
func syncData(ctx context.Context, emulator *wearableemulator.DeviceEmulator, connectorURL string) error {
	// Generate batch
	samples := emulator.GenerateBatch()
	if len(samples) == 0 {
		return nil
	}

	// Build ingest request
	records := make([]wearableemulator.BiometricSample, 0, len(samples))
	records = append(records, samples...)

	ingestReq := map[string]interface{}{
		"device_type":      emulator.DeviceType,
		"device_token":     emulator.DeviceToken,
		"sync_interval_ms": emulator.SyncInterval.Milliseconds(),
		"records":          records,
	}

	body, err := json.Marshal(ingestReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send to device connector
	ingestURL := fmt.Sprintf("%s/api/v1/devices/%s/ingest", connectorURL, emulator.DeviceID)

	req, err := http.NewRequestWithContext(ctx, "POST", ingestURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("Synced %d samples: %v forwarded, %v duplicates, %v failed",
		len(samples),
		stats["forwarded"],
		stats["duplicates"],
		stats["failed"],
	)

	return nil
}
