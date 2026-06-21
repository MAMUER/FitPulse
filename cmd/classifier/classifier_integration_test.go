package main

import (
	"testing"
)

func TestClassifierIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Skip("Classifier integration test requires running HTTP client and classifier service")
}
