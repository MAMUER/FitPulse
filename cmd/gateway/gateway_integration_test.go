package main

import (
	"testing"
)

func TestGatewayIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Skip("Gateway integration test requires all services running: user-service, biometric-service, training-service, classifier, Valkey, PostgreSQL, RabbitMQ")
}
