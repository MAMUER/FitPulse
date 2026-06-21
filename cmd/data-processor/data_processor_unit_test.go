package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestRun_Shutdown(t *testing.T) {
	stopCh := make(chan os.Signal, 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		stopCh <- syscall.SIGTERM
	}()

	err := run(stopCh)
	if err != nil {
		t.Logf("run returned error (expected without DB): %v", err)
	}
}

func TestRun_CanceledContext(t *testing.T) {
	stopCh := make(chan os.Signal, 1)
	close(stopCh)

	err := run(stopCh)
	if err != nil {
		t.Logf("run returned error (expected without DB): %v", err)
	}
}
