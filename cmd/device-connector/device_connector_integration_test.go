package main

import (
	"testing"
)

func TestDeviceConnectorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Skip("Device connector integration test requires running PostgreSQL, biometric-service, and device simulator")
}
