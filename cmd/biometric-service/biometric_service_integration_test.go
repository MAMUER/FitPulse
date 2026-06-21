package main

import (
	"testing"
)

func TestBiometricServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Skip("Biometric service integration test requires running PostgreSQL and RabbitMQ")
}
