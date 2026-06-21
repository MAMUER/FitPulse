package main

import (
	"testing"
)

func TestDataProcessorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Skip("Data processor integration test requires running PostgreSQL and RabbitMQ")
}
