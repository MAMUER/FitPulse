package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// healthHandler returns service health status (degraded if optional services are down)
func (g *gateway) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	services := map[string]string{
		"user":      "up",
		"biometric": "up",
		"training":  "up",
	}

	if g.biometricClient == nil {
		services["biometric"] = "down"
		status = "degraded"
	}
	if g.trainingClient == nil {
		services["training"] = "down"
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
