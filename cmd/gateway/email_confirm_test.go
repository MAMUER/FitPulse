package main

import (
	"testing"
)

func TestEmailConfirmIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Skip("Email confirm integration test requires web templates and SMTP server")
}
