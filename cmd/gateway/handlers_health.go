package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"
)

const healthCheckTimeout = 2 * time.Second

func (g *gateway) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	services := map[string]string{
		"user":      "down",
		"biometric": "down",
		"training":  "down",
	}

	// Check user service (critical)
	if g.userConn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
		healthClient := grpc_health_v1.NewHealthClient(g.userConn)
		resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		cancel()
		if err == nil && resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			services["user"] = "up"
		} else {
			status = "degraded"
		}
	} else if g.userClient != nil {
		services["user"] = "up"
	} else {
		status = "degraded"
	}

	// Check biometric service (optional)
	if g.biometricClient != nil || g.biometricAddr != "" {
		services["biometric"] = "up"
	}

	// Check training service (optional)
	if g.trainingClient != nil || g.trainingAddr != "" {
		services["training"] = "up"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        status,
		"service":       "gateway",
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"services":      services,
		"ml_classifier": g.mlClassifierURL,
		"ml_generator":  g.mlGeneratorURL,
	})
}
