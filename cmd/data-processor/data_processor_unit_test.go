package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/MAMUER/project/internal/logger"
)

func TestRun_Shutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	log := logger.New("data-processor-test")
	err := run(ctx, log)
	if err != nil {
		t.Logf("run returned error (expected without DB): %v", err)
	}
}

func TestRun_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	log := logger.New("data-processor-test")
	err := run(ctx, log)
	if err != nil {
		t.Logf("run returned error (expected without DB): %v", err)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
