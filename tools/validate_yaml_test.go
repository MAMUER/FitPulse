package main

import "testing"

func TestTools(t *testing.T) {
	ok := isSafeYAMLPath("valid.yaml")
	if !ok {
		t.Error("expected true")
	}
	bad := isSafeYAMLPath("../evil.yaml")
	if bad {
		t.Error("expected false")
	}

	// exercise more code paths
	_, _ = readYAMLFile("nonexistent.yaml")
}
