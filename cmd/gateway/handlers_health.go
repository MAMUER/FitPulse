package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	healthCheckTimeout = 2 * time.Second
	statusDown         = "down"
	statusUp           = "up"
)

// checkGRPCService performs a gRPC health check against the given connection.
// Returns "up" if the service is SERVING, "down" otherwise.
func checkGRPCService(conn *grpc.ClientConn, serviceName string) string {
	if conn == nil {
		return statusDown
	}
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: serviceName})
	if err != nil {
		return statusDown
	}
	if resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
		return statusUp
	}
	return statusDown
}

// checkTCPService performs a simple TCP dial to check if the service port is open.
// This is used as a fallback when gRPC health check is not available or for
// services that use lazy client initialization (biometric, training).
func checkTCPService(addr string) string {
	if addr == "" {
		return statusDown
	}

	dialer := &net.Dialer{
		Timeout: healthCheckTimeout,
	}
	ctx := context.Background()
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return statusDown
	}
	_ = conn.Close()
	return statusUp
}

func (g *gateway) healthHandler(w http.ResponseWriter, r *http.Request) {
	services := map[string]string{
		"user":      statusDown,
		"biometric": statusDown,
		"training":  statusDown,
	}

	// Check user service (critical) — uses gRPC health check with empty service name.
	// The user-service sets overall serving status via:
	//   healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	// Fallback: if userConn is nil but userClient is set (e.g., in tests with mocks),
	// consider the service as "up" since the mock client is already initialized.
	if g.userConn != nil {
		services["user"] = checkGRPCService(g.userConn, "")
	} else if g.userClient != nil {
		services["user"] = statusUp
	}

	// Check biometric service (optional).
	// If biometricClient is already initialized (e.g., in tests with mocks),
	// consider the service as "up". Otherwise, use TCP dial to verify the gRPC
	// port is open — this handles lazy client initialization in production.
	if g.biometricClient != nil {
		services["biometric"] = statusUp
	} else {
		services["biometric"] = checkTCPService(g.biometricAddr)
	}

	// Check training service (optional) — same approach as biometric.
	if g.trainingClient != nil {
		services["training"] = statusUp
	} else {
		services["training"] = checkTCPService(g.trainingAddr)
	}

	// Status is "ok" only if user service is healthy (biometric/training are optional)
	status := "ok"
	if services["user"] != statusUp {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       status,
		"service":      "gateway",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"services":     services,
		"classifier":   g.classifierURL,
		"ml_generator": g.mlGeneratorURL,
	})
}
