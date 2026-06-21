package main

import (
	"testing"
)

func TestTrainingServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Skip("Training service integration test requires running PostgreSQL, gRPC, and RabbitMQ")
}
