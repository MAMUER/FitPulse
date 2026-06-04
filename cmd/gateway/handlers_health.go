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
		"user":      "up",
		"biometric": "up",
		"training":  "up",
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	if g.userConn != nil {
		healthClient := grpc_health_v1.NewHealthClient(g.userConn)
		resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err != nil || resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
			services["user"] = "down"
			status = "degraded"
		}
	} else {
		services["user"] = "down"
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
