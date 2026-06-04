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
	services := map[string]string{
		"user":      "down",
		"biometric": "down",
		"training":  "down",
	}

	// Check user service (critical) - uses gRPC health check if connection exists
	if g.userConn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
		healthClient := grpc_health_v1.NewHealthClient(g.userConn)
		resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		cancel()
		if err == nil && resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			services["user"] = "up"
		}
	} else if g.userClient != nil {
		// Fallback: client is set but no connection for health check (e.g., in tests)
		services["user"] = "up"
	}

	// Check biometric service (optional) - show as up only if connection was established
	if g.biometricClient != nil {
		services["biometric"] = "up"
	}

	// Check training service (optional) - show as up only if connection was established
	if g.trainingClient != nil {
		services["training"] = "up"
	}

	// Status is "ok" only if user service is healthy (biometric/training are optional)
	status := "ok"
	if services["user"] != "up" {
		status = "degraded"
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
