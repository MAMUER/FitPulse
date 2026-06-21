package main

import (
	"testing"
)

func TestUserServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Skip("User service integration test requires running PostgreSQL, gRPC, SMTP server, and TOTP service")
}
